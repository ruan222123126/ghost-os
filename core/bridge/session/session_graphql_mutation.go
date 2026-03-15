package session

import (
	"sort"
	"strings"
	"time"
)

const (
	GraphQLMutationIntentPendingApproval = "pending_approval"
	GraphQLMutationIntentApproved        = GraphQLMutationCommitStateApproved
	GraphQLMutationIntentCommitting      = GraphQLMutationCommitStateCommitting
	GraphQLMutationIntentDeliveryUnknown = GraphQLMutationCommitStateDeliveryUnknown
	GraphQLMutationIntentRejected        = "rejected"
	GraphQLMutationIntentExecuted        = GraphQLMutationCommitStateExecuted
	GraphQLMutationIntentDiscarded       = GraphQLMutationCommitStateDiscarded
	GraphQLMutationIntentExpired         = GraphQLMutationCommitStateExpired
)

const (
	GraphQLMutationCommitStateApproved        = "approved"
	GraphQLMutationCommitStateCommitting      = "committing"
	GraphQLMutationCommitStateDeliveryUnknown = "delivery_unknown"
	GraphQLMutationCommitStateExecuted        = "executed"
	GraphQLMutationCommitStateDiscarded       = "discarded"
	GraphQLMutationCommitStateExpired         = "expired"
)

type PendingGraphQLMutationIntent struct {
	IntentID                string                   `json:"intent_id"`
	Source                  string                   `json:"source"`
	Domain                  string                   `json:"domain"`
	PolicyName              string                   `json:"policy_name"`
	RootMutation            string                   `json:"root_mutation"`
	IdempotencyMode         string                   `json:"idempotency_mode,omitempty"`
	IdempotencyHeader       string                   `json:"idempotency_header,omitempty"`
	IdempotencyVariablePath string                   `json:"idempotency_variable_path,omitempty"`
	OperationName           string                   `json:"operation_name,omitempty"`
	Query                   string                   `json:"query"`
	Variables               map[string]any           `json:"variables,omitempty"`
	DeliveryKey             string                   `json:"delivery_key,omitempty"`
	RequestHash             string                   `json:"request_hash,omitempty"`
	CommitState             string                   `json:"commit_state,omitempty"`
	AttemptCount            int                      `json:"attempt_count,omitempty"`
	LastAttemptAt           time.Time                `json:"last_attempt_at,omitempty"`
	LastError               string                   `json:"last_error,omitempty"`
	ResponseHash            string                   `json:"response_hash,omitempty"`
	ResponseBytes           int                      `json:"response_bytes,omitempty"`
	Receipts                []GraphQLMutationReceipt `json:"receipts,omitempty"`
	QuestionID              string                   `json:"question_id"`
	ToolCallID              string                   `json:"tool_call_id"`
	TraceID                 string                   `json:"trace_id"`
	PreparedAt              time.Time                `json:"prepared_at"`
	ApprovedAt              time.Time                `json:"approved_at,omitempty"`
	ExecutedAt              time.Time                `json:"executed_at,omitempty"`
	Status                  string                   `json:"status"`
	HumanAnswer             string                   `json:"human_answer,omitempty"`
	Summary                 string                   `json:"summary,omitempty"`
}

func (s *Session) StorePendingGraphQLMutationIntent(intent PendingGraphQLMutationIntent) {
	if s == nil {
		return
	}
	intentID := strings.TrimSpace(intent.IntentID)
	if intentID == "" {
		return
	}
	if s.PendingGraphQLMutationIntents == nil {
		s.PendingGraphQLMutationIntents = make(
			map[string]PendingGraphQLMutationIntent,
			2,
		)
	}
	normalized := normalizePendingGraphQLMutationIntent(intent)
	normalized.IntentID = intentID
	if normalized.PreparedAt.IsZero() {
		normalized.PreparedAt = time.Now().UTC()
	}
	if normalized.Status == "" {
		normalized.Status = GraphQLMutationIntentPendingApproval
	}
	s.PendingGraphQLMutationIntents[intentID] = normalized
	s.UpdatedAt = time.Now().UTC()
}

