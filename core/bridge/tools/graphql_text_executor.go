package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vektah/gqlparser/v2/ast"

	"ghost-os/bridge/session"
	"ghost-os/bridge/tools/internal/graphqlschema"
)

const (
	GraphQLTextMutationToolName = "graphql_text_mutation"
	graphQLTextSourceName       = "graphql_text.graphql"
)

// GraphQLTextExecutionResult 描述文本 GraphQL 的执行结果。
type GraphQLTextExecutionResult struct {
	Recognized bool
	Output     string
	Meta       ExecuteMeta
}

// GraphQLTextExecutor 执行模型直接输出的 GraphQL 文本。
type GraphQLTextExecutor interface {
	Execute(ctx context.Context, text string, traceID string) (GraphQLTextExecutionResult, error)
}

type graphQLTextExecutor struct {
	registry     *GraphQLSourceRegistry
	queryTool    *GraphQLQueryTool
	mutationTool *GraphQLMutationTool
}

func NewGraphQLTextExecutor(registry *GraphQLSourceRegistry) GraphQLTextExecutor {
	return &graphQLTextExecutor{
		registry:     registry,
		queryTool:    NewGraphQLQueryTool(registry).(*GraphQLQueryTool),
		mutationTool: NewGraphQLMutationTool(registry).(*GraphQLMutationTool),
	}
}

func (e *graphQLTextExecutor) Execute(
	ctx context.Context,
	text string,
	traceID string,
) (GraphQLTextExecutionResult, error) {
	documentText, looksLikeGraphQL := normalizeGraphQLTextDocument(text)
	if !looksLikeGraphQL {
		return GraphQLTextExecutionResult{}, nil
	}
	operation, parseErr := parseSingleGraphQLOperation(documentText)
	if parseErr != nil {
		return GraphQLTextExecutionResult{Recognized: true}, parseErr
	}
	source, err := e.registry.resolveSource("")
	if err != nil {
		return GraphQLTextExecutionResult{Recognized: true}, err
	}
	switch operation {
	case ast.Query:
		return e.executeQuery(ctx, source, documentText, traceID)
	case ast.Mutation:
		return e.executeMutation(ctx, source, documentText, traceID, resolveGraphQLTextToolCallID(ctx))
	default:
		return GraphQLTextExecutionResult{Recognized: true}, fmt.Errorf("graphql text executor only supports query or mutation")
	}
}

func (e *graphQLTextExecutor) executeQuery(
	ctx context.Context,
	source *graphqlschema.Source,
	documentText string,
	traceID string,
) (GraphQLTextExecutionResult, error) {
	startedAt := time.Now()
	summary, err := validateGraphQLQueryDocument(documentText, "", source, "")
	if err != nil {
		logGraphQLQuery(traceID, sourceName(source), "", summary, 0, time.Since(startedAt), err)
		return GraphQLTextExecutionResult{Recognized: true}, err
	}
	domain, err := inferQueryDomain(source, summary.RootFields)
	if err != nil {
		logGraphQLQuery(traceID, sourceName(source), "", summary, 0, time.Since(startedAt), err)
		return GraphQLTextExecutionResult{Recognized: true}, err
	}
	if domain != "" {
		summary, err = validateGraphQLQueryDocument(documentText, "", source, domain)
		if err != nil {
			logGraphQLQuery(traceID, source.Name, domain, summary, 0, time.Since(startedAt), err)
			return GraphQLTextExecutionResult{Recognized: true}, err
		}
	}
	body, err := e.queryTool.executeRequest(ctx, source, graphQLQueryArgs{Query: documentText})
	if err != nil {
		logGraphQLQuery(traceID, source.Name, domain, summary, 0, time.Since(startedAt), err)
		return GraphQLTextExecutionResult{Recognized: true}, err
	}
	if err := validateGraphQLResponseBody(body); err != nil {
		logGraphQLQuery(traceID, source.Name, domain, summary, len(body), time.Since(startedAt), err)
		return GraphQLTextExecutionResult{Recognized: true}, err
	}
	logGraphQLQuery(traceID, source.Name, domain, summary, len(body), time.Since(startedAt), nil)
	return GraphQLTextExecutionResult{
		Recognized: true,
		Output:     string(body),
	}, nil
}

func (e *graphQLTextExecutor) executeMutation(
	ctx context.Context,
	source *graphqlschema.Source,
	documentText string,
	traceID string,
	toolCallID string,
) (GraphQLTextExecutionResult, error) {
	domain, err := inferMutationDomain(e.registry, source, documentText)
	if err != nil {
		return GraphQLTextExecutionResult{Recognized: true}, err
	}
	args := graphQLMutationArgs{
		Source:   source.Name,
		Domain:   domain,
		Mutation: documentText,
	}
	prepared, err := prepareGraphQLMutationDocument(
		documentText,
		"",
		source,
		domain,
		e.registry,
	)
	if err != nil {
		return GraphQLTextExecutionResult{Recognized: true}, err
	}
	if prepared.Policy.ApprovalRequired {
		return e.prepareAwaitingMutationApproval(ctx, traceID, args, prepared, toolCallID)
	}
	output, err := e.executeMutationImmediately(ctx, traceID, args, prepared, toolCallID)
	if err != nil {
		return GraphQLTextExecutionResult{Recognized: true}, err
	}
	return GraphQLTextExecutionResult{
		Recognized: true,
		Output:     output,
	}, nil
}

