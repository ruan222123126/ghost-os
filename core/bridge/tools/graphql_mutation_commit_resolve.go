package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/session"
	"ghost-os/bridge/tools/internal/graphqlschema"
)

type graphQLCommitSource struct {
	raw *graphqlschema.Source
}

func (t *GraphQLMutationTool) commitIntent(
	ctx context.Context,
	intentID string,
	retry bool,
) (
	session.PendingGraphQLMutationIntent,
	graphQLCommitSource,
	SessionCheckpoint,
	error,
) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return session.PendingGraphQLMutationIntent{}, graphQLCommitSource{}, nil, fmt.Errorf("graphql_mutation requires an active session")
	}
	checkpoint := SessionCheckpointFromContext(ctx)
	if checkpoint == nil {
		return session.PendingGraphQLMutationIntent{}, graphQLCommitSource{}, nil, fmt.Errorf("graphql_mutation requires session checkpoint persistence")
	}
	id := strings.TrimSpace(intentID)
	if id == "" {
		return session.PendingGraphQLMutationIntent{}, graphQLCommitSource{}, nil, fmt.Errorf("intent_id is required")
	}
	intent, ok := sess.PendingGraphQLMutationIntent(id)
	if !ok {
		return session.PendingGraphQLMutationIntent{}, graphQLCommitSource{}, nil, fmt.Errorf("graphql mutation intent %q was not found", id)
	}
	if err := validateGraphQLMutationIntentForCommit(intent, retry); err != nil {
		return session.PendingGraphQLMutationIntent{}, graphQLCommitSource{}, nil, err
	}
	if err := validateFrozenGraphQLMutationIntent(intent); err != nil {
		return session.PendingGraphQLMutationIntent{}, graphQLCommitSource{}, nil, err
	}
	source, err := t.registry.resolveSource(intent.Source)
	if err != nil {
		return session.PendingGraphQLMutationIntent{}, graphQLCommitSource{}, nil, err
	}
	return intent, graphQLCommitSource{raw: source}, checkpoint, nil
}

func validateGraphQLMutationIntentForCommit(
	intent session.PendingGraphQLMutationIntent,
	retry bool,
) error {
	if intent.Status == session.GraphQLMutationIntentExecuted {
		return fmt.Errorf("graphql mutation intent %q was already executed", intent.IntentID)
	}
	if retry {
		return validateGraphQLMutationRetryable(intent)
	}
	if intent.Status == session.GraphQLMutationIntentApproved {
		return nil
	}
	if intent.Status == session.GraphQLMutationIntentDeliveryUnknown {
		return fmt.Errorf(
			"graphql mutation intent %q is in delivery_unknown; use action=retry_commit",
			intent.IntentID,
		)
	}
	return fmt.Errorf("graphql mutation intent %q is not approved", intent.IntentID)
}

func validateGraphQLMutationRetryable(
	intent session.PendingGraphQLMutationIntent,
) error {
	switch intent.Status {
	case session.GraphQLMutationIntentApproved,
		session.GraphQLMutationIntentDeliveryUnknown:
		return nil
	default:
		return fmt.Errorf(
			"graphql mutation intent %q is not retryable for action=%s",
			intent.IntentID,
			graphQLMutationActionRetryCommit,
		)
	}
}

func validateFrozenGraphQLMutationIntent(
	intent session.PendingGraphQLMutationIntent,
) error {
	if strings.TrimSpace(intent.DeliveryKey) == "" {
		return fmt.Errorf("graphql mutation intent %q is missing delivery_key", intent.IntentID)
	}
	if strings.TrimSpace(intent.RequestHash) == "" {
		return fmt.Errorf("graphql mutation intent %q is missing request_hash", intent.IntentID)
	}
	switch strings.TrimSpace(intent.IdempotencyMode) {
	case graphQLMutationIdempotencyModeHeader:
		if strings.TrimSpace(intent.IdempotencyHeader) == "" {
			return fmt.Errorf("graphql mutation intent %q is missing idempotency_header", intent.IntentID)
		}
	case graphQLMutationIdempotencyModeVariablePath:
		if strings.TrimSpace(intent.IdempotencyVariablePath) == "" {
			return fmt.Errorf("graphql mutation intent %q is missing idempotency_variable_path", intent.IntentID)
		}
	default:
		return fmt.Errorf(
			"graphql mutation intent %q has unsupported idempotency_mode %q",
			intent.IntentID,
			intent.IdempotencyMode,
		)
	}
	return nil
}

func decodeGraphQLMutationResponse(body []byte) (any, error) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode graphql response: %w", err)
	}
	return payload, nil
}
