package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	bridgeTasks "ghost-os/bridge/tasks"
)

const (
	RunTranscriptEventMarker = "[TASK_RUN_EVENT]"

	runStatusError        = bridgeTasks.RunStatusError
	runStatusSuccess      = bridgeTasks.RunStatusSuccess
	kindWorkflow          = bridgeTasks.KindWorkflow
	kindOrchestration     = bridgeTasks.KindOrchestration
	workflowNodeStart     = "start"
	workflowNodeTool      = "tool"
	workflowNodeLLM       = "llm"
	workflowNodeAgent     = "agent"
	workflowNodeIf        = "if"
	workflowNodeLoop      = "loop"
	workflowNodeEnd       = "end"
	sessionReadFailPrefix = "执行会话读取失败："
)

type SessionSource struct {
	Messages []llm.Message
	Err      error
}

type RunTranscriptOptions struct {
	Task     bridgeTasks.ScheduledTask
	TraceID  string
	Result   bridgeTasks.ExecutionResult
	Sessions map[string]SessionSource
}

type RunTranscript struct {
	Title    string
	Messages []llm.Message
}

type runTranscriptBuilder struct {
	options      RunTranscriptOptions
	messages     []llm.Message
	sessionOffsets map[string]int
}

type runToolMessage struct {
	sender    string
	toolName  string
	arguments any
	output    string
	status    string
}

func ShouldCreateRunTranscript(taskKind string) bool {
	switch bridgeTasks.NormalizeKind(taskKind) {
	case kindWorkflow, kindOrchestration:
		return true
	default:
		return false
	}
}

func BuildRunTranscriptMessages(options RunTranscriptOptions) RunTranscript {
	builder := newRunTranscriptBuilder(options)
	builder.build()
	return RunTranscript{
		Title:    builder.title(),
		Messages: llm.CloneMessages(builder.messages),
	}
}

func newRunTranscriptBuilder(options RunTranscriptOptions) *runTranscriptBuilder {
	return &runTranscriptBuilder{
		options:        normalizeRunTranscriptOptions(options),
		messages:       make([]llm.Message, 0, len(options.Result.NodeResults)+2),
		sessionOffsets: make(map[string]int),
	}
}

func normalizeRunTranscriptOptions(options RunTranscriptOptions) RunTranscriptOptions {
	options.TraceID = strings.TrimSpace(options.TraceID)
	if options.Sessions == nil {
		options.Sessions = map[string]SessionSource{}
	}
	return options
}

func (b *runTranscriptBuilder) build() {
	b.addEvent(b.startEvent())
	switch bridgeTasks.NormalizeKind(b.options.Task.TaskKind) {
	case kindWorkflow:
		b.appendWorkflowResults()
	case kindOrchestration:
		b.appendOrchestrationResults()
	}
	b.addEvent(b.finishEvent())
}

func (b *runTranscriptBuilder) title() string {
	name := strings.TrimSpace(b.options.Task.Name)
	if name == "" {
		name = strings.TrimSpace(b.options.Task.ID)
	}
	kind := bridgeTasks.NormalizeKind(b.options.Task.TaskKind)
	if name == "" {
		return kind + " run"
	}
	return kind + ": " + name
}

func (b *runTranscriptBuilder) startEvent() string {
	return "任务运行开始：" + b.title()
}

func (b *runTranscriptBuilder) finishEvent() string {
	status := strings.TrimSpace(b.options.Result.Status)
	if status == "" {
		status = runStatusError
	}
	if errText := strings.TrimSpace(b.options.Result.Error); errText != "" {
		return "任务运行结束：" + status + "\nerror: " + errText
	}
	return "任务运行结束：" + status
}

func (b *runTranscriptBuilder) addEvent(text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	b.messages = append(b.messages, llm.Message{
		Role: llm.RoleInternal,
		Text: RunTranscriptEventMarker + "\n" + trimmed,
	})
}

func (b *runTranscriptBuilder) addSpeakerMessage(sender string, content string) {
	body := strings.TrimSpace(content)
	if body == "" {
		body = "(empty)"
	}
	b.messages = append(b.messages, llm.Message{
		Role: llm.RoleAssistant,
		Text: strings.TrimSpace(sender) + "\n\n" + body,
	})
}

func (b *runTranscriptBuilder) addToolMessage(message runToolMessage) {
	callID := runToolCallID(message.sender, message.toolName, len(b.messages)+1)
	b.messages = append(b.messages, llm.Message{
		Role: llm.RoleAssistant,
		Text: strings.TrimSpace(message.sender) + "\n\n使用工具：" + strings.TrimSpace(message.toolName),
		ToolCalls: []llm.ToolCall{{
			ID:        callID,
			Name:      strings.TrimSpace(message.toolName),
			Arguments: transcriptJSON(message.arguments),
		}},
	})
	b.messages = append(b.messages, llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: callID,
		Text:       agent.FormatToolResult(message.toolName, b.options.TraceID, message.output, runToolError(message.status)),
	})
}

func transcriptJSON(value any) json.RawMessage {
	if value == nil {
		return json.RawMessage(`{}`)
	}
	encoded, err := json.Marshal(value)
	if err != nil || len(encoded) == 0 {
		return json.RawMessage(`{}`)
	}
	return encoded
}

func runToolError(status string) error {
	if strings.TrimSpace(status) == runStatusError {
		return errors.New("tool execution failed")
	}
	return nil
}

func runToolCallID(sender string, toolName string, index int) string {
	base := strings.NewReplacer(" ", "-", "\n", "-", "/", "-").Replace(strings.TrimSpace(sender))
	tool := strings.TrimSpace(toolName)
	if base == "" {
		base = "task-run"
	}
	return fmt.Sprintf("%s:%s:%d", base, tool, index)
}
