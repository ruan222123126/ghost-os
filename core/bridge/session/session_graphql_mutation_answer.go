package session

import (
	"strings"
	"time"
)

const (
	graphQLMutationQuestionToolName     = "graphql_mutation"
	graphQLTextMutationQuestionToolName = "graphql_text_mutation"
)

func (s *Session) applyToolSpecificHumanAnswer(
	questionID string,
	question PendingHumanQuestion,
	answer string,
) {
	if !isGraphQLMutationQuestion(question.ToolName) {
		return
	}
	s.applyPendingGraphQLMutationAnswer(questionID, answer)
}

func (s *Session) handleRemovedPendingQuestion(questionID string, question PendingHumanQuestion) {
	if !isGraphQLMutationQuestion(question.ToolName) {
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
		intent.CommitState = graphQLMutationCommitStateFromStatus(intent.Status)
		if intent.Status == GraphQLMutationIntentApproved {
			intent.ApprovedAt = time.Now().UTC()
		}
		s.PendingGraphQLMutationIntents[intentID] = intent
		return
	}
}

func isGraphQLMutationQuestion(toolName string) bool {
	switch strings.TrimSpace(toolName) {
	case graphQLMutationQuestionToolName, graphQLTextMutationQuestionToolName:
		return true
	default:
		return false
	}
}

func graphQLMutationIntentStatusFromAnswer(answer string) string {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "approve", "approved":
		return GraphQLMutationIntentApproved
	default:
		return GraphQLMutationIntentRejected
	}
}
