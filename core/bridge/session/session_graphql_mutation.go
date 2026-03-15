package session

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

const (
	GraphQLMutationIntentPendingApproval = "pending_approval"
	GraphQLMutationIntentApproved        = "approved"
	GraphQLMutationIntentRejected        = "rejected"
	GraphQLMutationIntentExecuted        = "executed"
	GraphQLMutationIntentDiscarded       = "discarded"
	GraphQLMutationIntentExpired         = "expired"
)

type PendingGraphQLMutationIntent struct {
	IntentID      string         `json:"intent_id"`
	Source        string         `json:"source"`
	Domain        string         `json:"domain"`
	PolicyName    string         `json:"policy_name"`
	RootMutation  string         `json:"root_mutation"`
	OperationName string         `json:"operation_name,omitempty"`
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables,omitempty"`
	QuestionID    string         `json:"question_id"`
	ToolCallID    string         `json:"tool_call_id"`
	TraceID       string         `json:"trace_id"`
	PreparedAt    time.Time      `json:"prepared_at"`
	ApprovedAt    time.Time      `json:"approved_at,omitempty"`
	ExecutedAt    time.Time      `json:"executed_at,omitempty"`
	Status        string         `json:"status"`
	HumanAnswer   string         `json:"human_answer,omitempty"`
	Summary       string         `json:"summary,omitempty"`
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
		intent.ExecutedAt = normalizeGraphQLMutationTime(at)
		return intent
	})
}

func (s *Session) MarkPendingGraphQLMutationIntentDiscarded(intentID string) bool {
	return s.updatePendingGraphQLMutationIntent(intentID, func(intent PendingGraphQLMutationIntent) PendingGraphQLMutationIntent {
		intent.Status = GraphQLMutationIntentDiscarded
		return intent
	})
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

func (s *Session) applyToolSpecificHumanAnswer(
	questionID string,
	question PendingHumanQuestion,
	answer string,
) {
	if strings.TrimSpace(question.ToolName) != "graphql_mutation" {
		return
	}
	s.applyPendingGraphQLMutationAnswer(questionID, answer)
}

func (s *Session) handleRemovedPendingQuestion(questionID string, question PendingHumanQuestion) {
	if strings.TrimSpace(question.ToolName) != "graphql_mutation" {
		return
	}
	intent, ok := s.PendingGraphQLMutationIntentByQuestionID(questionID)
	if !ok || intent.ExecutedAt != (time.Time{}) {
		return
	}
	_ = s.MarkPendingGraphQLMutationIntentDiscarded(intent.IntentID)
}

func (s *Session) applyPendingGraphQLMutationAnswer(questionID string, answer string) {
	if s == nil || len(s.PendingGraphQLMutationIntents) == 0 {
		return
	}
	for intentID, intent := range s.PendingGraphQLMutationIntents {
		if intent.QuestionID != strings.TrimSpace(questionID) {
			continue
		}
		intent.HumanAnswer = strings.TrimSpace(answer)
		intent.Status = graphQLMutationIntentStatusFromAnswer(answer)
		if intent.Status == GraphQLMutationIntentApproved {
			intent.ApprovedAt = time.Now().UTC()
		}
		s.PendingGraphQLMutationIntents[intentID] = intent
		return
	}
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

func normalizePendingGraphQLMutationIntent(intent PendingGraphQLMutationIntent) PendingGraphQLMutationIntent {
	return PendingGraphQLMutationIntent{
		IntentID:      strings.TrimSpace(intent.IntentID),
		Source:        strings.TrimSpace(intent.Source),
		Domain:        strings.TrimSpace(intent.Domain),
		PolicyName:    strings.TrimSpace(intent.PolicyName),
		RootMutation:  strings.TrimSpace(intent.RootMutation),
		OperationName: strings.TrimSpace(intent.OperationName),
		Query:         strings.TrimSpace(intent.Query),
		Variables:     cloneJSONMap(intent.Variables),
		QuestionID:    strings.TrimSpace(intent.QuestionID),
		ToolCallID:    strings.TrimSpace(intent.ToolCallID),
		TraceID:       strings.TrimSpace(intent.TraceID),
		PreparedAt:    intent.PreparedAt.UTC(),
		ApprovedAt:    intent.ApprovedAt.UTC(),
		ExecutedAt:    intent.ExecutedAt.UTC(),
		Status:        strings.TrimSpace(intent.Status),
		HumanAnswer:   strings.TrimSpace(intent.HumanAnswer),
		Summary:       strings.TrimSpace(intent.Summary),
	}
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

func graphQLMutationIntentStatusFromAnswer(answer string) string {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "approve", "approved":
		return GraphQLMutationIntentApproved
	default:
		return GraphQLMutationIntentRejected
	}
}

func normalizeGraphQLMutationTime(at time.Time) time.Time {
	if at.IsZero() {
		return time.Now().UTC()
	}
	return at.UTC()
}
