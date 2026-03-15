package orchestration

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type toolSearchProjectionArgs struct {
	Action string `json:"action"`
}

type toolSearchProjectionPayload struct {
	Action string                     `json:"action"`
	Items  []toolSearchProjectionItem `json:"items,omitempty"`
}

type toolSearchProjectionItem struct {
	Name               string `json:"name"`
	Status             string `json:"status,omitempty"`
	AvailableNow       bool   `json:"available_now,omitempty"`
	AvailableNextTurn  bool   `json:"available_next_turn,omitempty"`
	RemainingIdleTurns int    `json:"remaining_idle_turns,omitempty"`
}

func projectMessagesForModel(messages []llm.Message, _ int) []llm.Message {
	if len(messages) == 0 {
		return nil
	}
	return compressToolSearchSpans(messages)
}

func compressToolSearchSpans(messages []llm.Message) []llm.Message {
	projected := make([]llm.Message, 0, len(messages))
	for index := 0; index < len(messages); {
		summary, ok := projectToolSearchSpan(messages, index)
		if ok {
			projected = append(projected, summary)
			index += 2
			continue
		}
		projected = append(projected, messages[index])
		index++
	}
	return llm.CloneMessages(projected)
}

func projectToolSearchSpan(messages []llm.Message, index int) (llm.Message, bool) {
	if index+1 >= len(messages) {
		return llm.Message{}, false
	}
	text, ok := summarizeToolSearchSpan(messages[index], messages[index+1])
	if !ok {
		return llm.Message{}, false
	}
	return llm.Message{Role: llm.RoleAssistant, Text: text}, true
}

func summarizeToolSearchSpan(assistantMsg llm.Message, toolMsg llm.Message) (string, bool) {
	call, ok := projectableToolSearchCall(assistantMsg)
	if !ok || toolMsg.Role != llm.RoleTool || strings.TrimSpace(toolMsg.ToolCallID) != strings.TrimSpace(call.ID) {
		return "", false
	}

	envelope, ok := agent.ParseToolResultEnvelope(toolMsg.Text)
	if !ok || envelope.Tool != tools.ToolSearchToolName {
		return "", false
	}
	args, ok := decodeToolSearchProjectionArgs(call.Arguments)
	if !ok || !isProjectableToolSearchAction(args.Action) {
		return "", false
	}
	return toolSearchSummaryText(args.Action, envelope)
}

func projectableToolSearchCall(message llm.Message) (llm.ToolCall, bool) {
	if message.Role != llm.RoleAssistant || len(message.ToolCalls) != 1 {
		return llm.ToolCall{}, false
	}
	call := message.ToolCalls[0]
	if strings.TrimSpace(call.Name) != tools.ToolSearchToolName {
		return llm.ToolCall{}, false
	}
	return call, true
}

func decodeToolSearchProjectionArgs(raw json.RawMessage) (toolSearchProjectionArgs, bool) {
	var args toolSearchProjectionArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return toolSearchProjectionArgs{}, false
	}
	return args, true
}

func isProjectableToolSearchAction(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "load", "unload", "list":
		return true
	default:
		return false
	}
}

func toolSearchSummaryText(action string, envelope agent.ToolResultEnvelope) (string, bool) {
	normalizedAction := strings.ToLower(strings.TrimSpace(action))
	if envelope.Status == "error" {
		return summarizeToolSearchError(normalizedAction, envelope.Error), true
	}

	payload, ok := decodeToolSearchProjectionPayload(envelope.Output)
	if !ok {
		return "", false
	}
	switch normalizedAction {
	case "load":
		return summarizeToolSearchLoad(payload.Items), true
	case "unload":
		return summarizeToolSearchUnload(payload.Items), true
	case "list":
		return summarizeToolSearchList(payload.Items), true
	default:
		return "", false
	}
}

func summarizeToolSearchError(action string, errText string) string {
	trimmed := strings.TrimSpace(errText)
	if trimmed == "" {
		trimmed = "unknown error"
	}
	return fmt.Sprintf("tfind %s failed: %s.", action, trimmed)
}

