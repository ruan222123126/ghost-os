package sessiondraft

import (
	"fmt"
	"strconv"
	"strings"

	bridgesession "ghost-os/bridge/session"
)

func projectTurnDraftTextDelta(
	draft *bridgesession.TurnDraft,
	traceID string,
	text string,
) bool {
	if draft == nil || text == "" {
		return false
	}

	changed := false
	for _, unit := range consumeTurnDraftToolTagChunk(draft, text) {
		switch unit.Kind {
		case draftToolTagUnitText:
			changed = appendTurnDraftAssistant(draft, unit.Text) || changed
		case draftToolTagUnitOpen:
			changed = projectTurnDraftToolTagOpen(draft, traceID, unit) || changed
		case draftToolTagUnitArgs:
			changed = projectTurnDraftToolTagArgs(draft, traceID, unit) || changed
		case draftToolTagUnitClose:
			changed = projectTurnDraftToolTagClose(draft, traceID, unit) || changed
		}
	}
	return changed
}

func projectTurnDraftStructuredToolStart(
	draft *bridgesession.TurnDraft,
	traceID string,
	payload any,
) bool {
	messageID := resolveTurnDraftPreviewToolMessageID(draft, traceID, payload)
	if messageID == "" {
		return false
	}

	toolName := payloadString(payload, "tool_name")
	tool := draftToolByID(draft, messageID)
	content := ""
	if tool != nil {
		content = tool.Content
	}
	next := bridgesession.TurnDraftTool{
		ID:         messageID,
		Content:    content,
		ToolInput:  draftResolveToolInput(toolName, content),
		ToolName:   draftFirstNonEmpty(toolName, draftToolName(tool)),
		ToolStatus: draftToolPendingStatus,
		ToolCallID: draftToolCallID(tool),
		TraceID:    strings.TrimSpace(traceID),
	}
	return upsertTurnDraftTool(draft, next)
}

func projectTurnDraftStructuredToolDelta(
	draft *bridgesession.TurnDraft,
	traceID string,
	payload any,
) bool {
	messageID := resolveTurnDraftPreviewToolMessageID(draft, traceID, payload)
	argsFragment := payloadString(payload, "arguments_fragment")
	if messageID == "" || argsFragment == "" {
		return false
	}

	tool := draftToolByID(draft, messageID)
	content := argsFragment
	if tool != nil {
		content = tool.Content + argsFragment
	}
	next := bridgesession.TurnDraftTool{
		ID:         messageID,
		Content:    content,
		ToolInput:  draftResolveToolInput(draftToolName(tool), content),
		ToolName:   draftToolName(tool),
		ToolStatus: draftToolPendingStatus,
		ToolCallID: draftToolCallID(tool),
		TraceID:    strings.TrimSpace(traceID),
	}
	return upsertTurnDraftTool(draft, next)
}

func projectTurnDraftStructuredToolEnd(
	draft *bridgesession.TurnDraft,
	traceID string,
	payload any,
) bool {
	messageID := resolveTurnDraftPreviewToolMessageID(draft, traceID, payload)
	if messageID == "" {
		return false
	}

	tool := draftToolByID(draft, messageID)
	content := ""
	if tool != nil {
		content = tool.Content
	}
	next := bridgesession.TurnDraftTool{
		ID:         messageID,
		Content:    content,
		ToolInput:  draftResolveToolInput(draftToolName(tool), content),
		ToolName:   draftToolName(tool),
		ToolStatus: draftToolPendingStatus,
		ToolCallID: draftToolCallID(tool),
		TraceID:    strings.TrimSpace(traceID),
	}
	return upsertTurnDraftTool(draft, next)
}

func resolveTurnDraftPreviewToolMessageID(
	draft *bridgesession.TurnDraft,
	traceID string,
	payload any,
) string {
	toolCallIndex, hasIndex := payloadInt(payload, "tool_call_index")
	toolCallID := payloadString(payload, "tool_call_id")
	if !hasIndex {
		return resolveDraftToolMessageID(draft, traceID, toolCallID, "")
	}

	if existing := draftPreviewToolMessageID(draft, toolCallIndex); existing != "" {
		return existing
	}
	if strings.TrimSpace(toolCallID) != "" {
		return fmt.Sprintf("stream-tool:%s:%s", strings.TrimSpace(traceID), strings.TrimSpace(toolCallID))
	}
	return fmt.Sprintf("stream-tool:%s:preview:%d:index:%d", strings.TrimSpace(traceID), nextTurnDraftPreviewSeq(draft), toolCallIndex)
}

