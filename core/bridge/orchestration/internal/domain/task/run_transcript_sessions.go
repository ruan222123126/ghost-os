package task

import (
	"errors"
	"strings"

	"ghost-os/bridge/llm"
	bridgeTasks "ghost-os/bridge/tasks"
)

func CollectRunTranscriptSessionIDs(result bridgeTasks.ExecutionResult) []string {
	collector := sessionIDCollector{seen: map[string]struct{}{}}
	for _, node := range result.NodeResults {
		collector.collectNode(node)
	}
	return collector.ids
}

type sessionIDCollector struct {
	ids  []string
	seen map[string]struct{}
}

func (c *sessionIDCollector) collectNode(node bridgeTasks.RunNodeResult) {
	if node.NodeType == workflowNodeAgent {
		c.add(transcriptString(transcriptRecord(node.Output), "session_id_output"))
		return
	}
	output := transcriptRecord(node.Output)
	c.collectMembers(transcriptSlice(output, "member_results"))
	for _, item := range transcriptSlice(output, "dispatch_results") {
		c.collectMembers(transcriptSlice(transcriptRecord(item), "member_results"))
	}
}

func (c *sessionIDCollector) collectMembers(items []any) {
	for _, item := range items {
		c.add(transcriptString(transcriptRecord(item), "session_id"))
	}
}

func (c *sessionIDCollector) add(sessionID string) {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return
	}
	if _, exists := c.seen[trimmed]; exists {
		return
	}
	c.seen[trimmed] = struct{}{}
	c.ids = append(c.ids, trimmed)
}

func (b *runTranscriptBuilder) appendAgentSessionMessages(sessionID string, sender string) bool {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return false
	}
	source, ok := b.options.Sessions[id]
	if !ok || source.Err != nil {
		return b.addSessionReadFailure(id, source.Err)
	}
	return b.appendSessionMessages(sender, b.nextSessionTurnMessages(id, source.Messages))
}

func (b *runTranscriptBuilder) nextSessionTurnMessages(
	sessionID string,
	messages []llm.Message,
) []llm.Message {
	if len(messages) == 0 {
		return nil
	}
	start := normalizeSessionOffset(b.sessionOffsets[sessionID], len(messages))
	from, to := nextSessionTurnSpan(messages, start)
	b.sessionOffsets[sessionID] = to
	if from >= to {
		return nil
	}
	return llm.CloneMessages(messages[from:to])
}

func normalizeSessionOffset(offset int, size int) int {
	if offset <= 0 {
		return 0
	}
	if offset >= size {
		return size
	}
	return offset
}

func nextSessionTurnSpan(messages []llm.Message, start int) (int, int) {
	if start >= len(messages) {
		return len(messages), len(messages)
	}
	turnStart := nextSessionTurnStart(messages, start)
	return turnStart, nextSessionTurnEnd(messages, turnStart)
}

func nextSessionTurnStart(messages []llm.Message, start int) int {
	for index := start; index < len(messages); index++ {
		if messages[index].Role == llm.RoleUser {
			return index
		}
	}
	return start
}

func nextSessionTurnEnd(messages []llm.Message, start int) int {
	end := start + 1
	for end < len(messages) && messages[end].Role != llm.RoleUser {
		end++
	}
	return end
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
