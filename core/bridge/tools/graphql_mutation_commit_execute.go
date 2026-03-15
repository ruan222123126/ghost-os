package tools

import (
	"context"
	"time"

	"ghost-os/bridge/session"
)

type graphQLPreparedCommit struct {
	intent     session.PendingGraphQLMutationIntent
	committing session.PendingGraphQLMutationIntent
	source     graphQLCommitSource
	checkpoint SessionCheckpoint
	startedAt  time.Time
}

func (t *GraphQLMutationTool) commit(
	ctx context.Context,
	args graphQLMutationArgs,
	traceID string,
) (string, error) {
	return t.executeCommit(ctx, args, traceID, false)
}

func (t *GraphQLMutationTool) retryCommit(
	ctx context.Context,
	args graphQLMutationArgs,
	traceID string,
) (string, error) {
	return t.executeCommit(ctx, args, traceID, true)
}

func (t *GraphQLMutationTool) executeCommit(
	ctx context.Context,
	args graphQLMutationArgs,
	traceID string,
	retry bool,
) (string, error) {
	prepared, err := t.prepareGraphQLMutationCommit(ctx, args, traceID, retry)
	if err != nil {
		return "", err
	}
	result, err := t.dispatchGraphQLMutationCommit(ctx, traceID, prepared)
	if err != nil {
		return "", err
	}
	return t.finishCommitExecuted(
		ctx,
		traceID,
		prepared.checkpoint,
		prepared.intent,
		prepared.committing,
		prepared.startedAt,
		result,
		retry,
	)
}

func (t *GraphQLMutationTool) dispatchGraphQLMutationCommit(
	ctx context.Context,
	traceID string,
	prepared graphQLPreparedCommit,
) (graphQLHTTPResult, error) {
	headers, err := injectGraphQLMutationRequestHeaders(
		prepared.source.raw.Headers,
		prepared.committing,
	)
	if err != nil {
		_, finishErr := t.finishCommitWithoutDispatch(
			ctx,
			traceID,
			prepared.checkpoint,
			prepared.intent,
			prepared.committing.AttemptCount,
			prepared.startedAt,
			err,
		)
		return graphQLHTTPResult{}, finishErr
	}
	result, err := executeGraphQLMutationCommitRequest(ctx, t, prepared, headers)
	if err != nil {
		_, finishErr := t.finishCommitAfterDispatch(
			ctx,
			traceID,
			prepared.checkpoint,
			prepared.intent,
			prepared.committing.AttemptCount,
			prepared.startedAt,
			result,
			err,
		)
		return graphQLHTTPResult{}, finishErr
	}
	if err := validateGraphQLMutationCommitResult(result); err != nil {
		_, finishErr := t.finishCommitAfterDispatch(
			ctx,
			traceID,
			prepared.checkpoint,
			prepared.intent,
			prepared.committing.AttemptCount,
			prepared.startedAt,
			result,
			err,
		)
		return graphQLHTTPResult{}, finishErr
	}
	return result, nil
}

func (t *GraphQLMutationTool) prepareGraphQLMutationCommit(
	ctx context.Context,
	args graphQLMutationArgs,
	traceID string,
	retry bool,
) (graphQLPreparedCommit, error) {
	startedAt := time.Now().UTC()
	intent, source, checkpoint, err := t.commitIntent(ctx, args.IntentID, retry)
	if err != nil {
		logGraphQLMutationCommit(
			traceID,
			session.PendingGraphQLMutationIntent{IntentID: args.IntentID},
			nil,
			time.Since(startedAt),
			err,
		)
		return graphQLPreparedCommit{}, err
	}
	committing, err := markGraphQLMutationIntentCommitting(ctx, intent, checkpoint, startedAt)
	if err != nil {
		logGraphQLMutationCommit(traceID, intent, nil, time.Since(startedAt), err)
		return graphQLPreparedCommit{}, err
	}
	return graphQLPreparedCommit{
		intent:     intent,
		committing: committing,
		source:     source,
		checkpoint: checkpoint,
		startedAt:  startedAt,
	}, nil
}

