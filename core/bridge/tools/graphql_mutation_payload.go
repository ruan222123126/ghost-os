package tools

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/session"
	"ghost-os/bridge/tools/internal/tooljson"
)

const (
	graphQLMutationActionPrepare     = "prepare"
	graphQLMutationActionCommit      = "commit"
	graphQLMutationActionDiscard     = "discard"
	graphQLMutationActionListPending = "list_pending"

	graphQLMutationApprovalApprove = "Approve"
	graphQLMutationApprovalReject  = "Reject"
	graphQLMutationApprovalEdit    = "Reject and edit"

	graphQLMutationSummaryMaxChars = 240
	graphQLMutationValueMaxChars   = 72
)

type graphQLMutationArgs struct {
	Action        string         `json:"action"`
	Source        string         `json:"source,omitempty"`
	Domain        string         `json:"domain,omitempty"`
	Mutation      string         `json:"mutation,omitempty"`
	Variables     map[string]any `json:"variables,omitempty"`
	OperationName string         `json:"operation_name,omitempty"`
	IntentID      string         `json:"intent_id,omitempty"`
}

type graphQLMutationAwaitingPayload struct {
	Status        string           `json:"status"`
	IntentID      string           `json:"intent_id"`
	QuestionID    string           `json:"question_id"`
	Prompt        string           `json:"prompt"`
	Summary       string           `json:"summary"`
	SelectionMode string           `json:"selection_mode,omitempty"`
	Options       []AskHumanOption `json:"options,omitempty"`
}

type graphQLMutationPendingItem struct {
	IntentID     string `json:"intent_id"`
	Source       string `json:"source"`
	Domain       string `json:"domain"`
	Policy       string `json:"policy"`
	Status       string `json:"status"`
	PreparedAt   string `json:"prepared_at"`
	ApprovedAt   string `json:"approved_at,omitempty"`
	ExecutedAt   string `json:"executed_at,omitempty"`
	Summary      string `json:"summary,omitempty"`
	RootMutation string `json:"root_mutation"`
}

func decodeGraphQLMutationArgs(argsJSON json.RawMessage) (graphQLMutationArgs, error) {
	var args graphQLMutationArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return graphQLMutationArgs{}, fmt.Errorf("decode args: %w", err)
	}
	args.Action = strings.ToLower(strings.TrimSpace(args.Action))
	args.Source = strings.TrimSpace(args.Source)
	args.Domain = strings.TrimSpace(args.Domain)
	args.Mutation = strings.TrimSpace(args.Mutation)
	args.OperationName = strings.TrimSpace(args.OperationName)
	args.IntentID = strings.TrimSpace(args.IntentID)
	return args, nil
}

func interpretGraphQLMutationAwaitingResult(output string) ExecuteMeta {
	var payload graphQLMutationAwaitingPayload
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return ExecuteMeta{}
	}
	if payload.Status != "awaiting_human" {
		return ExecuteMeta{}
	}
	if strings.TrimSpace(payload.QuestionID) == "" || strings.TrimSpace(payload.Prompt) == "" {
		return ExecuteMeta{}
	}
	return ExecuteMeta{
		AwaitingHuman: &AwaitingHumanSignal{
			QuestionID:    payload.QuestionID,
			Prompt:        payload.Prompt,
			SelectionMode: payload.SelectionMode,
			Options:       cloneAskHumanOptions(payload.Options),
		},
	}
}

func newGraphQLMutationAwaitingPayload(
	intent session.PendingGraphQLMutationIntent,
	prompt string,
) graphQLMutationAwaitingPayload {
	return graphQLMutationAwaitingPayload{
		Status:        "awaiting_human",
		IntentID:      intent.IntentID,
		QuestionID:    intent.QuestionID,
		Prompt:        prompt,
		Summary:       intent.Summary,
		SelectionMode: session.HumanQuestionSelectionSingle,
		Options: []AskHumanOption{
			{Label: graphQLMutationApprovalApprove},
			{Label: graphQLMutationApprovalReject},
			{Label: graphQLMutationApprovalEdit, AllowCustom: true},
		},
	}
}

func encodeGraphQLMutationAwaitingPayload(
	intent session.PendingGraphQLMutationIntent,
	prompt string,
) (string, error) {
	encoded, err := json.Marshal(newGraphQLMutationAwaitingPayload(intent, prompt))
	if err != nil {
		return "", fmt.Errorf("encode awaiting payload: %w", err)
	}
	return string(encoded), nil
}

