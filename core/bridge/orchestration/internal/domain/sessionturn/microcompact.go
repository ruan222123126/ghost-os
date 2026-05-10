package sessionturn

import (
	"fmt"
	"log"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

const (
	microcompactKeepRecentSpans = 2
	microcompactNewlineJoin     = "\n\n"
)

type microcompactProjector struct {
	traceID string
}

type microcompactSpan struct {
	start        int
	end          int
	summary      llm.Message
	compressible bool
}

type microcompactToolPair struct {
	call       llm.ToolCall
	tool       llm.Message
	envelope   agent.ToolResultEnvelope
	toolName   string
	toolCallID string
}

type microcompactSkip struct {
	reason string
	tool   string
	callID string
}

func (s microcompactSkip) Error() string {
	return s.reason
}

func newMicrocompactProjector(options ProjectionOptions) microcompactProjector {
	return microcompactProjector{traceID: strings.TrimSpace(options.TraceID)}
}

func (p microcompactProjector) Project(messages []llm.Message) []llm.Message {
	spans := p.collectSpans(messages)
	protected := protectedMicrocompactSpans(spans, microcompactKeepRecentSpans)
	projected, compacted := p.applyProjection(messages, spans, protected)
	p.logProjection(messages, projected, compacted)
	return llm.CloneMessages(projected)
}

func (p microcompactProjector) collectSpans(messages []llm.Message) []microcompactSpan {
	spans := make([]microcompactSpan, 0, len(messages))
	for index := 0; index < len(messages); {
		end := microcompactSpanEnd(messages, index)
		span := microcompactSpan{start: index, end: end}
		if summary, ok := p.trySummarizeSpan(messages, index, end); ok {
			span.summary = summary
			span.compressible = true
		}
		spans = append(spans, span)
		index = end
	}
	return spans
}

func microcompactSpanEnd(messages []llm.Message, start int) int {
	end := start + 1
	if start >= len(messages) || messages[start].Role != llm.RoleAssistant || len(messages[start].ToolCalls) == 0 {
		return end
	}
	for end < len(messages) && messages[end].Role == llm.RoleTool {
		end++
	}
	return end
}

func (p microcompactProjector) trySummarizeSpan(messages []llm.Message, start int, end int) (llm.Message, bool) {
	if turnContainsReasoningReplay(messages, start) {
		return llm.Message{}, false
	}
	span := messages[start:end]
	assistant := spanAssistantMessage(span)
	if assistant == nil || !allMicrocompactTargetTools(assistant.ToolCalls) {
		return llm.Message{}, false
	}
	pairs, err := buildMicrocompactToolPairs(*assistant, span[1:])
	if err != nil {
		p.logSkip(err)
		return llm.Message{}, false
	}
	text, err := p.summarizePairs(pairs)
	if err != nil {
		p.logSkip(err)
		return llm.Message{}, false
	}
	return llm.Message{Role: llm.RoleAssistant, Text: text}, true
}

func turnContainsReasoningReplay(messages []llm.Message, index int) bool {
	start := turnStartIndex(messages, index)
	end := turnEndIndex(messages, index)
	for _, message := range messages[start:end] {
		if message.Role != llm.RoleAssistant || len(message.ToolCalls) == 0 {
			continue
		}
		if hasReasoningReplayContent(message.ReasoningContent) {
			return true
		}
	}
	return false
}

func turnStartIndex(messages []llm.Message, index int) int {
	for current := index; current >= 0; current-- {
		if messages[current].Role == llm.RoleUser {
			return current
		}
	}
	return 0
}

func turnEndIndex(messages []llm.Message, index int) int {
	end := index + 1
	for end < len(messages) && messages[end].Role != llm.RoleUser {
		end++
	}
	return end
}

func hasReasoningReplayContent(raw []byte) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed != "" && trimmed != "null"
}

func spanAssistantMessage(span []llm.Message) *llm.Message {
	if len(span) == 0 || span[0].Role != llm.RoleAssistant || len(span[0].ToolCalls) == 0 {
		return nil
	}
	return &span[0]
}

func allMicrocompactTargetTools(calls []llm.ToolCall) bool {
	if len(calls) == 0 {
		return false
	}
	for _, call := range calls {
		if !isMicrocompactTargetTool(call.Name) {
			return false
		}
	}
	return true
}

