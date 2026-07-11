package sessionturn

import (
	"encoding/json"
	"strings"
)

type AnsweredHumanQuestionInput struct {
	QuestionID    string
	Prompt        string
	SelectionMode string
	Options       []HumanQuestionOptionInput
	Answer        string
}

type HumanQuestionOptionInput struct {
	Label       string
	AllowCustom bool
}

func BuildResolvedHumanQuestionToolResult(item AnsweredHumanQuestionInput) (string, string, string) {
	return askHumanResolvedQuestionToolResult(item)
}

func askHumanResolvedQuestionToolResult(item AnsweredHumanQuestionInput) (string, string, string) {
	payload := map[string]any{
		"question_id": item.QuestionID,
		"prompt":      item.Prompt,
		"answer":      item.Answer,
	}
	if selectionMode := strings.TrimSpace(item.SelectionMode); selectionMode != "" {
		payload["selection_mode"] = selectionMode
	}
	if options := askHumanResolvedQuestionOptions(item.Options); len(options) > 0 {
		payload["options"] = options
	}
	return "ask_human", mustEncodeResolvedQuestionPayload(payload), ""
}

func askHumanResolvedQuestionOptions(
	raw []HumanQuestionOptionInput,
) []map[string]any {
	if len(raw) == 0 {
		return nil
	}

	out := make([]map[string]any, 0, len(raw))
	for _, option := range raw {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			continue
		}
		out = append(out, map[string]any{
			"label":        label,
			"allow_custom": option.AllowCustom,
		})
	}
	return out
}

func mustEncodeResolvedQuestionPayload(payload map[string]any) string {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return `{}`
	}
	return string(encoded)
}
