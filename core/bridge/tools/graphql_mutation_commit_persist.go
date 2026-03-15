package tools

import (
	"context"
	"fmt"
	"time"

	"ghost-os/bridge/session"
)

func markGraphQLMutationIntentCommitting(
	ctx context.Context,
	intent session.PendingGraphQLMutationIntent,
	checkpoint SessionCheckpoint,
	startedAt time.Time,
) (session.PendingGraphQLMutationIntent, error) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return session.PendingGraphQLMutationIntent{}, fmt.Errorf("graphql_mutation requires an active session")
	}
	updated := intent
	updated.Status = session.GraphQLMutationIntentCommitting
	updated.CommitState = session.GraphQLMutationCommitStateCommitting
	updated.AttemptCount = intent.AttemptCount + 1
	updated.LastAttemptAt = startedAt.UTC()
	updated.LastError = ""
	if !sess.ReplacePendingGraphQLMutationIntent(updated) {
		return session.PendingGraphQLMutationIntent{}, fmt.Errorf(
			"failed to mark graphql mutation intent %q committing",
			intent.IntentID,
		)
	}
	if err := checkpoint.Save(sess); err != nil {
		_ = revertGraphQLMutationCommittingFailure(sess, intent, updated, startedAt, err)
		return session.PendingGraphQLMutationIntent{}, fmt.Errorf(
			"persist graphql mutation intent %q committing state: %w",
			intent.IntentID,
			err,
		)
	}
	return updated, nil
}

func revertGraphQLMutationCommittingFailure(
	sess *session.Session,
	intent session.PendingGraphQLMutationIntent,
	committing session.PendingGraphQLMutationIntent,
	startedAt time.Time,
	err error,
) bool {
	if sess == nil {
		return false
	}
	return sess.ReplacePendingGraphQLMutationIntent(applyGraphQLMutationReceipt(
		intent,
		session.GraphQLMutationIntentApproved,
		buildGraphQLMutationReceipt(
			intent,
			committing.AttemptCount,
			session.GraphQLMutationIntentApproved,
			startedAt,
			time.Now().UTC(),
			nil,
			0,
			err,
		),
	))
}

func persistGraphQLMutationIntentResult(
	ctx context.Context,
	intent session.PendingGraphQLMutationIntent,
	checkpoint SessionCheckpoint,
	state string,
	receipt session.GraphQLMutationReceipt,
) (session.PendingGraphQLMutationIntent, error) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return session.PendingGraphQLMutationIntent{}, fmt.Errorf("graphql_mutation requires an active session")
	}
	updated := applyGraphQLMutationReceipt(intent, state, receipt)
	if !sess.ReplacePendingGraphQLMutationIntent(updated) {
		return session.PendingGraphQLMutationIntent{}, fmt.Errorf(
			"failed to persist graphql mutation intent %q state %q",
			intent.IntentID,
			state,
		)
	}
	if err := checkpoint.Save(sess); err != nil {
		return session.PendingGraphQLMutationIntent{}, fmt.Errorf(
			"persist graphql mutation intent %q state %q: %w",
			intent.IntentID,
			state,
			err,
		)
	}
	return updated, nil
}