func executeGraphQLMutationCommitRequest(
	ctx context.Context,
	tool *GraphQLMutationTool,
	prepared graphQLPreparedCommit,
	headers map[string]string,
) (graphQLHTTPResult, error) {
	return executeGraphQLRequestDetailed(
		ctx,
		tool.httpClient,
		prepared.source.raw,
		graphQLRequestPayload{
			Query:         prepared.committing.Query,
			Variables:     cloneGraphQLMutationVariables(prepared.committing.Variables),
			OperationName: prepared.committing.OperationName,
		},
		graphQLRequestOptions{Headers: headers},
	)
}

func validateGraphQLMutationCommitResult(result graphQLHTTPResult) error {
	return validateGraphQLResponseBody(result.Body)
}

func (t *GraphQLMutationTool) finishCommitWithoutDispatch(
	ctx context.Context,
	traceID string,
	checkpoint SessionCheckpoint,
	intent session.PendingGraphQLMutationIntent,
	attempt int,
	startedAt time.Time,
	commitErr error,
) (string, error) {
	updated, err := persistGraphQLMutationIntentResult(
		ctx,
		intent,
		checkpoint,
		session.GraphQLMutationIntentApproved,
		buildGraphQLMutationReceipt(
			intent,
			attempt,
			session.GraphQLMutationIntentApproved,
			startedAt,
			time.Now().UTC(),
			nil,
			0,
			commitErr,
		),
	)
	logGraphQLMutationCommit(
		traceID,
		updated,
		lastGraphQLMutationReceipt(updated.Receipts),
		time.Since(startedAt),
		commitErr,
	)
	if err != nil {
		return "", err
	}
	return "", commitErr
}

func (t *GraphQLMutationTool) finishCommitAfterDispatch(
	ctx context.Context,
	traceID string,
	checkpoint SessionCheckpoint,
	intent session.PendingGraphQLMutationIntent,
	attempt int,
	startedAt time.Time,
	result graphQLHTTPResult,
	commitErr error,
) (string, error) {
	updated, err := persistGraphQLMutationIntentResult(
		ctx,
		intent,
		checkpoint,
		session.GraphQLMutationIntentDeliveryUnknown,
		buildGraphQLMutationReceipt(
			intent,
			attempt,
			session.GraphQLMutationIntentDeliveryUnknown,
			startedAt,
			time.Now().UTC(),
			result.Body,
			result.HTTPStatus,
			commitErr,
		),
	)
	logGraphQLMutationCommit(
		traceID,
		updated,
		lastGraphQLMutationReceipt(updated.Receipts),
		time.Since(startedAt),
		commitErr,
	)
	if err != nil {
		return "", err
	}
	return "", commitErr
}

func (t *GraphQLMutationTool) finishCommitExecuted(
	ctx context.Context,
	traceID string,
	checkpoint SessionCheckpoint,
	intent session.PendingGraphQLMutationIntent,
	committing session.PendingGraphQLMutationIntent,
	startedAt time.Time,
	result graphQLHTTPResult,
	retry bool,
) (string, error) {
	response, err := decodeGraphQLMutationResponse(result.Body)
	if err != nil {
		return t.finishCommitAfterDispatch(
			ctx,
			traceID,
			checkpoint,
			intent,
			committing.AttemptCount,
			startedAt,
			result,
			err,
		)
	}
	updated, err := persistGraphQLMutationIntentResult(
		ctx,
		intent,
		checkpoint,
		session.GraphQLMutationIntentExecuted,
		buildGraphQLMutationReceipt(
			committing,
			committing.AttemptCount,
			session.GraphQLMutationIntentExecuted,
			startedAt,
			time.Now().UTC(),
			result.Body,
			result.HTTPStatus,
			nil,
		),
	)
	logGraphQLMutationCommit(
		traceID,
		updated,
		lastGraphQLMutationReceipt(updated.Receipts),
		time.Since(startedAt),
		err,
	)
	if err != nil {
		return "", err
	}
	action := graphQLMutationActionCommit
	if retry {
		action = graphQLMutationActionRetryCommit
	}
	return encodeGraphQLMutationCommitResult(action, updated, response)
}
