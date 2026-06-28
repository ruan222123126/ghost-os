package sessionturn

import (
	"encoding/json"
	"strings"

	"ghost-os/bridge/llm"
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

type answeredHumanInteractionPayload struct {
	QuestionID    string           `json:"question_id"`
	Prompt        string           `json:"prompt"`
	SelectionMode string           `json:"selection_mode,omitempty"`
	Options       []askHumanOption `json:"options,omitempty"`
	Answer        string           `json:"answer"`
}

func BuildResolvedHumanQuestionToolResult(item AnsweredHumanQuestionInput) (string, string, string) {
	return askHumanResolvedQuestionToolResult(item)
}

func decodeSessionHumanInteraction(result llm.ToolResultEnvelope) *sessionHumanInteraction {
	if strings.TrimSpace(result.Tool) != "ask_human" || strings.TrimSpace(result.Output) == "" {
		return nil
	}

	var payload answeredHumanInteractionPayload
	if err := json.Unmarshal([]byte(result.Output), &payload); err != nil {
		return nil
	}

	questionID := strings.TrimSpace(payload.QuestionID)
	prompt := strings.TrimSpace(payload.Prompt)
	if questionID == "" || prompt == "" {
		return nil
	}

	humanInteraction := &sessionHumanInteraction{
		QuestionID: questionID,
		Prompt:     prompt,
	}
	if selectionMode := strings.TrimSpace(payload.SelectionMode); selectionMode != "" {
		humanInteraction.SelectionMode = selectionMode
	}
	if len(payload.Options) > 0 {
		humanInteraction.Options = cloneSessionHumanInteractionOptions(payload.Options)
	}
	if strings.TrimSpace(payload.Answer) != "" {
		humanInteraction.Answer = payload.Answer
	}
	return humanInteraction
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

func cloneSessionHumanInteractionOptions(options []askHumanOption) []askHumanOption {
	if len(options) == 0 {
		return nil
	}
	cloned := make([]askHumanOption, 0, len(options))
	for _, option := range options {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			continue
		}
		cloned = append(cloned, askHumanOption{
			Label:       label,
			AllowCustom: option.AllowCustom,
		})
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}
