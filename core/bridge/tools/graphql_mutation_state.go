package tools

import (
	"context"
	"fmt"
	"strings"

	"ghost-os/bridge/session"
	"ghost-os/bridge/tools/internal/tooljson"
)

func (t *GraphQLMutationTool) status(
	ctx context.Context,
	args graphQLMutationArgs,
) (string, error) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return "", fmt.Errorf("graphql_mutation requires an active session")
	}
	intent, err := requireGraphQLMutationIntent(sess, args.IntentID, graphQLMutationActionStatus)
	if err != nil {
		return "", err
	}
	return encodeGraphQLMutationStatusPayload(intent)
}

func (t *GraphQLMutationTool) discard(
	ctx context.Context,
	args graphQLMutationArgs,
	traceID string,
) (string, error) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return "", fmt.Errorf("graphql_mutation requires an active session")
	}
	intent, err := requireGraphQLMutationIntent(sess, args.IntentID, graphQLMutationActionDiscard)
	if err != nil {
		logGraphQLMutationDiscard(traceID, session.PendingGraphQLMutationIntent{
			IntentID: args.IntentID,
		}, err)
		return "", err
	}
	if !graphQLMutationIntentDiscardable(intent.Status) {
		err := fmt.Errorf("graphql mutation intent %q with status %q cannot be discarded", intent.IntentID, intent.Status)
		logGraphQLMutationDiscard(traceID, intent, err)
		return "", err
	}
	if !sess.MarkPendingGraphQLMutationIntentDiscarded(intent.IntentID) {
		err := fmt.Errorf("failed to discard graphql mutation intent %q", intent.IntentID)
		logGraphQLMutationDiscard(traceID, intent, err)
		return "", err
	}
	intent.Status = session.GraphQLMutationIntentDiscarded
	intent.CommitState = session.GraphQLMutationCommitStateDiscarded
	logGraphQLMutationDiscard(traceID, intent, nil)
	return encodeGraphQLMutationDiscardPayload(intent)
}

func (t *GraphQLMutationTool) listPending(ctx context.Context) (string, error) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return "", fmt.Errorf("graphql_mutation requires an active session")
	}
	intents := sess.PendingGraphQLMutationIntentsSnapshot()
	items := make([]graphQLMutationPendingItem, 0, len(intents))
	for _, intent := range intents {
		if !graphQLMutationIntentListable(intent.Status) {
			continue
		}
		items = append(items, graphqlMutationPendingItemFromIntent(intent))
	}
	return encodeGraphQLMutationListPayload(items)
}

func requireGraphQLMutationIntent(
	sess *session.Session,
	intentID string,
	action string,
) (session.PendingGraphQLMutationIntent, error) {
	if strings.TrimSpace(intentID) == "" {
		return session.PendingGraphQLMutationIntent{}, fmt.Errorf("intent_id is required for action=%s", action)
	}
	intent, ok := sess.PendingGraphQLMutationIntent(intentID)
	if !ok {
		return session.PendingGraphQLMutationIntent{}, fmt.Errorf("graphql mutation intent %q was not found", intentID)
	}
	return intent, nil
}

func graphQLMutationIntentDiscardable(status string) bool {
	switch strings.TrimSpace(status) {
	case session.GraphQLMutationIntentPendingApproval,
		session.GraphQLMutationIntentApproved,
		session.GraphQLMutationIntentRejected:
		return true
	default:
		return false
	}
}

func graphQLMutationIntentListable(status string) bool {
	switch strings.TrimSpace(status) {
	case session.GraphQLMutationIntentPendingApproval,
		session.GraphQLMutationIntentApproved,
		session.GraphQLMutationIntentCommitting,
		session.GraphQLMutationIntentDeliveryUnknown,
		session.GraphQLMutationIntentRejected:
		return true
	default:
		return false
	}
}

func encodeGraphQLMutationCommitResult(
	action string,
	intent session.PendingGraphQLMutationIntent,
	response any,
) (string, error) {
	return tooljson.Encode(map[string]any{
		"action":        action,
		"intent_id":     intent.IntentID,
		"source":        intent.Source,
		"domain":        intent.Domain,
		"policy":        intent.PolicyName,
		"root_mutation": intent.RootMutation,
		"status":        intent.Status,
		"commit_state":  intent.CommitState,
		"delivery_key":  intent.DeliveryKey,
		"request_hash":  intent.RequestHash,
		"attempt_count": intent.AttemptCount,
		"receipt":       lastGraphQLMutationReceipt(intent.Receipts),
		"response":      response,
	})
}

func encodeGraphQLMutationStatusPayload(
	intent session.PendingGraphQLMutationIntent,
) (string, error) {
	return tooljson.Encode(map[string]any{
		"action":          graphQLMutationActionStatus,
		"intent_id":       intent.IntentID,
		"source":          intent.Source,
		"domain":          intent.Domain,
		"policy":          intent.PolicyName,
		"root_mutation":   intent.RootMutation,
		"status":          intent.Status,
		"commit_state":    intent.CommitState,
		"delivery_key":    intent.DeliveryKey,
		"request_hash":    intent.RequestHash,
		"attempt_count":   intent.AttemptCount,
		"last_attempt_at": formatOptionalGraphQLMutationTime(intent.LastAttemptAt),
		"last_error":      strings.TrimSpace(intent.LastError),
		"receipt":         lastGraphQLMutationReceipt(intent.Receipts),
	})
}