func newGraphQLMutationID(prefix string) (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return strings.TrimSpace(prefix) + "-" + hex.EncodeToString(raw[:]), nil
}

func encodeGraphQLMutationCommitPayload(
	intent session.PendingGraphQLMutationIntent,
	response any,
) (string, error) {
	return tooljson.Encode(map[string]any{
		"action":        graphQLMutationActionCommit,
		"intent_id":     intent.IntentID,
		"source":        intent.Source,
		"domain":        intent.Domain,
		"policy":        intent.PolicyName,
		"root_mutation": intent.RootMutation,
		"status":        intent.Status,
		"response":      response,
	})
}

func encodeGraphQLMutationDiscardPayload(
	intent session.PendingGraphQLMutationIntent,
) (string, error) {
	return tooljson.Encode(map[string]any{
		"action":        graphQLMutationActionDiscard,
		"intent_id":     intent.IntentID,
		"status":        intent.Status,
		"source":        intent.Source,
		"domain":        intent.Domain,
		"policy":        intent.PolicyName,
		"root_mutation": intent.RootMutation,
	})
}

func encodeGraphQLMutationListPayload(
	items []graphQLMutationPendingItem,
) (string, error) {
	return tooljson.Encode(map[string]any{
		"action":  graphQLMutationActionListPending,
		"intents": items,
	})
}

func graphqlMutationPendingItemFromIntent(
	intent session.PendingGraphQLMutationIntent,
) graphQLMutationPendingItem {
	return graphQLMutationPendingItem{
		IntentID:     intent.IntentID,
		Source:       intent.Source,
		Domain:       intent.Domain,
		Policy:       intent.PolicyName,
		Status:       intent.Status,
		PreparedAt:   intent.PreparedAt.Format(time.RFC3339),
		ApprovedAt:   formatOptionalGraphQLMutationTime(intent.ApprovedAt),
		ExecutedAt:   formatOptionalGraphQLMutationTime(intent.ExecutedAt),
		Summary:      intent.Summary,
		RootMutation: intent.RootMutation,
	}
}

func formatOptionalGraphQLMutationTime(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return at.UTC().Format(time.RFC3339)
}

func buildGraphQLMutationSummary(
	source string,
	domain string,
	policy string,
	rootMutation string,
	variables map[string]any,
) string {
	parts := []string{
		"source=" + source,
		"domain=" + domain,
		"policy=" + policy,
		"root_mutation=" + rootMutation,
		"variables=" + summarizeGraphQLMutationVariables(variables),
	}
	return strings.Join(parts, " | ")
}

func buildGraphQLMutationApprovalPrompt(
	intent session.PendingGraphQLMutationIntent,
) string {
	lines := []string{
		"Approve this GraphQL mutation?",
		"source: " + intent.Source,
		"domain: " + intent.Domain,
		"policy: " + intent.PolicyName,
		"root mutation: " + intent.RootMutation,
		"variables: " + summarizeGraphQLMutationVariables(intent.Variables),
		"risk: this will execute a write operation exactly as prepared if approved and committed.",
	}
	return strings.Join(lines, "\n")
}

func summarizeGraphQLMutationVariables(raw map[string]any) string {
	if len(raw) == 0 {
		return "{}"
	}
	sanitized := sanitizeGraphQLMutationValue(raw)
	encoded, err := json.Marshal(sanitized)
	if err != nil {
		return "{invalid}"
	}
	return truncateGraphQLMutationText(string(encoded), graphQLMutationSummaryMaxChars)
}

func sanitizeGraphQLMutationValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSensitiveGraphQLMutationKey(key) {
				out[key] = "***"
				continue
			}
			out[key] = sanitizeGraphQLMutationValue(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizeGraphQLMutationValue(item))
		}
		return out
	case string:
		return truncateGraphQLMutationText(typed, graphQLMutationValueMaxChars)
	default:
		return value
	}
}

func cloneGraphQLMutationVariables(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(encoded, &out); err != nil {
		return nil
	}
	return out
}

func truncateGraphQLMutationText(text string, maxChars int) string {
	if maxChars <= 0 || len(text) <= maxChars {
		return text
	}
	return text[:maxChars] + "..."
}

func isSensitiveGraphQLMutationKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	for _, candidate := range []string{
		"token", "secret", "password", "authorization", "cookie", "session", "api_key",
	} {
		if strings.Contains(lower, candidate) {
			return true
		}
	}
	return false
}
