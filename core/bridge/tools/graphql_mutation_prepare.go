package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/session"
	"ghost-os/bridge/tools/internal/graphqlschema"
)

func (t *GraphQLMutationTool) prepare(
	ctx context.Context,
	args graphQLMutationArgs,
	traceID string,
) (string, error) {
	startedAt := time.Now()
	logIntent := session.PendingGraphQLMutationIntent{
		Source: args.Source,
		Domain: args.Domain,
	}
	source, prepared, intent, prompt, err := t.prepareMutationIntent(ctx, args, traceID)
	if source != nil {
		logIntent.Source = source.Name
	}
	if prepared.Policy.Name != "" {
		logIntent.PolicyName = prepared.Policy.Name
		logIntent.RootMutation = prepared.Policy.RootMutation
	}
	if intent.IntentID != "" {
		logIntent = intent
	}
	logGraphQLMutationPrepare(traceID, logIntent, prepared.Summary, time.Since(startedAt), err)
	if err != nil {
		return "", err
	}
	return encodeGraphQLMutationAwaitingPayload(intent, prompt)
}

func (t *GraphQLMutationTool) prepareMutationIntent(
	ctx context.Context,
	args graphQLMutationArgs,
	traceID string,
) (
	*graphqlschema.Source,
	graphQLPreparedMutation,
	session.PendingGraphQLMutationIntent,
	string,
	error,
) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return nil, graphQLPreparedMutation{}, session.PendingGraphQLMutationIntent{}, "", fmt.Errorf("graphql_mutation requires an active session")
	}
	toolCallID := ToolCallIDFromContext(ctx)
	if toolCallID == "" {
		return nil, graphQLPreparedMutation{}, session.PendingGraphQLMutationIntent{}, "", fmt.Errorf("graphql_mutation requires tool call id in context")
	}
	if strings.TrimSpace(args.Mutation) == "" {
		return nil, graphQLPreparedMutation{}, session.PendingGraphQLMutationIntent{}, "", fmt.Errorf("mutation is required for action=prepare")
	}
	if strings.TrimSpace(args.Domain) == "" {
		return nil, graphQLPreparedMutation{}, session.PendingGraphQLMutationIntent{}, "", fmt.Errorf("domain is required for action=prepare")
	}
	source, err := t.registry.resolveSource(args.Source)
	if err != nil {
		return nil, graphQLPreparedMutation{}, session.PendingGraphQLMutationIntent{}, "", err
	}
	prepared, err := prepareGraphQLMutationDocument(
		args.Mutation,
		args.OperationName,
		source,
		args.Domain,
		t.registry,
	)
	if err != nil {
		return source, graphQLPreparedMutation{}, session.PendingGraphQLMutationIntent{}, "", err
	}
	intent, prompt, err := buildPendingGraphQLMutationIntent(
		traceID,
		toolCallID,
		prepared,
		args,
	)
	if err != nil {
		return source, prepared, session.PendingGraphQLMutationIntent{}, "", err
	}
	sess.StorePendingGraphQLMutationIntent(intent)
	sess.AddPendingQuestion(intent.QuestionID, session.PendingHumanQuestion{
		Prompt:        prompt,
		SelectionMode: session.HumanQuestionSelectionSingle,
		Options: []session.HumanQuestionOption{
			{Label: graphQLMutationApprovalApprove},
			{Label: graphQLMutationApprovalReject},
			{Label: graphQLMutationApprovalEdit, AllowCustom: true},
		},
		ToolName:   "graphql_mutation",
		ToolCallID: toolCallID,
		TraceID:    strings.TrimSpace(traceID),
		CreatedAt:  time.Now().UTC(),
	})
	return source, prepared, intent, prompt, nil
}

func buildPendingGraphQLMutationIntent(
	traceID string,
	toolCallID string,
	prepared graphQLPreparedMutation,
	args graphQLMutationArgs,
) (session.PendingGraphQLMutationIntent, string, error) {
	intentID, err := newGraphQLMutationID("intent")
	if err != nil {
		return session.PendingGraphQLMutationIntent{}, "", fmt.Errorf("generate intent id: %w", err)
	}
	questionID, err := newGraphQLMutationID("q")
	if err != nil {
		return session.PendingGraphQLMutationIntent{}, "", fmt.Errorf("generate question id: %w", err)
	}
	frozen, err := freezeGraphQLMutationRequest(prepared, args)
	if err != nil {
		return session.PendingGraphQLMutationIntent{}, "", err
	}
	intent := session.PendingGraphQLMutationIntent{
		IntentID:                intentID,
		Source:                  prepared.Source.Name,
		Domain:                  prepared.Domain,
		PolicyName:              prepared.Policy.Name,
		RootMutation:            prepared.Policy.RootMutation,
		IdempotencyMode:         prepared.Policy.IdempotencyMode,
		IdempotencyHeader:       prepared.Policy.IdempotencyHeader,
		IdempotencyVariablePath: prepared.Policy.IdempotencyVariablePath,
		OperationName:           args.OperationName,
		Query:                   prepared.MutationDoc,
		Variables:               frozen.Variables,
		DeliveryKey:             frozen.DeliveryKey,
		RequestHash:             frozen.RequestHash,
		QuestionID:              questionID,
		ToolCallID:              toolCallID,
		TraceID:                 strings.TrimSpace(traceID),
		PreparedAt:              time.Now().UTC(),
		Status:                  session.GraphQLMutationIntentPendingApproval,
		Summary: buildGraphQLMutationSummary(
			prepared.Source.Name,
			prepared.Domain,
			prepared.Policy.Name,
			prepared.Policy.RootMutation,
			frozen.Variables,
		),
	}
	return intent, buildGraphQLMutationApprovalPrompt(intent), nil
}
