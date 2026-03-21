package tools

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"ghost-os/bridge/session"
)

func TestGraphQLTextExecutorMutationRequiresApprovalWhenPolicyEnabled(t *testing.T) {
	registry := testGraphQLTextRegistry(t, true)
	executor := NewGraphQLTextExecutor(registry).(*graphQLTextExecutor)
	sess := session.NewSession("system")
	ctx := graphQLTextExecutionContext(sess)

	result, err := executor.Execute(
		ctx,
		`mutation { updateViewer(input: {id: "user-1"}) { ok } }`,
		"trace-graphql-text-await",
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Recognized || result.Meta.AwaitingHuman == nil {
		t.Fatalf("expected awaiting human result, got %+v", result)
	}
	if result.Meta.AwaitingHuman.QuestionID == "" {
		t.Fatalf("expected question id in awaiting result: %+v", result.Meta.AwaitingHuman)
	}
	intents := sess.PendingGraphQLMutationIntentsSnapshot()
	if len(intents) != 1 {
		t.Fatalf("expected one pending intent, got %+v", intents)
	}
	question := sess.PendingQuestions[intents[0].QuestionID]
	if question.ToolName != GraphQLTextMutationToolName {
		t.Fatalf("expected %q question, got %+v", GraphQLTextMutationToolName, question)
	}
}

func TestGraphQLTextExecutorMutationAutoCommitsWhenApprovalDisabled(t *testing.T) {
	registry := testGraphQLTextRegistry(t, false)
	executor := NewGraphQLTextExecutor(registry).(*graphQLTextExecutor)
	sess := session.NewSession("system")
	ctx := graphQLTextExecutionContext(sess)
	executor.mutationTool.httpClient = &http.Client{
		Transport: graphQLRoundTripper(func(req *http.Request) (*http.Response, error) {
			return newGraphQLResponse(http.StatusOK, `{"data":{"updateViewer":{"ok":true}}}`), nil
		}),
	}

	result, err := executor.Execute(
		ctx,
		`mutation { updateViewer(input: {id: "user-1"}) { ok } }`,
		"trace-graphql-text-auto-commit",
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Recognized {
		t.Fatalf("expected recognized graphql mutation, got %+v", result)
	}
	if !strings.Contains(result.Output, `"status":"executed"`) {
		t.Fatalf("expected executed commit payload, got %s", result.Output)
	}
	intents := sess.PendingGraphQLMutationIntentsSnapshot()
	if len(intents) != 1 || intents[0].Status != session.GraphQLMutationIntentExecuted {
		t.Fatalf("expected one executed intent, got %+v", intents)
	}
	if len(sess.PendingQuestions) != 0 {
		t.Fatalf("expected no pending question for auto-commit mutation, got %+v", sess.PendingQuestions)
	}
}

func testGraphQLTextRegistry(t *testing.T, approvalRequired bool) *GraphQLSourceRegistry {
	t.Helper()
	return testGraphQLRegistry(t, GraphQLRegistryConfig{
		DefaultSource: "crm",
		Sources: []GraphQLSourceConfig{{
			Name:             "crm",
			Endpoint:         "https://crm.test/query",
			SchemaPath:       writeGraphQLSchema(t, "crm-text"),
			TimeoutMS:        3000,
			MaxResponseBytes: 4096,
			MaxDepth:         6,
			MaxFields:        16,
			MaxRootFields:    2,
			MaxFragments:     4,
			Domains: []GraphQLDomainConfig{{
				Name:        "people",
				RootQueries: []string{"viewer"},
				Types:       []string{"Viewer", "MutationPayload"},
			}},
		}},
		MutationPolicies: []GraphQLMutationPolicyConfig{{
			Name:              "update_viewer",
			Source:            "crm",
			Domain:            "people",
			RootMutation:      "updateViewer",
			ApprovalRequired:  approvalRequired,
			IdempotencyMode:   graphQLMutationIdempotencyModeHeader,
			IdempotencyHeader: "Idempotency-Key",
			MaxDepth:          4,
			MaxFields:         8,
			MaxRootFields:     1,
			MaxFragments:      2,
		}},
	})
}

func graphQLTextExecutionContext(sess *session.Session) context.Context {
	ctx := WithSession(context.Background(), sess)
	ctx = WithSessionCheckpoint(ctx, graphQLMutationCheckpointFunc(func(*session.Session) error {
		return nil
	}))
	return ctx
}
