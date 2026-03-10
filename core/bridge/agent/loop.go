package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type Completer interface {
	Complete(context.Context, llm.CompletionRequest) (*llm.CompletionResponse, error)
}

type ToolCatalog = tools.ToolCatalog

// Agent 只编排对话循环，不绑定具体 Provider/Tool 实现。
type Agent struct {
	completer Completer
	tools     ToolCatalog
	history   *History
	maxTurns  int

	initialHistoryLen int
	lastTurn          int
}

const maxConsecutiveNonExecutableToolCallTurns = 3

// ErrAwaitingHuman 表示 ask_human 已发起问题，当前回合需要等待用户输入。
type ErrAwaitingHuman struct {
	QuestionID    string
	Prompt        string
	SelectionMode string
	Options       []tools.AskHumanOption
}

func (e *ErrAwaitingHuman) Error() string {
	return fmt.Sprintf("awaiting human input: question_id=%s", strings.TrimSpace(e.QuestionID))
}

func NewAgent(completer Completer, toolCatalog ToolCatalog, systemPrompt string, maxTurns int) *Agent {
	return NewAgentWithHistory(completer, toolCatalog, NewHistory(systemPrompt), maxTurns)
}

// NewAgentWithHistory 使用预加载历史初始化 Agent，适用于跨请求会话。
func NewAgentWithHistory(completer Completer, toolCatalog ToolCatalog, history *History, maxTurns int) *Agent {
	if maxTurns <= 0 {
		maxTurns = 20
	}
	if history == nil {
		history = NewHistory("")
	}

	return &Agent{
		completer:         completer,
		tools:             toolCatalog,
		history:           history,
		maxTurns:          maxTurns,
		initialHistoryLen: history.Len(),
	}
}

// Run 负责循环与退出条件；单步执行下沉给独立协作者处理。
func (a *Agent) Run(ctx context.Context, userMessage string) (string, error) {
	return a.runWithSink(ctx, userMessage, "", nil)
}

// RunWithTraceID 允许调用方注入请求级 trace_id，保障跨层链路追踪一致。
func (a *Agent) RunWithTraceID(ctx context.Context, userMessage string, traceID string) (string, error) {
	return a.runWithSink(ctx, userMessage, traceID, nil)
}

func (a *Agent) RunStream(ctx context.Context, userMessage string, sink EventSink) (string, error) {
	return a.runWithSink(ctx, userMessage, "", sink)
}

func (a *Agent) RunStreamWithTraceID(ctx context.Context, userMessage string, traceID string, sink EventSink) (string, error) {
	return a.runWithSink(ctx, userMessage, traceID, sink)
}

func (a *Agent) runWithSink(ctx context.Context, userMessage string, traceID string, sink EventSink) (string, error) {
	traceID = strings.TrimSpace(traceID)
	if traceID == "" {
		traceID = fmt.Sprintf("agent-%d", time.Now().UnixNano())
	}

	events := newAgentEventEmitter(sink)
	turnHistory := a.history.Clone()
	completion := newCompletionRunner(a.completer, a.tools, turnHistory)
	toolCalls := newToolCallExecutor(a.tools, turnHistory, nil, events)

	appendUserMessage(turnHistory, userMessage)
	consecutiveNonExecutableToolCallTurns := 0
	a.lastTurn = 0

	for turn := 0; turn < a.maxTurns; turn++ {
		a.lastTurn = turn
		msg, finishReason, err := completion.complete(ctx, sink, traceID, turn)
		if err != nil {
			runErr := fmt.Errorf("trace_id=%s turn=%d complete_once: %w", traceID, turn, err)
			return "", events.terminalError(ctx, traceID, turn, AssistantStepID(turn), runErr)
		}

		switch finishReason {
		case llm.FinishStop:
			a.commitTurn(turnHistory)
			return a.handleAssistantStop(msg), nil
		case llm.FinishToolCalls:
			stats, err := toolCalls.execute(ctx, traceID, turn, msg.ToolCalls)
			if err != nil {
				if isEventEmitError(err) {
					return "", err
				}
				var awaitingErr *ErrAwaitingHuman
				if errors.As(err, &awaitingErr) {
					a.commitTurn(turnHistory)
					return "", err
				}
				runErr := fmt.Errorf("trace_id=%s turn=%d handle_tool_calls: %w", traceID, turn, err)
				return "", events.terminalError(ctx, traceID, turn, AssistantStepID(turn), runErr)
			}
			if stats.nonExecutable() {
				consecutiveNonExecutableToolCallTurns++
			} else {
				consecutiveNonExecutableToolCallTurns = 0
			}
			// 防止模型反复生成不可执行的 tool_call（空参数/缺失工具）导致无效循环。
			if consecutiveNonExecutableToolCallTurns >= maxConsecutiveNonExecutableToolCallTurns {
				runErr := fmt.Errorf("trace_id=%s turn=%d repeated non-executable tool_calls; aborting tool-call loop", traceID, turn)
				return "", events.terminalError(ctx, traceID, turn, AssistantStepID(turn), runErr)
			}
		case llm.FinishLength:
			content := strings.TrimSpace(msg.Text)
			if content != "" {
				a.commitTurn(turnHistory)
				return content, nil
			}
			runErr := fmt.Errorf("trace_id=%s turn=%d finish_reason=%q with empty content", traceID, turn, finishReason)
			return "", events.terminalError(ctx, traceID, turn, AssistantStepID(turn), runErr)
		default:
			runErr := fmt.Errorf("trace_id=%s turn=%d unsupported finish_reason: %q", traceID, turn, finishReason)
			return "", events.terminalError(ctx, traceID, turn, AssistantStepID(turn), runErr)
		}
	}

	if a.maxTurns > 0 {
		a.lastTurn = a.maxTurns - 1
	}
	runErr := fmt.Errorf("trace_id=%s max turns exceeded: %d", traceID, a.maxTurns)
	return "", events.terminalError(ctx, traceID, a.lastTurn, AssistantStepID(a.lastTurn), runErr)
}

func (a *Agent) commitTurn(history *History) {
	if a == nil || history == nil {
		return
	}
	a.history = history
}

// appendUserMessage 只负责把用户输入追加到会话历史。
func appendUserMessage(history *History, userMessage string) {
	if history == nil {
		return
	}

	trimmed := strings.TrimSpace(userMessage)
	if trimmed == "" {
		return
	}

	history.Append(llm.Message{
		Role: llm.RoleUser,
		Text: trimmed,
	})
}

func (a *Agent) handleAssistantStop(msg llm.Message) string {
	return msg.Text
}

// GetNewMessages 返回 Agent 初始化后已提交的新增会话消息。
func (a *Agent) GetNewMessages() []llm.Message {
	if a == nil || a.history == nil {
		return nil
	}

	messages := a.history.Messages()
	if a.initialHistoryLen <= 0 {
		return messages
	}
	if a.initialHistoryLen >= len(messages) {
		return nil
	}

	return llm.CloneMessages(messages[a.initialHistoryLen:])
}

func (a *Agent) LastTurn() int {
	if a == nil {
		return 0
	}
	return a.lastTurn
}

func (a *Agent) GetConversationState() llm.ConversationState {
	if a == nil || a.history == nil {
		return llm.ConversationState{}
	}
	return a.history.ConversationState()
}
