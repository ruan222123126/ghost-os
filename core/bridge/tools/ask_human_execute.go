package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/session"
)

func registerAwaitingHumanQuestion(
	ctx context.Context,
	questionID string,
	question askHumanArgs,
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
		Prompt:        question.Prompt,
		SelectionMode: question.SelectionMode,
		Options:       sessionOptionsFromAskHuman(question.Options),
		ToolName:      strings.TrimSpace(toolName),
		ToolCallID:    toolCallID,
		TraceID:       strings.TrimSpace(traceID),
		CreatedAt:     time.Now().UTC(),
	})
	payload := askHumanAwaitingPayload{
		Status:        "awaiting_human",
		QuestionID:    questionID,
		Prompt:        question.Prompt,
		SelectionMode: question.SelectionMode,
		Options:       cloneAskHumanOptions(question.Options),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode awaiting payload: %w", err)
	}
	return string(encoded), nil
}
