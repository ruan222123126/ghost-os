package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/session"
	"ghost-os/bridge/tools/internal/graphqlschema"
)

func (t *GraphQLMutationTool) commit(
	ctx context.Context,
	args graphQLMutationArgs,
	traceID string,
) (string, error) {
	startedAt := time.Now()
	intent, source, err := t.commitIntent(ctx, args.IntentID)
	if err != nil {
		logGraphQLMutationCommit(traceID, session.PendingGraphQLMutationIntent{
			IntentID: args.IntentID,
		}, 0, time.Since(startedAt), err)
		return "", err
	}
	body, err := executeGraphQLRequest(ctx, t.httpClient, source, graphQLRequestPayload{
		Query:         intent.Query,
		Variables:     cloneGraphQLMutationVariables(intent.Variables),
		OperationName: intent.OperationName,
	})
	if err != nil {
		logGraphQLMutationCommit(traceID, intent, 0, time.Since(startedAt), err)
		return "", err
	}
	if err := validateGraphQLResponseBody(body); err != nil {
		logGraphQLMutationCommit(traceID, intent, len(body), time.Since(startedAt), err)
		return "", err
	}
	if !SessionFromContext(ctx).MarkPendingGraphQLMutationIntentExecuted(intent.IntentID, time.Now().UTC()) {
		return "", fmt.Errorf("failed to mark graphql mutation intent %q executed", intent.IntentID)
	}
	intent.Status = session.GraphQLMutationIntentExecuted
	intent.ExecutedAt = time.Now().UTC()
	response, err := decodeGraphQLMutationResponse(body)
	logGraphQLMutationCommit(traceID, intent, len(body), time.Since(startedAt), err)
	if err != nil {
		return "", err
	}
	return encodeGraphQLMutationCommitPayload(intent, response)
}

func (t *GraphQLMutationTool) commitIntent(
	ctx context.Context,
	intentID string,
) (session.PendingGraphQLMutationIntent, *graphqlschema.Source, error) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return session.PendingGraphQLMutationIntent{}, nil, fmt.Errorf("graphql_mutation requires an active session")
	}
	id := strings.TrimSpace(intentID)
	if id == "" {
		return session.PendingGraphQLMutationIntent{}, nil, fmt.Errorf("intent_id is required for action=commit")
	}
	intent, ok := sess.PendingGraphQLMutationIntent(id)
	if !ok {
		return session.PendingGraphQLMutationIntent{}, nil, fmt.Errorf("graphql mutation intent %q was not found", id)
	}
	if intent.Status == session.GraphQLMutationIntentExecuted {
		return session.PendingGraphQLMutationIntent{}, nil, fmt.Errorf("graphql mutation intent %q was already executed", id)
	}
	if intent.Status != session.GraphQLMutationIntentApproved {
		return session.PendingGraphQLMutationIntent{}, nil, fmt.Errorf("graphql mutation intent %q is not approved", id)
	}
	source, err := t.registry.resolveSource(intent.Source)
	if err != nil {
		return session.PendingGraphQLMutationIntent{}, nil, err
	}
	return intent, source, nil
}

func decodeGraphQLMutationResponse(body []byte) (any, error) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode graphql response: %w", err)
	}
	return payload, nil
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
		session.GraphQLMutationIntentRejected:
		return true
	default:
		return false
	}
}
