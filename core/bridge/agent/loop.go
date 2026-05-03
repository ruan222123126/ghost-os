package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

type Completer interface {
	Complete(context.Context, llm.CompletionRequest) (*llm.CompletionResponse, error)
}

type ToolCatalog = tools.ToolCatalog

// Agent 只编排对话循环，不绑定具体 Provider/Tool 实现。
// Agent 非并发安全，调用方需自行串行化或加锁保护。
type Agent struct {
	completer              Completer
	tools                  ToolCatalog
	responseOptions        llm.ResponseOptions
	assistantTextHandlers  []AssistantTextHandler
	beforeCompletion       BeforeCompletionHook
	strictToolCallProtocol bool
	history                *History
	maxTurns               int
	completionRetryPolicy  *CompletionRetryPolicy

	initialHistoryLen int
	lastTurn          int
	streamLifecycle   StreamLifecyclePayloadBuilder
}

const maxConsecutiveNonExecutableToolCallTurns = 3

type BeforeCompletionHook func(context.Context, int, *History) error

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

// ErrIterationHandoff tells the outer orchestrator to end this fresh-memory agent run
// and continue with the next iteration or finish the pro run.
type ErrIterationHandoff struct {
	Did            string
	Remaining      string
	Completed      bool
	FinalMessage   string
	FinalChangeLog string
}

func (e *ErrIterationHandoff) Error() string {
	if e == nil {
		return "iteration handoff"
	}
	if e.Completed {
		return "iteration completed"
	}
	return "iteration handoff"
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

func (a *Agent) SetStreamLifecyclePayloadBuilder(builder StreamLifecyclePayloadBuilder) {
	if a == nil {
		return
	}
	a.streamLifecycle = builder
}

func (a *Agent) AddAssistantTextHandler(handler AssistantTextHandler) {
	if a == nil {
		return
	}
	if handler == nil {
		return
	}
	a.assistantTextHandlers = append(a.assistantTextHandlers, handler)
}

func (a *Agent) SetBeforeCompletionHook(hook BeforeCompletionHook) {
	if a == nil {
		return
	}
	a.beforeCompletion = hook
}

func (a *Agent) SetStrictToolCallProtocol(strict bool) {
	if a == nil {
		return
	}
	a.strictToolCallProtocol = strict
}

func (a *Agent) SetResponseOptions(options llm.ResponseOptions) {
	if a == nil {
		return
	}
	a.responseOptions = llm.CloneResponseOptions(options)
}

func (a *Agent) SetCompletionRetryPolicy(policy CompletionRetryPolicy) {
	if a == nil {
		return
	}
	cloned := policy
	a.completionRetryPolicy = &cloned
}

// Run 负责循环与退出条件；单步执行下沉给独立协作者处理。
func (a *Agent) Run(ctx context.Context, userMessage string) (string, error) {
	return a.RunMessage(ctx, llm.Message{
		Role: llm.RoleUser,
		Text: userMessage,
	})
}

// RunWithTraceID 允许调用方注入请求级 trace_id，保障跨层链路追踪一致。
func (a *Agent) RunWithTraceID(ctx context.Context, userMessage string, traceID string) (string, error) {
	return a.RunMessageWithTraceID(ctx, llm.Message{
		Role: llm.RoleUser,
		Text: userMessage,
	}, traceID)
}

func (a *Agent) RunMessage(ctx context.Context, userInput llm.Message) (string, error) {
	return a.runWithSink(ctx, userInput, "", nil)
}

func (a *Agent) RunMessageWithTraceID(ctx context.Context, userInput llm.Message, traceID string) (string, error) {
	return a.runWithSink(ctx, userInput, traceID, nil)
}

func (a *Agent) RunMessageStreamWithTraceID(ctx context.Context, userInput llm.Message, traceID string, sink streaming.Sink) (string, error) {
	return a.runWithSink(ctx, userInput, traceID, sink)
}

func (a *Agent) runWithSink(ctx context.Context, userInput llm.Message, traceID string, sink streaming.Sink) (string, error) {
	state, err := newAgentRunState(a, sink, traceID)
	if err != nil {
		runErr := fmt.Errorf("initialize agent runtime: %w", err)
		if sink != nil {
			if emitErr := state.terminalRunError(ctx, 0, runErr); emitErr != nil {
				if errors.Is(emitErr, runErr) {
					return "", runErr
				}
				return "", errors.Join(runErr, fmt.Errorf("emit runtime initialization error event: %w", emitErr))
			}
		}
		return "", runErr
	}
	if err := state.events.runStarted(ctx, state.traceID, state.lifecycle); err != nil {
		return "", err
	}

	appendUserMessage(state.history, userInput)
	a.lastTurn = 0

	for turn := 0; turn < a.maxTurns; turn++ {
		outcome, err := a.runTurn(ctx, turn, &state)
		if err != nil {
			return "", err
		}
		if outcome.done {
			return outcome.output, nil
		}
	}

	if a.maxTurns > 0 {
		a.lastTurn = a.maxTurns - 1
	}
	return "", state.maxTurnsExceeded(ctx, a.lastTurn, a.maxTurns)
}

// GetNewMessages 返回 Agent 初始化以来（或上次 ResetNewMessages 以来）已提交的新增会话消息。
// 注意：该方法不会“消费”消息；如需按增量消费，请在处理后调用 ResetNewMessages。
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

// ResetNewMessages 将“新增消息”的基准推进到当前历史末尾，用于按增量消费 GetNewMessages。
func (a *Agent) ResetNewMessages() {
	if a == nil || a.history == nil {
		return
	}
	a.initialHistoryLen = a.history.Len()
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
