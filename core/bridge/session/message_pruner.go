package session

import "ghost-os/bridge/llm"

type messageSpan struct {
	start  int
	end    int
	tokens int
}

type messagePruner struct {
	recentMessagesToKeep int
	estimateTokens       func(llm.Message) int
}

func newMessagePruner(recentMessagesToKeep int, estimate func(llm.Message) int) messagePruner {
	return messagePruner{
		recentMessagesToKeep: recentMessagesToKeep,
		estimateTokens:       estimate,
	}
}

func (p messagePruner) Prune(messages []llm.Message, maxTokens int) []llm.Message {
	if len(messages) == 0 {
		return nil
	}
	if maxTokens <= 0 {
		return llm.CloneMessages(messages)
	}
	total := p.totalTokens(messages)
	if total <= maxTokens {
		return llm.CloneMessages(messages)
	}
	systemIndex, start := systemMessageIndex(messages)
	recentStart := maxInt(start, len(messages)-p.recentMessagesToKeep)
	spans := p.buildMessageSpans(messages, start)
	keep, total := p.dropOldSpans(spans, total, maxTokens, recentStart)
	keep, total = p.dropRemainingSpans(keep, spans, total, maxTokens)
	pruned := collectPrunedMessages(messages, spans, keep, systemIndex)
	if total > maxTokens {
		pruned = p.clampMessagesToTokenLimit(pruned, maxTokens)
	}
	return llm.CloneMessages(pruned)
}

func systemMessageIndex(messages []llm.Message) (int, int) {
	if len(messages) > 0 && messages[0].Role == llm.RoleSystem {
		return 0, 1
	}
	return -1, 0
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func (p messagePruner) totalTokens(messages []llm.Message) int {
	total := 0
	for _, msg := range messages {
		total += p.safeEstimate(msg)
	}
	return total
}

func (p messagePruner) safeEstimate(msg llm.Message) int {
	if p.estimateTokens == nil {
		return 0
	}
	return p.estimateTokens(msg)
}

func (p messagePruner) buildMessageSpans(messages []llm.Message, start int) []messageSpan {
	if start >= len(messages) {
		return nil
	}
	spans := make([]messageSpan, 0, len(messages)-start)
	for index := start; index < len(messages); {
		span := p.messageSpanAt(messages, index)
		spans = append(spans, span)
		index = span.end
	}
	return spans
}

func (p messagePruner) messageSpanAt(messages []llm.Message, start int) messageSpan {
	end := assistantToolSpanEnd(messages, start)
	return messageSpan{
		start:  start,
		end:    end,
		tokens: p.totalTokens(messages[start:end]),
	}
}

func assistantToolSpanEnd(messages []llm.Message, start int) int {
	end := start + 1
	if messages[start].Role != llm.RoleAssistant || len(messages[start].ToolCalls) == 0 {
		return end
	}
	for end < len(messages) && messages[end].Role == llm.RoleTool {
		end++
	}
	return end
}

func (p messagePruner) dropOldSpans(spans []messageSpan, total int, maxTokens int, recentStart int) ([]bool, int) {
	keep := make([]bool, len(spans))
	for index := range keep {
		keep[index] = true
	}
	for index, span := range spans {
		if total <= maxTokens {
			break
		}
		if span.end > recentStart {
			continue
		}
		keep[index] = false
		total -= span.tokens
	}
	return keep, total
}

func (p messagePruner) dropRemainingSpans(keep []bool, spans []messageSpan, total int, maxTokens int) ([]bool, int) {
	kept := countKeptSpans(keep)
	for index, span := range spans {
		if total <= maxTokens || kept <= 1 {
			break
		}
		if !keep[index] {
			continue
		}
		keep[index] = false
		kept--
		total -= span.tokens
	}
	return keep, total
}

func countKeptSpans(keep []bool) int {
	total := 0
	for _, keepSpan := range keep {
		if keepSpan {
			total++
		}
	}
	return total
}

func collectPrunedMessages(messages []llm.Message, spans []messageSpan, keep []bool, systemIndex int) []llm.Message {
	pruned := make([]llm.Message, 0, len(messages))
	if systemIndex >= 0 {
		pruned = append(pruned, messages[systemIndex])
	}
	for index, span := range spans {
		if !keep[index] {
			continue
		}
		pruned = append(pruned, messages[span.start:span.end]...)
	}
	return pruned
}