func (s *Session) PendingGraphQLMutationIntent(intentID string) (PendingGraphQLMutationIntent, bool) {
	if s == nil || len(s.PendingGraphQLMutationIntents) == 0 {
		return PendingGraphQLMutationIntent{}, false
	}
	intent, ok := s.PendingGraphQLMutationIntents[strings.TrimSpace(intentID)]
	if !ok {
		return PendingGraphQLMutationIntent{}, false
	}
	return clonePendingGraphQLMutationIntent(intent), true
}

func (s *Session) PendingGraphQLMutationIntentByQuestionID(
	questionID string,
) (PendingGraphQLMutationIntent, bool) {
	if s == nil || len(s.PendingGraphQLMutationIntents) == 0 {
		return PendingGraphQLMutationIntent{}, false
	}
	id := strings.TrimSpace(questionID)
	for _, intent := range s.PendingGraphQLMutationIntents {
		if intent.QuestionID == id {
			return clonePendingGraphQLMutationIntent(intent), true
		}
	}
	return PendingGraphQLMutationIntent{}, false
}

func (s *Session) MarkPendingGraphQLMutationIntentExecuted(intentID string, at time.Time) bool {
	return s.updatePendingGraphQLMutationIntent(intentID, func(intent PendingGraphQLMutationIntent) PendingGraphQLMutationIntent {
		intent.Status = GraphQLMutationIntentExecuted
		intent.CommitState = GraphQLMutationCommitStateExecuted
		intent.ExecutedAt = normalizeGraphQLMutationTime(at)
		return intent
	})
}

func (s *Session) MarkPendingGraphQLMutationIntentDiscarded(intentID string) bool {
	return s.updatePendingGraphQLMutationIntent(intentID, func(intent PendingGraphQLMutationIntent) PendingGraphQLMutationIntent {
		intent.Status = GraphQLMutationIntentDiscarded
		intent.CommitState = GraphQLMutationCommitStateDiscarded
		return intent
	})
}

func (s *Session) ReplacePendingGraphQLMutationIntent(
	intent PendingGraphQLMutationIntent,
) bool {
	if s == nil || len(s.PendingGraphQLMutationIntents) == 0 {
		return false
	}
	id := strings.TrimSpace(intent.IntentID)
	if id == "" {
		return false
	}
	if _, ok := s.PendingGraphQLMutationIntents[id]; !ok {
		return false
	}
	s.PendingGraphQLMutationIntents[id] = normalizePendingGraphQLMutationIntent(intent)
	s.UpdatedAt = time.Now().UTC()
	return true
}

func (s *Session) PendingGraphQLMutationIntentsSnapshot() []PendingGraphQLMutationIntent {
	if s == nil || len(s.PendingGraphQLMutationIntents) == 0 {
		return nil
	}
	out := make([]PendingGraphQLMutationIntent, 0, len(s.PendingGraphQLMutationIntents))
	for _, intent := range s.PendingGraphQLMutationIntents {
		out = append(out, clonePendingGraphQLMutationIntent(intent))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].PreparedAt.Before(out[j].PreparedAt)
	})
	return out
}

func (s *Session) updatePendingGraphQLMutationIntent(
	intentID string,
	update func(PendingGraphQLMutationIntent) PendingGraphQLMutationIntent,
) bool {
	if s == nil || len(s.PendingGraphQLMutationIntents) == 0 {
		return false
	}
	id := strings.TrimSpace(intentID)
	intent, ok := s.PendingGraphQLMutationIntents[id]
	if !ok {
		return false
	}
	s.PendingGraphQLMutationIntents[id] = normalizePendingGraphQLMutationIntent(update(intent))
	s.UpdatedAt = time.Now().UTC()
	return true
}
