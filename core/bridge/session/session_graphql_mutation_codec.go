package session

import (
	"encoding/json"
	"strings"
	"time"
)

func normalizePendingGraphQLMutationIntent(intent PendingGraphQLMutationIntent) PendingGraphQLMutationIntent {
	normalized := PendingGraphQLMutationIntent{
		IntentID:                strings.TrimSpace(intent.IntentID),
		Source:                  strings.TrimSpace(intent.Source),
		Domain:                  strings.TrimSpace(intent.Domain),
		PolicyName:              strings.TrimSpace(intent.PolicyName),
		RootMutation:            strings.TrimSpace(intent.RootMutation),
		IdempotencyMode:         strings.TrimSpace(intent.IdempotencyMode),
		IdempotencyHeader:       strings.TrimSpace(intent.IdempotencyHeader),
		IdempotencyVariablePath: strings.TrimSpace(intent.IdempotencyVariablePath),
		OperationName:           strings.TrimSpace(intent.OperationName),
		Query:                   strings.TrimSpace(intent.Query),
		Variables:               cloneJSONMap(intent.Variables),
		DeliveryKey:             strings.TrimSpace(intent.DeliveryKey),
		RequestHash:             strings.TrimSpace(intent.RequestHash),
		CommitState:             strings.TrimSpace(intent.CommitState),
		AttemptCount:            intent.AttemptCount,
		LastAttemptAt:           intent.LastAttemptAt.UTC(),
		LastError:               strings.TrimSpace(intent.LastError),
		ResponseHash:            strings.TrimSpace(intent.ResponseHash),
		ResponseBytes:           intent.ResponseBytes,
		Receipts:                cloneGraphQLMutationReceipts(intent.Receipts),
		QuestionID:              strings.TrimSpace(intent.QuestionID),
		ToolCallID:              strings.TrimSpace(intent.ToolCallID),
		TraceID:                 strings.TrimSpace(intent.TraceID),
		PreparedAt:              intent.PreparedAt.UTC(),
		ApprovedAt:              intent.ApprovedAt.UTC(),
		ExecutedAt:              intent.ExecutedAt.UTC(),
		Status:                  strings.TrimSpace(intent.Status),
		HumanAnswer:             strings.TrimSpace(intent.HumanAnswer),
		Summary:                 strings.TrimSpace(intent.Summary),
	}
	if normalized.CommitState == "" {
		normalized.CommitState = graphQLMutationCommitStateFromStatus(normalized.Status)
	}
	return normalized
}

func clonePendingGraphQLMutationIntent(intent PendingGraphQLMutationIntent) PendingGraphQLMutationIntent {
	return normalizePendingGraphQLMutationIntent(intent)
}

func cloneJSONMap(raw map[string]any) map[string]any {
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

func graphQLMutationCommitStateFromStatus(status string) string {
	switch strings.TrimSpace(status) {
	case GraphQLMutationIntentApproved,
		GraphQLMutationIntentCommitting,
		GraphQLMutationIntentDeliveryUnknown,
		GraphQLMutationIntentExecuted,
		GraphQLMutationIntentDiscarded,
		GraphQLMutationIntentExpired:
		return strings.TrimSpace(status)
	default:
		return ""
	}
}

func normalizeGraphQLMutationTime(at time.Time) time.Time {
	if at.IsZero() {
		return time.Now().UTC()
	}
	return at.UTC()
}