func decodeToolSearchProjectionPayload(raw string) (toolSearchProjectionPayload, bool) {
	var payload toolSearchProjectionPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return toolSearchProjectionPayload{}, false
	}
	return payload, true
}

func summarizeToolSearchLoad(items []toolSearchProjectionItem) string {
	loaded := filterToolSearchItems(items, "loaded")
	alreadyLoaded := filterToolSearchItems(items, "already_loaded")
	parts := make([]string, 0, 2)

	if len(loaded) > 0 {
		part := fmt.Sprintf("Loaded dynamic session tools via tfind: %s.", joinToolSearchNames(loaded))
		if allToolSearchItemsPending(loaded) {
			part += " These tools become available next turn."
		}
		if allToolSearchItemsAvailableNow(loaded) {
			part += " These tools are available now."
		}
		parts = append(parts, part)
	}
	if len(alreadyLoaded) > 0 {
		parts = append(parts, fmt.Sprintf("Already loaded via tfind: %s.", joinToolSearchNames(alreadyLoaded)))
	}
	if len(parts) == 0 {
		return "Loaded dynamic session tools via tfind."
	}
	return strings.Join(parts, " ")
}

func summarizeToolSearchUnload(items []toolSearchProjectionItem) string {
	if len(items) == 0 {
		return "Unloaded dynamic session tools via tfind."
	}
	return fmt.Sprintf("Unloaded dynamic session tools via tfind: %s.", joinToolSearchNames(items))
}

func summarizeToolSearchList(items []toolSearchProjectionItem) string {
	if len(items) == 0 {
		return "Dynamic tool status via tfind: no tools loaded."
	}
	parts := make([]string, 0, len(items))
	for _, item := range sortedToolSearchItems(items) {
		parts = append(parts, summarizeToolSearchStatus(item))
	}
	return "Dynamic tool status via tfind: " + strings.Join(parts, "; ") + "."
}

func summarizeToolSearchStatus(item toolSearchProjectionItem) string {
	name := fmt.Sprintf("`%s`", strings.TrimSpace(item.Name))
	status := strings.ToLower(strings.TrimSpace(item.Status))

	switch status {
	case "pending":
		return fmt.Sprintf("%s is pending and becomes available next turn", name)
	case "expired":
		return fmt.Sprintf("%s is expired", name)
	case "active":
		return fmt.Sprintf("%s is active (remaining_idle_turns=%d)", name, item.RemainingIdleTurns)
	default:
		if item.AvailableNextTurn && !item.AvailableNow {
			return fmt.Sprintf("%s becomes available next turn", name)
		}
		if item.AvailableNow {
			return fmt.Sprintf("%s is available now", name)
		}
		return fmt.Sprintf("%s status=%s", name, status)
	}
}

func filterToolSearchItems(items []toolSearchProjectionItem, wantStatus string) []toolSearchProjectionItem {
	filtered := make([]toolSearchProjectionItem, 0, len(items))
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), wantStatus) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func allToolSearchItemsPending(items []toolSearchProjectionItem) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if item.AvailableNow || !item.AvailableNextTurn {
			return false
		}
	}
	return true
}

func allToolSearchItemsAvailableNow(items []toolSearchProjectionItem) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if !item.AvailableNow || item.AvailableNextTurn {
			return false
		}
	}
	return true
}

func joinToolSearchNames(items []toolSearchProjectionItem) string {
	sorted := sortedToolSearchItems(items)
	names := make([]string, 0, len(sorted))
	for _, item := range sorted {
		names = append(names, fmt.Sprintf("`%s`", strings.TrimSpace(item.Name)))
	}
	return strings.Join(names, ", ")
}

func sortedToolSearchItems(items []toolSearchProjectionItem) []toolSearchProjectionItem {
	sorted := append([]toolSearchProjectionItem(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}
