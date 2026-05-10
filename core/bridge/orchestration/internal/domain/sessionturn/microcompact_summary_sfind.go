package sessionturn

import (
	"fmt"
	"sort"
	"strings"
)

type toolSearchProjectionArgs struct {
	Action string `json:"action"`
	Kind   string `json:"kind,omitempty"`
	Query  string `json:"query,omitempty"`
}

type toolSearchProjectionPayload struct {
	Action string                     `json:"action"`
	Kind   string                     `json:"kind,omitempty"`
	Items  []toolSearchProjectionItem `json:"items,omitempty"`
}

type toolSearchProjectionItem struct {
	Kind               string `json:"kind,omitempty"`
	Name               string `json:"name"`
	Status             string `json:"status,omitempty"`
	Summary            string `json:"summary,omitempty"`
	AvailableNow       bool   `json:"available_now,omitempty"`
	AvailableNextTurn  bool   `json:"available_next_turn,omitempty"`
	RemainingIdleTurns int    `json:"remaining_idle_turns,omitempty"`
}

func summarizeToolSearchResult(pair microcompactToolPair) (string, error) {
	args, err := decodeJSONText[toolSearchProjectionArgs](string(pair.call.Arguments))
	if err != nil {
		return "", microcompactParseFailed(pair.toolName, pair.toolCallID)
	}
	payload, err := decodeJSONText[toolSearchProjectionPayload](pair.envelope.Output)
	if err != nil {
		return "", microcompactUnsupported(pair.toolName, pair.toolCallID, "sfind payload")
	}
	switch normalizeToolSearchAction(args.Action) {
	case "search":
		return summarizeToolSearchSearch(args, payload), nil
	case "load":
		return summarizeToolSearchLoad(normalizeToolSearchKind(args.Kind), payload.Items), nil
	case "unload":
		return summarizeToolSearchUnload(normalizeToolSearchKind(args.Kind), payload.Items), nil
	case "list":
		return summarizeToolSearchList(normalizeToolSearchKind(args.Kind), payload.Items), nil
	default:
		return "", microcompactUnsupported(pair.toolName, pair.toolCallID, "sfind action")
	}
}

func summarizeToolSearchSearch(args toolSearchProjectionArgs, payload toolSearchProjectionPayload) string {
	kind := normalizeToolSearchKind(args.Kind)
	count := len(payload.Items)
	names := make([]string, 0, microcompactTopSearchResults)
	for _, item := range payload.Items {
		if len(names) == microcompactTopSearchResults {
			break
		}
		names = append(names, fmt.Sprintf("`%s`", strings.TrimSpace(item.Name)))
	}
	query := truncateMicrocompactText(args.Query, microcompactMaxTextPreview)
	if len(names) == 0 {
		return fmt.Sprintf("sfind search kind=%s query=%q results=%d.", kind, query, count)
	}
	return fmt.Sprintf("sfind search kind=%s query=%q results=%d: %s.", kind, query, count, strings.Join(names, ", "))
}

func summarizeToolSearchLoad(kind string, items []toolSearchProjectionItem) string {
	loaded := filterToolSearchItems(items, "loaded")
	alreadyLoaded := filterToolSearchItems(items, "already_loaded")
	noun := toolSearchKindNoun(kind)
	parts := make([]string, 0, 2)
	if len(loaded) > 0 {
		part := fmt.Sprintf("Loaded dynamic session %s via sfind: %s.", noun, joinToolSearchNames(loaded))
		if allToolSearchItemsPending(loaded) {
			part += " These items become available on a future turn."
		}
		if allToolSearchItemsAvailableNow(loaded) {
			part += " These items are available now in the current user turn."
		}
		parts = append(parts, part)
	}
	if len(alreadyLoaded) > 0 {
		parts = append(parts, fmt.Sprintf("Already loaded via sfind: %s.", joinToolSearchNames(alreadyLoaded)))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("Loaded dynamic session %s via sfind.", noun)
	}
	return strings.Join(parts, " ")
}

func summarizeToolSearchUnload(kind string, items []toolSearchProjectionItem) string {
	noun := toolSearchKindNoun(kind)
	if len(items) == 0 {
		return fmt.Sprintf("Unloaded dynamic session %s via sfind.", noun)
	}
	return fmt.Sprintf("Unloaded dynamic session %s via sfind: %s.", noun, joinToolSearchNames(items))
}

func summarizeToolSearchList(kind string, items []toolSearchProjectionItem) string {
	noun := toolSearchKindNoun(kind)
	if len(items) == 0 {
		return fmt.Sprintf("Dynamic %s status via sfind: no items loaded.", noun)
	}
	parts := make([]string, 0, len(items))
	for _, item := range sortedToolSearchItems(items) {
		parts = append(parts, summarizeToolSearchStatus(item))
	}
	return fmt.Sprintf("Dynamic %s status via sfind: %s.", noun, strings.Join(parts, "; "))
}

func summarizeToolSearchStatus(item toolSearchProjectionItem) string {
	name := fmt.Sprintf("`%s`", strings.TrimSpace(item.Name))
	status := strings.ToLower(strings.TrimSpace(item.Status))
	switch status {
	case "pending":
		return fmt.Sprintf("%s is pending for a future turn", name)
	case "expired":
		return fmt.Sprintf("%s is expired", name)
	case "active":
		return fmt.Sprintf("%s is active (remaining_idle_turns=%d)", name, item.RemainingIdleTurns)
	default:
		if item.AvailableNextTurn && !item.AvailableNow {
			return fmt.Sprintf("%s becomes available on a future turn", name)
		}
		if item.AvailableNow {
			return fmt.Sprintf("%s is available now", name)
		}
		return fmt.Sprintf("%s status=%s", name, status)
	}
}

func normalizeToolSearchAction(action string) string {
	return strings.ToLower(strings.TrimSpace(action))
}

func normalizeToolSearchKind(kind string) string {
	if trimmed := strings.ToLower(strings.TrimSpace(kind)); trimmed != "" {
		return trimmed
	}
	return "skill"
}

func toolSearchKindNoun(kind string) string {
	if normalizeToolSearchKind(kind) == "skill" {
		return "skills"
	}
	return "tools"
}

func filterToolSearchItems(items []toolSearchProjectionItem, status string) []toolSearchProjectionItem {
	filtered := make([]toolSearchProjectionItem, 0, len(items))
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), status) {
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
