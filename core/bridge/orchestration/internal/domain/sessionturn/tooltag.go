package sessionturn

import (
	"strconv"
	"strings"

	bridgesession "ghost-os/bridge/session"
)

const (
	draftToolTagOpenToken  = "<t:"
	draftToolTagCloseToken = "</t>"

	draftToolTagUnitText  = "text"
	draftToolTagUnitOpen  = "tool_open"
	draftToolTagUnitArgs  = "tool_args"
	draftToolTagUnitClose = "tool_close"
)

type turnDraftToolTagUnit struct {
	Kind     string
	Text     string
	CallSeq  int
	ToolID   string
	ArgsText string
}

func consumeTurnDraftToolTagChunk(
	draft *bridgesession.TurnDraft,
	chunk string,
) []turnDraftToolTagUnit {
	if draft == nil || chunk == "" {
		return nil
	}

	state := ensureTurnDraftToolTagState(draft)
	units := make([]turnDraftToolTagUnit, 0, len(chunk))
	for _, char := range chunk {
		switch state.Mode {
		case "capture_id":
			consumeTurnDraftCaptureIDChar(state, string(char), &units)
		case "capture_args":
			consumeTurnDraftCaptureArgsChar(state, string(char), &units)
		default:
			consumeTurnDraftNormalChar(state, string(char), &units)
		}
	}
	return units
}

func ensureTurnDraftToolTagState(draft *bridgesession.TurnDraft) *bridgesession.TurnDraftToolTagState {
	if draft.ToolTagState != nil {
		return draft.ToolTagState
	}
	draft.ToolTagState = &bridgesession.TurnDraftToolTagState{
		Mode:        "normal",
		NextCallSeq: 1,
	}
	return draft.ToolTagState
}

func consumeTurnDraftNormalChar(
	state *bridgesession.TurnDraftToolTagState,
	char string,
	units *[]turnDraftToolTagUnit,
) {
	state.NormalCandidate += char
	for len(state.NormalCandidate) > 0 {
		if strings.HasPrefix(draftToolTagOpenToken, state.NormalCandidate) {
			if state.NormalCandidate == draftToolTagOpenToken {
				state.Mode = "capture_id"
				state.RawTagPrefix = draftToolTagOpenToken
				state.CurrentToolID = ""
				state.NormalCandidate = ""
			}
			return
		}
		appendTurnDraftTextUnit(units, state.NormalCandidate[:1])
		state.NormalCandidate = state.NormalCandidate[1:]
	}
}

func consumeTurnDraftCaptureIDChar(
	state *bridgesession.TurnDraftToolTagState,
	char string,
	units *[]turnDraftToolTagUnit,
) {
	state.RawTagPrefix += char
	if char != ">" {
		state.CurrentToolID += char
		return
	}

	toolID := strings.TrimSpace(state.CurrentToolID)
	if !isValidTurnDraftToolTagID(toolID) {
		appendTurnDraftTextUnit(units, state.RawTagPrefix)
		resetTurnDraftToolTagState(state)
		return
	}

	state.Mode = "capture_args"
	state.CurrentToolID = toolID
	state.ArgsBuffer = ""
	state.CloseCandidate = ""
	state.InString = false
	state.Escaped = false
	state.CurrentCallSeq = state.NextCallSeq
	state.NextCallSeq += 1
	state.RawTagPrefix = ""
	*units = append(*units, turnDraftToolTagUnit{
		Kind:    draftToolTagUnitOpen,
		CallSeq: state.CurrentCallSeq,
		ToolID:  toolID,
	})
}

func consumeTurnDraftCaptureArgsChar(
	state *bridgesession.TurnDraftToolTagState,
	char string,
	units *[]turnDraftToolTagUnit,
) {
	if !state.InString {
		state.CloseCandidate += char
		if strings.HasPrefix(draftToolTagCloseToken, state.CloseCandidate) {
			if state.CloseCandidate == draftToolTagCloseToken {
				*units = append(*units, turnDraftToolTagUnit{
					Kind:     draftToolTagUnitClose,
					CallSeq:  state.CurrentCallSeq,
					ToolID:   state.CurrentToolID,
					ArgsText: state.ArgsBuffer,
				})
				resetTurnDraftToolTagState(state)
			}
			return
		}
		flushTurnDraftCloseCandidate(state, units)
		return
	}
	appendTurnDraftArgsChar(state, char, units)
}