func buildMicrocompactToolPairs(
	assistant llm.Message,
	toolMessages []llm.Message,
) ([]microcompactToolPair, error) {
	if len(toolMessages) != len(assistant.ToolCalls) {
		return nil, microcompactUnsupported("", "", "tool result count mismatch")
	}
	toolByCallID := make(map[string]llm.ToolCall, len(assistant.ToolCalls))
	for _, call := range assistant.ToolCalls {
		callID := strings.TrimSpace(call.ID)
		toolByCallID[callID] = call
	}
	seen := make(map[string]bool, len(toolMessages))
	pairs := make([]microcompactToolPair, 0, len(toolMessages))
	for _, toolMessage := range toolMessages {
		pair, err := newMicrocompactToolPair(toolByCallID, toolMessage)
		if err != nil {
			return nil, err
		}
		if seen[pair.toolCallID] {
			return nil, microcompactUnsupported(pair.toolName, pair.toolCallID, "duplicate tool result")
		}
		seen[pair.toolCallID] = true
		pairs = append(pairs, pair)
	}
	if len(seen) != len(assistant.ToolCalls) {
		return nil, microcompactUnsupported("", "", "missing tool result")
	}
	return pairs, nil
}

func newMicrocompactToolPair(
	toolByCallID map[string]llm.ToolCall,
	toolMessage llm.Message,
) (microcompactToolPair, error) {
	callID := strings.TrimSpace(toolMessage.ToolCallID)
	call, ok := toolByCallID[callID]
	if !ok {
		return microcompactToolPair{}, microcompactUnsupported("", callID, "unknown tool_call_id")
	}
	envelope, ok := agent.ParseToolResultEnvelope(toolMessage.Text)
	if !ok {
		return microcompactToolPair{}, microcompactParseFailed(call.Name, callID)
	}
	toolName := strings.TrimSpace(call.Name)
	if !strings.EqualFold(strings.TrimSpace(envelope.Tool), toolName) {
		return microcompactToolPair{}, microcompactUnsupported(toolName, callID, "tool name mismatch")
	}
	return microcompactToolPair{
		call:       call,
		tool:       toolMessage,
		envelope:   envelope,
		toolName:   toolName,
		toolCallID: callID,
	}, nil
}

func (p microcompactProjector) summarizePairs(pairs []microcompactToolPair) (string, error) {
	parts := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		text, err := summarizeMicrocompactPair(pair)
		if err != nil {
			return "", err
		}
		parts = append(parts, text)
	}
	text := strings.TrimSpace(strings.Join(parts, microcompactNewlineJoin))
	if text == "" {
		return "", microcompactUnsupported("", "", "empty summary")
	}
	return text, nil
}

func protectedMicrocompactSpans(spans []microcompactSpan, keepRecent int) map[int]bool {
	indexes := make([]int, 0, len(spans))
	for index, span := range spans {
		if span.compressible {
			indexes = append(indexes, index)
		}
	}
	protected := make(map[int]bool, keepRecent)
	start := len(indexes) - keepRecent
	if start < 0 {
		start = 0
	}
	for _, index := range indexes[start:] {
		protected[index] = true
	}
	return protected
}

func (p microcompactProjector) applyProjection(
	messages []llm.Message,
	spans []microcompactSpan,
	protected map[int]bool,
) ([]llm.Message, int) {
	projected := make([]llm.Message, 0, len(messages))
	compacted := 0
	for index, span := range spans {
		if span.compressible && !protected[index] {
			projected = append(projected, span.summary)
			compacted++
			continue
		}
		projected = append(projected, messages[span.start:span.end]...)
	}
	return projected, compacted
}

func (p microcompactProjector) logProjection(
	original []llm.Message,
	projected []llm.Message,
	compacted int,
) {
	log.Printf(
		"trace_id=%s microcompact spans_compacted=%d estimated_tokens_before=%d estimated_tokens_after=%d",
		p.traceID,
		compacted,
		EstimateMessagesTokens(original),
		EstimateMessagesTokens(projected),
	)
}

func EstimateMessagesTokens(messages []llm.Message) int {
	total := 0
	for _, message := range messages {
		total += session.EstimateTokens(message)
	}
	return total
}

func (p microcompactProjector) logSkip(err error) {
	var skip microcompactSkip
	if !asMicrocompactSkip(err, &skip) {
		return
	}
	log.Printf(
		"trace_id=%s microcompact skipped: %s tool=%s tool_call_id=%s",
		p.traceID,
		skip.reason,
		skip.tool,
		skip.callID,
	)
}

func microcompactParseFailed(toolName string, callID string) error {
	return microcompactSkip{
		reason: "parse failed",
		tool:   strings.TrimSpace(toolName),
		callID: strings.TrimSpace(callID),
	}
}

func microcompactUnsupported(toolName string, callID string, detail string) error {
	reason := "unsupported payload"
	if trimmed := strings.TrimSpace(detail); trimmed != "" {
		reason = fmt.Sprintf("%s (%s)", reason, trimmed)
	}
	return microcompactSkip{
		reason: reason,
		tool:   strings.TrimSpace(toolName),
		callID: strings.TrimSpace(callID),
	}
}

func asMicrocompactSkip(err error, target *microcompactSkip) bool {
	skip, ok := err.(microcompactSkip)
	if !ok || target == nil {
		return false
	}
	*target = skip
	return true
}