func (e *graphQLTextExecutor) prepareAwaitingMutationApproval(
	ctx context.Context,
	traceID string,
	args graphQLMutationArgs,
	prepared graphQLPreparedMutation,
	toolCallID string,
) (GraphQLTextExecutionResult, error) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return GraphQLTextExecutionResult{Recognized: true}, fmt.Errorf("graphql text mutation requires an active session")
	}
	intent, prompt, err := buildPendingGraphQLMutationIntent(traceID, toolCallID, prepared, args)
	if err != nil {
		return GraphQLTextExecutionResult{Recognized: true}, err
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
		ToolName:   GraphQLTextMutationToolName,
		ToolCallID: toolCallID,
		TraceID:    strings.TrimSpace(traceID),
		CreatedAt:  time.Now().UTC(),
	})
	output, err := encodeGraphQLMutationAwaitingPayload(intent, prompt)
	if err != nil {
		return GraphQLTextExecutionResult{Recognized: true}, err
	}
	return GraphQLTextExecutionResult{
		Recognized: true,
		Output:     output,
		Meta:       interpretGraphQLMutationAwaitingResult(output),
	}, nil
}

func (e *graphQLTextExecutor) executeMutationImmediately(
	ctx context.Context,
	traceID string,
	args graphQLMutationArgs,
	prepared graphQLPreparedMutation,
	toolCallID string,
) (string, error) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return "", fmt.Errorf("graphql text mutation requires an active session")
	}
	intent, err := buildApprovedGraphQLTextIntent(traceID, toolCallID, args, prepared)
	if err != nil {
		return "", err
	}
	sess.StorePendingGraphQLMutationIntent(intent)
	argsJSON, err := json.Marshal(map[string]any{
		"action":    graphQLMutationActionCommit,
		"intent_id": intent.IntentID,
	})
	if err != nil {
		return "", fmt.Errorf("encode graphql commit args: %w", err)
	}
	return e.mutationTool.Execute(ctx, argsJSON, traceID)
}

func buildApprovedGraphQLTextIntent(
	traceID string,
	toolCallID string,
	args graphQLMutationArgs,
	prepared graphQLPreparedMutation,
) (session.PendingGraphQLMutationIntent, error) {
	intentID, err := newGraphQLMutationID("intent")
	if err != nil {
		return session.PendingGraphQLMutationIntent{}, fmt.Errorf("generate intent id: %w", err)
	}
	frozen, err := freezeGraphQLMutationRequest(prepared, args)
	if err != nil {
		return session.PendingGraphQLMutationIntent{}, err
	}
	now := time.Now().UTC()
	return session.PendingGraphQLMutationIntent{
		IntentID:                intentID,
		Source:                  prepared.Source.Name,
		Domain:                  prepared.Domain,
		PolicyName:              prepared.Policy.Name,
		RootMutation:            prepared.Policy.RootMutation,
		IdempotencyMode:         prepared.Policy.IdempotencyMode,
		IdempotencyHeader:       prepared.Policy.IdempotencyHeader,
		IdempotencyVariablePath: prepared.Policy.IdempotencyVariablePath,
		Query:                   prepared.MutationDoc,
		Variables:               frozen.Variables,
		DeliveryKey:             frozen.DeliveryKey,
		RequestHash:             frozen.RequestHash,
		ToolCallID:              toolCallID,
		TraceID:                 strings.TrimSpace(traceID),
		PreparedAt:              now,
		ApprovedAt:              now,
		Status:                  session.GraphQLMutationIntentApproved,
		CommitState:             session.GraphQLMutationCommitStateApproved,
		Summary: buildGraphQLMutationSummary(
			prepared.Source.Name,
			prepared.Domain,
			prepared.Policy.Name,
			prepared.Policy.RootMutation,
			frozen.Variables,
		),
	}, nil
}

func normalizeGraphQLTextDocument(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", false
	}
	if fenced := extractGraphQLFenceContent(trimmed); fenced != "" {
		return fenced, true
	}
	return trimmed, looksLikeGraphQLDocument(trimmed)
}

func parseSingleGraphQLOperation(text string) (ast.Operation, error) {
	document, err := parseGraphQLDocument(graphQLTextSourceName, text)
	if err != nil {
		return ast.Query, err
	}
	if len(document.Operations) != 1 {
		return ast.Query, fmt.Errorf("graphql text executor requires exactly one operation")
	}
	operation := document.Operations[0]
	if operation == nil {
		return ast.Query, fmt.Errorf("graphql text executor found an empty operation definition")
	}
	if operation.Operation != ast.Query && operation.Operation != ast.Mutation {
		return ast.Query, fmt.Errorf("graphql text executor only supports query or mutation operations")
	}
	return operation.Operation, nil
}
