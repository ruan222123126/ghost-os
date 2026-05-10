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
	seenSessions map[string]struct{}
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
		options:      normalizeRunTranscriptOptions(options),
		messages:     make([]llm.Message, 0, len(options.Result.NodeResults)+2),
		seenSessions: make(map[string]struct{}),
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
	lines := []string{"任务运行开始：" + b.title()}
	if b.options.TraceID != "" {
		lines = append(lines, "trace_id: "+b.options.TraceID)
	}
	return strings.Join(lines, "\n")
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

func (b *runTranscriptBuilder) appendAgentSessionMessages(sessionID string, sender string) bool {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return false
	}
	if _, exists := b.seenSessions[id]; exists {
		return true
	}
	b.seenSessions[id] = struct{}{}
	source, ok := b.options.Sessions[id]
	if !ok || source.Err != nil {
		return b.addSessionReadFailure(id, source.Err)
	}
	return b.appendSessionMessages(sender, source.Messages)
}

func (b *runTranscriptBuilder) addSessionReadFailure(sessionID string, cause error) bool {
	if cause == nil {
		cause = errors.New("session messages are not available")
	}
	b.addEvent(sessionReadFailPrefix + sessionID + "\nerror: " + cause.Error())
	return false
}

func (b *runTranscriptBuilder) appendSessionMessages(sender string, messages []llm.Message) bool {
	appended := false
	for _, message := range messages {
		if b.appendAgentSessionMessage(sender, message) {
			appended = true
		}
	}
	return appended
}

func (b *runTranscriptBuilder) appendAgentSessionMessage(sender string, message llm.Message) bool {
	switch message.Role {
	case llm.RoleAssistant:
		return b.appendAssistantSessionMessage(sender, message)
	case llm.RoleTool:
		b.messages = append(b.messages, message)
		return true
	default:
		return false
	}
}

func (b *runTranscriptBuilder) appendAssistantSessionMessage(sender string, message llm.Message) bool {
	clonedMessages := llm.CloneMessages([]llm.Message{message})
	if len(clonedMessages) == 0 {
		return false
	}
	cloned := prefixAssistantSessionMessage(sender, clonedMessages[0])
	b.messages = append(b.messages, cloned)
	return true
}

func prefixAssistantSessionMessage(sender string, message llm.Message) llm.Message {
	prefix := strings.TrimSpace(sender)
	if prefix != "" && strings.TrimSpace(message.Text) != "" {
		message.Text = prefix + "\n\n" + strings.TrimSpace(message.Text)
	}
	if prefix != "" && strings.TrimSpace(message.Text) == "" && len(message.ToolCalls) > 0 {
		message.Text = prefix + "\n\n使用工具"
	}
	return message
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
