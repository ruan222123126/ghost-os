package session

import (
	"strings"

	"ghost-os/bridge/llm"
)

const (
	minimumMessageBudget = 4
	minimumTruncateRatio = 0.05
)

func (p messagePruner) clampMessagesToTokenLimit(messages []llm.Message, maxTokens int) []llm.Message {
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
	out := llm.CloneMessages(messages)
	systemTokens, start := p.systemMessageBudget(out)
	if systemTokens > maxTokens {
		out[0] = p.truncateMessage(out[0], maxTokens)
		return out[:1]
	}
	if start >= len(out) {
		return out
	}
	spanBudget := maxTokens - systemTokens
	if spanBudget <= 0 {
		return out[:start]
	}
	return p.mergeSystemAndTruncated(out, start, spanBudget)
}

func (p messagePruner) systemMessageBudget(messages []llm.Message) (int, int) {
	if len(messages) == 0 || messages[0].Role != llm.RoleSystem {
		return 0, 0
	}
	return p.safeEstimate(messages[0]), 1
}

func (p messagePruner) mergeSystemAndTruncated(messages []llm.Message, start int, spanBudget int) []llm.Message {
	truncated := p.truncateMessagesToBudget(messages[start:], spanBudget)
	if start == 0 {
		return truncated
	}
	merged := make([]llm.Message, 0, 1+len(truncated))
	merged = append(merged, messages[0])
	merged = append(merged, truncated...)
	return merged
}

func (p messagePruner) truncateMessagesToBudget(messages []llm.Message, budget int) []llm.Message {
	if len(messages) == 0 || budget <= 0 {
		return nil
	}
	out := llm.CloneMessages(messages)
	weights, total := p.messageWeights(out)
	if total <= budget {
		return out
	}
	shares := allocateMessageShares(weights, budget)
	for index := range out {
		out[index] = p.truncateMessage(out[index], shares[index])
	}
	return p.rebalanceBudget(out, budget)
}

func (p messagePruner) messageWeights(messages []llm.Message) ([]int, int) {
	weights := make([]int, len(messages))
	total := 0
	for index, msg := range messages {
		weight := p.safeEstimate(msg)
		if weight < minimumEstimatedTokens {
			weight = minimumEstimatedTokens
		}
		weights[index] = weight
		total += weight
	}
	return weights, total
}

func allocateMessageShares(weights []int, budget int) []int {
	shares := make([]int, len(weights))
	remaining := budget
	totalWeight := sumInts(weights)
	for index, weight := range weights {
		share := proportionalShare(weight, budget, totalWeight)
		shares[index] = share
		remaining -= share
	}
	remaining = rebalanceShareFloor(shares, remaining)
	if remaining > 0 && len(shares) > 0 {
		shares[len(shares)-1] += remaining
	}
	return shares
}

func sumInts(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func proportionalShare(weight int, budget int, totalWeight int) int {
	if totalWeight <= 0 {
		return minimumMessageBudget
	}
	share := int(float64(weight) * float64(budget) / float64(totalWeight))
	if share < minimumMessageBudget {
		return minimumMessageBudget
	}
	return share
}

func rebalanceShareFloor(shares []int, remaining int) int {
	for remaining < 0 {
		adjusted := false
		for index := 0; index < len(shares) && remaining < 0; index++ {
			if shares[index] <= minimumMessageBudget {
				continue
			}
			shares[index]--
			remaining++
			adjusted = true
		}
		if !adjusted {
			break
		}
	}
	return remaining
}

func (p messagePruner) rebalanceBudget(messages []llm.Message, budget int) []llm.Message {
	total := p.totalTokens(messages)
	if total <= budget {
		return messages
	}
	for index := range messages {
		if total <= budget {
			break
		}
		remainingBudget := budget - (total - p.safeEstimate(messages[index]))
		if remainingBudget < minimumMessageBudget {
			remainingBudget = minimumMessageBudget
		}
		messages[index] = p.truncateMessage(messages[index], remainingBudget)
		total = p.totalTokens(messages)
	}
	return messages
}

func (p messagePruner) truncateMessage(msg llm.Message, maxTokens int) llm.Message {
	if maxTokens <= 0 {
		return msg
	}
	out := llm.CloneMessages([]llm.Message{msg})[0]
	if p.safeEstimate(out) <= maxTokens {
		return out
	}
	out = p.truncateMessageText(out, maxTokens)
	if p.safeEstimate(out) <= maxTokens {
		return out
	}
	out = dropImageContent(out)
	if p.safeEstimate(out) <= maxTokens {
		return out
	}
	out = truncateToolCallArguments(out)
	if p.safeEstimate(out) <= maxTokens {
		return out
	}
	return truncateToMarker(out, maxTokens, p.safeEstimate)
}

func (p messagePruner) truncateMessageText(msg llm.Message, maxTokens int) llm.Message {
	estimated := p.safeEstimate(msg)
	if estimated <= maxTokens {
		return msg
	}
	ratio := float64(maxTokens) / float64(estimated)
	if ratio <= 0 {
		ratio = minimumTruncateRatio
	}
	msg.Text = truncateByRatio(msg.Text, ratio)
	for index := range msg.Content {
		if strings.EqualFold(strings.TrimSpace(msg.Content[index].Type), llm.ContentTypeText) {
			msg.Content[index].Text = truncateByRatio(msg.Content[index].Text, ratio)
		}
	}
	return msg
}

func dropImageContent(msg llm.Message) llm.Message {
	if len(msg.Content) == 0 {
		return msg
	}
	filtered := make([]llm.ContentPart, 0, len(msg.Content))
	for _, part := range msg.Content {
		if strings.EqualFold(strings.TrimSpace(part.Type), llm.ContentTypeImage) {
			continue
		}
		filtered = append(filtered, part)
	}
	msg.Content = filtered
	return msg
}

func truncateToolCallArguments(msg llm.Message) llm.Message {
	for index := range msg.ToolCalls {
		msg.ToolCalls[index].Arguments = []byte(`{"_truncated":true}`)
	}
	return msg
}

func truncateToMarker(msg llm.Message, maxTokens int, estimate func(llm.Message) int) llm.Message {
	msg.Text = "[truncated]"
	for index := range msg.Content {
		msg.Content[index].Text = ""
		msg.Content[index].Image = nil
	}
	if estimate(msg) > maxTokens {
		msg.Content = nil
	}
	return msg
}

func truncateByRatio(value string, ratio float64) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	limit := int(float64(len(runes)) * ratio)
	if limit <= 0 {
		limit = 1
	}
	if limit >= len(runes) {
		return value
	}
	return string(runes[:limit])
}
