package orchestration

import (
	"encoding/json"
	"strings"

	"ghost-os/bridge/session"
)

func resolvedHumanQuestionToolResult(
	sess *session.Session,
	item session.AnsweredHumanQuestion,
) (string, string, string) {
	toolName := resolvedHumanQuestionToolName(item.Question)
	switch toolName {
	case "graphql_mutation", "graphql_text_mutation":
		return graphqlMutationResolvedQuestionToolResult(sess, item, toolName)
	default:
		return askHumanResolvedQuestionToolResult(item)
	}
}

func resolvedHumanQuestionToolName(question session.PendingHumanQuestion) string {
	toolName := strings.TrimSpace(question.ToolName)
	if toolName == "" {
		return "ask_human"
	}
	return toolName
}

func askHumanResolvedQuestionToolResult(item session.AnsweredHumanQuestion) (string, string, string) {
	payload := map[string]any{
		"question_id": item.QuestionID,
		"prompt":      item.Question.Prompt,
		"answer":      item.Answer,
	}
	if selectionMode := strings.TrimSpace(item.Question.SelectionMode); selectionMode != "" {
		payload["selection_mode"] = selectionMode
	}
	if options := askHumanResolvedQuestionOptions(item.Question.Options); len(options) > 0 {
		payload["options"] = options
	}
	return "ask_human", mustEncodeResolvedQuestionPayload(payload), ""
}

func askHumanResolvedQuestionOptions(
	raw []session.HumanQuestionOption,
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

func graphqlMutationResolvedQuestionToolResult(
	sess *session.Session,
	item session.AnsweredHumanQuestion,
	toolName string,
) (string, string, string) {
	intent, ok := sess.PendingGraphQLMutationIntentByQuestionID(item.QuestionID)
	if !ok {
		payload := map[string]any{
			"question_id":     item.QuestionID,
			"approval_status": "unknown",
			"answer":          item.Answer,
			"summary":         "",
			"intent_missing":  true,
			"prompt":          item.Question.Prompt,
		}
		return toolName, mustEncodeResolvedQuestionPayload(payload), ""
	}

	payload := map[string]any{
		"intent_id":       intent.IntentID,
		"approval_status": intent.Status,
		"question_id":     item.QuestionID,
		"answer":          item.Answer,
		"summary":         intent.Summary,
		"source":          intent.Source,
		"domain":          intent.Domain,
		"policy":          intent.PolicyName,
		"root_mutation":   intent.RootMutation,
	}
	return toolName, mustEncodeResolvedQuestionPayload(payload), intent.Summary
}

func mustEncodeResolvedQuestionPayload(payload map[string]any) string {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return `{}`
	}
	return string(encoded)
}
