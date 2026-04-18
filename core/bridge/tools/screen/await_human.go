package screen

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/session"
)

const awaitingHumanStatus = "awaiting_human"

type awaitingQuestion struct {
	Prompt        string
	SelectionMode string
	Options       []AskHumanOption
}

type awaitingHumanPayload struct {
	Status        string           `json:"status"`
	QuestionID    string           `json:"question_id"`
	Prompt        string           `json:"prompt"`
	SelectionMode string           `json:"selection_mode,omitempty"`
	Options       []AskHumanOption `json:"options,omitempty"`
}

func interpretAwaitingHumanResult(output string) ExecuteMeta {
	var payload awaitingHumanPayload
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return ExecuteMeta{}
	}
	if strings.TrimSpace(payload.Status) != awaitingHumanStatus {
		return ExecuteMeta{}
	}
	questionID := strings.TrimSpace(payload.QuestionID)
	prompt := strings.TrimSpace(payload.Prompt)
	if questionID == "" || prompt == "" {
		return ExecuteMeta{}
	}
	options, ok := normalizeAwaitingOptions(payload.Options)
	if !ok {
		return ExecuteMeta{}
	}
	selectionMode, ok := normalizeAwaitingSelectionMode(payload.SelectionMode, len(options) > 0)
	if !ok {
		return ExecuteMeta{}
	}
	return ExecuteMeta{
		AwaitingHuman: &AwaitingHumanSignal{
			QuestionID:    questionID,
			Prompt:        prompt,
			SelectionMode: selectionMode,
			Options:       options,
		},
	}
}

func normalizeAwaitingSelectionMode(raw string, hasOptions bool) (string, bool) {
	selectionMode := strings.ToLower(strings.TrimSpace(raw))
	if !hasOptions {
		return "", selectionMode == ""
	}
	if selectionMode == "" {
		return session.HumanQuestionSelectionSingle, true
	}
	switch selectionMode {
	case session.HumanQuestionSelectionSingle, session.HumanQuestionSelectionMultiple:
		return selectionMode, true
	default:
		return "", false
	}
}

func normalizeAwaitingOptions(options []AskHumanOption) ([]AskHumanOption, bool) {
	if len(options) == 0 {
		return nil, true
	}
	normalized := make([]AskHumanOption, 0, len(options))
	for index, option := range options {
		label := strings.TrimSpace(option.Label)
		if label == "" || (option.AllowCustom && index != len(options)-1) {
			return nil, false
		}
		normalized = append(normalized, AskHumanOption{Label: label, AllowCustom: option.AllowCustom})
	}
	return normalized, true
}

func cloneAwaitingOptions(options []AskHumanOption) []AskHumanOption {
	if len(options) == 0 {
		return nil
	}
	cloned := make([]AskHumanOption, 0, len(options))
	for _, option := range options {
		cloned = append(cloned, AskHumanOption{Label: option.Label, AllowCustom: option.AllowCustom})
	}
	return cloned
}

func sessionOptionsFromAskHuman(options []AskHumanOption) []session.HumanQuestionOption {
	if len(options) == 0 {
		return nil
	}
	mapped := make([]session.HumanQuestionOption, 0, len(options))
	for _, option := range options {
		mapped = append(mapped, session.HumanQuestionOption{Label: option.Label, AllowCustom: option.AllowCustom})
	}
	return mapped
}

func registerAwaitingHumanQuestion(
	ctx context.Context,
	questionID string,
	question awaitingQuestion,
	traceID string,
	toolName string,
) (string, error) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return "", fmt.Errorf("ask_human requires an active session")
	}
	toolCallID := ToolCallIDFromContext(ctx)
	if toolCallID == "" {
		return "", fmt.Errorf("ask_human requires tool call id in context")
	}
	sess.AddPendingQuestion(questionID, session.PendingHumanQuestion{
		Prompt:        strings.TrimSpace(question.Prompt),
		SelectionMode: strings.TrimSpace(question.SelectionMode),
		Options:       sessionOptionsFromAskHuman(question.Options),
		ToolName:      strings.TrimSpace(toolName),
		ToolCallID:    toolCallID,
		TraceID:       strings.TrimSpace(traceID),
		CreatedAt:     time.Now().UTC(),
	})
	encoded, err := json.Marshal(awaitingHumanPayload{
		Status:        awaitingHumanStatus,
		QuestionID:    strings.TrimSpace(questionID),
		Prompt:        strings.TrimSpace(question.Prompt),
		SelectionMode: strings.TrimSpace(question.SelectionMode),
		Options:       cloneAwaitingOptions(question.Options),
	})
	if err != nil {
		return "", fmt.Errorf("encode awaiting payload: %w", err)
	}
	return string(encoded), nil
}

func newQuestionID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "q-" + hex.EncodeToString(raw[:]), nil
}