func resolveDraftToolMessageID(
	draft *bridgesession.TurnDraft,
	traceID string,
	toolCallID string,
	fallback string,
) string {
	trimmedCallID := strings.TrimSpace(toolCallID)
	if trimmedCallID != "" {
		if existing := draftToolMessageIDByCallID(draft, trimmedCallID); existing != "" {
			return existing
		}
		if previewID := oldestPendingDraftPreviewID(draft); previewID != "" {
			return previewID
		}
		return fmt.Sprintf("stream-tool:%s:%s", strings.TrimSpace(traceID), trimmedCallID)
	}
	trimmedFallback := strings.TrimSpace(fallback)
	if trimmedFallback == "" {
		return ""
	}
	return fmt.Sprintf("stream-tool:%s:%s", strings.TrimSpace(traceID), trimmedFallback)
}

func draftToolMessageIDByCallID(draft *bridgesession.TurnDraft, toolCallID string) string {
	for _, tool := range draft.Tools {
		if strings.TrimSpace(tool.ToolCallID) == strings.TrimSpace(toolCallID) {
			return tool.ID
		}
	}
	return ""
}

func draftPreviewToolMessageID(draft *bridgesession.TurnDraft, toolCallIndex int) string {
	for _, tool := range draft.Tools {
		index, ok := parseTurnDraftPreviewIndex(tool.ID)
		if ok && index == toolCallIndex {
			return tool.ID
		}
	}
	return ""
}

func oldestPendingDraftPreviewID(draft *bridgesession.TurnDraft) string {
	for _, orderKey := range draft.ItemOrder {
		if !strings.HasPrefix(orderKey, draftToolOrderPrefix) {
			continue
		}
		toolID := strings.TrimPrefix(orderKey, draftToolOrderPrefix)
		tool := draftToolByID(draft, toolID)
		if tool == nil || strings.TrimSpace(tool.ToolCallID) != "" {
			continue
		}
		if _, ok := parseTurnDraftPreviewIndex(tool.ID); ok {
			return tool.ID
		}
	}
	return ""
}

func nextTurnDraftPreviewSeq(draft *bridgesession.TurnDraft) int {
	maxSeq := 0
	for _, tool := range draft.Tools {
		seq, ok := parseTurnDraftPreviewSequence(tool.ID)
		if ok && seq > maxSeq {
			maxSeq = seq
		}
	}
	return maxSeq + 1
}

func parseTurnDraftPreviewIndex(id string) (int, bool) {
	const marker = ":index:"
	markerIndex := strings.LastIndex(id, marker)
	if markerIndex < 0 {
		return 0, false
	}
	value, err := strconv.Atoi(id[markerIndex+len(marker):])
	return value, err == nil
}

func parseTurnDraftPreviewSequence(id string) (int, bool) {
	const marker = ":preview:"
	start := strings.Index(id, marker)
	if start < 0 {
		return 0, false
	}
	rest := id[start+len(marker):]
	end := strings.Index(rest, ":index:")
	if end < 0 {
		return 0, false
	}
	value, err := strconv.Atoi(rest[:end])
	return value, err == nil
}

func draftResolveToolInput(toolName string, inputText string) string {
	if inputText == "" || !draftSupportsToolInput(toolName) {
		return ""
	}
	return inputText
}

func draftSupportsToolInput(toolName string) bool {
	switch normalizeDraftToolName(toolName) {
	case "apply_diff", "bash_exec", "list_files", "read_file", "search_files", "sfind", "write_file":
		return true
	default:
		return false
	}
}

func normalizeDraftToolName(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = strings.TrimPrefix(normalized, "tools.")
	switch normalized {
	case "bash", "run":
		return "bash_exec"
	case "read":
		return "read_file"
	case "search":
		return "search_files"
	case "write":
		return "write_file"
	case "edit", "patch":
		return "apply_diff"
	case "script":
		return "script_exec"
	default:
		return normalized
	}
}

func draftToolName(tool *bridgesession.TurnDraftTool) string {
	if tool == nil {
		return ""
	}
	return strings.TrimSpace(tool.ToolName)
}

func draftToolCallID(tool *bridgesession.TurnDraftTool) string {
	if tool == nil {
		return ""
	}
	return strings.TrimSpace(tool.ToolCallID)
}