func flushTurnDraftCloseCandidate(
	state *bridgesession.TurnDraftToolTagState,
	units *[]turnDraftToolTagUnit,
) {
	for len(state.CloseCandidate) > 0 &&
		!strings.HasPrefix(draftToolTagCloseToken, state.CloseCandidate) {
		nextChar := state.CloseCandidate[:1]
		state.CloseCandidate = state.CloseCandidate[1:]
		appendTurnDraftArgsChar(state, nextChar, units)
	}
}

func appendTurnDraftArgsChar(
	state *bridgesession.TurnDraftToolTagState,
	char string,
	units *[]turnDraftToolTagUnit,
) {
	state.ArgsBuffer += char
	appendTurnDraftArgsUnit(units, state.CurrentCallSeq, char)

	if state.InString {
		if state.Escaped {
			state.Escaped = false
			return
		}
		if char == `\` {
			state.Escaped = true
			return
		}
		if char == `"` {
			state.InString = false
		}
		return
	}

	if char == `"` {
		state.InString = true
	}
}

func appendTurnDraftTextUnit(units *[]turnDraftToolTagUnit, text string) {
	if text == "" {
		return
	}
	lastIndex := len(*units) - 1
	if lastIndex >= 0 && (*units)[lastIndex].Kind == draftToolTagUnitText {
		(*units)[lastIndex].Text += text
		return
	}
	*units = append(*units, turnDraftToolTagUnit{
		Kind: draftToolTagUnitText,
		Text: text,
	})
}

func appendTurnDraftArgsUnit(units *[]turnDraftToolTagUnit, callSeq int, char string) {
	lastIndex := len(*units) - 1
	if lastIndex >= 0 &&
		(*units)[lastIndex].Kind == draftToolTagUnitArgs &&
		(*units)[lastIndex].CallSeq == callSeq {
		(*units)[lastIndex].Text += char
		return
	}
	*units = append(*units, turnDraftToolTagUnit{
		Kind:    draftToolTagUnitArgs,
		CallSeq: callSeq,
		Text:    char,
	})
}

func resetTurnDraftToolTagState(state *bridgesession.TurnDraftToolTagState) {
	state.Mode = "normal"
	state.NormalCandidate = ""
	state.RawTagPrefix = ""
	state.CurrentToolID = ""
	state.CloseCandidate = ""
	state.ArgsBuffer = ""
	state.InString = false
	state.Escaped = false
	state.CurrentCallSeq = 0
}

func isValidTurnDraftToolTagID(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}
	_, err := strconv.Atoi(strings.TrimSpace(raw))
	return err == nil
}

func projectTurnDraftToolTagOpen(
	draft *bridgesession.TurnDraft,
	traceID string,
	unit turnDraftToolTagUnit,
) bool {
	messageID := fmtTurnDraftTagToolMessageID(traceID, unit.CallSeq)
	content := draftToolContent(draft, messageID)
	return upsertTurnDraftTool(draft, bridgesession.TurnDraftTool{
		ID:         messageID,
		Content:    content,
		ToolName:   "tool#" + unit.ToolID,
		ToolStatus: draftToolPendingStatus,
		TraceID:    strings.TrimSpace(traceID),
	})
}

func projectTurnDraftToolTagArgs(
	draft *bridgesession.TurnDraft,
	traceID string,
	unit turnDraftToolTagUnit,
) bool {
	messageID := fmtTurnDraftTagToolMessageID(traceID, unit.CallSeq)
	content := draftToolContent(draft, messageID) + unit.Text
	return upsertTurnDraftTool(draft, bridgesession.TurnDraftTool{
		ID:         messageID,
		Content:    content,
		ToolName:   draftFirstNonEmpty(draftToolName(draftToolByID(draft, messageID)), "tool"),
		ToolStatus: draftToolPendingStatus,
		TraceID:    strings.TrimSpace(traceID),
	})
}

func projectTurnDraftToolTagClose(
	draft *bridgesession.TurnDraft,
	traceID string,
	unit turnDraftToolTagUnit,
) bool {
	messageID := fmtTurnDraftTagToolMessageID(traceID, unit.CallSeq)
	return upsertTurnDraftTool(draft, bridgesession.TurnDraftTool{
		ID:         messageID,
		Content:    unit.ArgsText,
		ToolName:   "tool#" + unit.ToolID,
		ToolStatus: draftToolPendingStatus,
		TraceID:    strings.TrimSpace(traceID),
	})
}

func fmtTurnDraftTagToolMessageID(traceID string, callSeq int) string {
	return "stream-tag-tool:" + strings.TrimSpace(traceID) + ":" + strconv.Itoa(callSeq)
}

func draftToolContent(draft *bridgesession.TurnDraft, id string) string {
	tool := draftToolByID(draft, id)
	if tool == nil {
		return ""
	}
	return tool.Content
}
