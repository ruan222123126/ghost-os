package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"ghost-os/bridge/session"
)

func TestGraphQLMutationToolPrepareStoresPendingIntent(t *testing.T) {
	registry := testGraphQLMutationRegistry(t)
	tool := NewGraphQLMutationTool(registry).(*GraphQLMutationTool)
	sess := session.NewSession("system")

	output, err := tool.Execute(
		graphQLMutationContext(sess, "call-mutation-prepare"),
		json.RawMessage(`{
			"action":"prepare",
			"source":"crm",
			"domain":"people",
			"mutation":"mutation UpdateViewer($input: ViewerInput!) { updateViewer(input: $input) { ok viewer { id } } }",
			"variables":{"input":{"id":"user-1","token":"secret-token"}}
		}`),
		"trace-mutation-prepare",
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var payload graphQLMutationAwaitingPayload
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Status != "awaiting_human" || payload.IntentID == "" || payload.QuestionID == "" {
		t.Fatalf("unexpected awaiting payload: %+v", payload)
	}
	if !strings.Contains(payload.Prompt, "source: crm") || !strings.Contains(payload.Prompt, "policy: update_viewer") {
		t.Fatalf("expected approval prompt details, got %q", payload.Prompt)
	}
	if strings.Contains(payload.Prompt, "secret-token") {
		t.Fatalf("expected prompt variables to be redacted, got %q", payload.Prompt)
	}

	intent, ok := sess.PendingGraphQLMutationIntent(payload.IntentID)
	if !ok {
		t.Fatalf("expected stored intent for %q", payload.IntentID)
	}
	if intent.Status != session.GraphQLMutationIntentPendingApproval {
		t.Fatalf("unexpected intent status: %q", intent.Status)
	}
	if intent.RootMutation != "updateViewer" || intent.PolicyName != "update_viewer" {
		t.Fatalf("unexpected stored intent: %+v", intent)
	}
	if !sess.HasPendingQuestion(payload.QuestionID) {
		t.Fatalf("expected pending question %q", payload.QuestionID)
	}
	if question := sess.PendingQuestions[payload.QuestionID]; question.ToolName != "graphql_mutation" {
		t.Fatalf("expected graphql_mutation question, got %+v", question)
	}
}

func TestGraphQLMutationToolRejectsDisallowedOrInvalidPrepare(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
		tool func(*testing.T) *GraphQLMutationTool
	}{
		{
			name: "not allowlisted",
			args: `{
				"action":"prepare",
				"source":"crm",
				"domain":"people",
				"mutation":"mutation { archiveViewer(id: \"user-1\") { ok } }"
			}`,
			want: `not allowed`,
			tool: testGraphQLMutationTool,
		},
		{
			name: "multiple roots",
			args: `{
				"action":"prepare",
				"source":"crm",
				"domain":"people",
				"mutation":"mutation { updateViewer(input: {id: \"user-1\"}) { ok } archiveViewer(id: \"user-1\") { ok } }"
			}`,
			want: "exactly one root mutation field",
			tool: testGraphQLMutationTool,
		},
		{
			name: "wrong operation",
			args: `{
				"action":"prepare",
				"source":"crm",
				"domain":"people",
				"mutation":"query { viewer { id } }"
			}`,
			want: `only supports mutation operations`,
			tool: testGraphQLMutationTool,
		},
		{
			name: "introspection",
			args: `{
				"action":"prepare",
				"source":"crm",
				"domain":"people",
				"mutation":"mutation { __type(name: \"Viewer\") { name } }"
			}`,
			want: "introspection fields are not allowed",
			tool: testGraphQLMutationTool,
		},
		{
			name: "budget",
			args: `{
				"action":"prepare",
				"source":"crm",
				"domain":"people",
				"mutation":"mutation { updateViewer(input: {id: \"user-1\"}) { viewer { id } } }"
			}`,
			want: "max_depth=1",
			tool: testGraphQLMutationBudgetTool,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sess := session.NewSession("system")
			ctx := graphQLMutationContext(sess, "call-mutation-invalid")
			tool := tc.tool(t)
			_, err := tool.Execute(ctx, json.RawMessage(tc.args), "trace-mutation-invalid")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

func TestGraphQLMutationToolCommitUsesFrozenIntentAndOneShotGuard(t *testing.T) {
	registry := testGraphQLMutationRegistry(t)
	tool := NewGraphQLMutationTool(registry).(*GraphQLMutationTool)
	sess := session.NewSession("system")
	ctx := graphQLMutationContext(sess, "call-mutation-commit")

	_, err := tool.Execute(ctx, json.RawMessage(`{
		"action":"prepare",
		"source":"crm",
		"domain":"people",
		"mutation":"mutation ApprovedMutation($input: ViewerInput!) { updateViewer(input: $input) { ok } }",
		"variables":{"input":{"id":"user-1"}}
	}`), "trace-mutation-commit-prepare")
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	intent := onlyPendingGraphQLMutationIntent(t, sess)
	if !sess.SetHumanAnswer(intent.QuestionID, "Approve") {
		t.Fatalf("expected human approval to be accepted")
	}

	tool.httpClient = &http.Client{
		Transport: graphQLRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://crm.test/query" {
				t.Fatalf("unexpected endpoint: %s", req.URL.String())
			}
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if !strings.Contains(string(body), "ApprovedMutation") {
				t.Fatalf("expected frozen mutation document, got %s", string(body))
			}
			if strings.Contains(string(body), "archiveViewer") {
				t.Fatalf("commit should ignore rewritten args, got %s", string(body))
			}
			return newGraphQLResponse(http.StatusOK, `{"data":{"updateViewer":{"ok":true}}}`), nil
		}),
	}

	output, err := tool.Execute(ctx, json.RawMessage(`{
		"action":"commit",
		"intent_id":"`+intent.IntentID+`",
		"source":"billing",
		"domain":"orders",
		"mutation":"mutation { archiveViewer(id: \"user-1\") { ok } }"
	}`), "trace-mutation-commit")
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if !strings.Contains(output, `"status":"executed"`) {
		t.Fatalf("expected executed payload, got %s", output)
	}
	updated := onlyPendingGraphQLMutationIntent(t, sess)
	if updated.Status != session.GraphQLMutationIntentExecuted || updated.ExecutedAt.IsZero() {
		t.Fatalf("expected executed intent, got %+v", updated)
	}

	if _, err := tool.Execute(ctx, json.RawMessage(`{
		"action":"commit",
		"intent_id":"`+intent.IntentID+`"
	}`), "trace-mutation-commit-repeat"); err == nil || !strings.Contains(err.Error(), "already executed") {
		t.Fatalf("expected one-shot guard error, got %v", err)
	}
}

func TestGraphQLMutationToolRejectsUnapprovedCommitAndSupportsDiscardList(t *testing.T) {
	registry := testGraphQLMutationRegistry(t)
	tool := NewGraphQLMutationTool(registry).(*GraphQLMutationTool)
	sess := session.NewSession("system")
	ctx := graphQLMutationContext(sess, "call-mutation-state")

	_, err := tool.Execute(ctx, json.RawMessage(`{
		"action":"prepare",
		"source":"crm",
		"domain":"people",
		"mutation":"mutation { updateViewer(input: {id: \"user-1\"}) { ok } }"
	}`), "trace-mutation-state")
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	intent := onlyPendingGraphQLMutationIntent(t, sess)
	if !sess.SetHumanAnswer(intent.QuestionID, "Reject") {
		t.Fatalf("expected reject answer to be accepted")
	}

	if _, err := tool.Execute(ctx, json.RawMessage(`{
		"action":"commit",
		"intent_id":"`+intent.IntentID+`"
	}`), "trace-mutation-state-commit"); err == nil || !strings.Contains(err.Error(), "not approved") {
		t.Fatalf("expected not approved error, got %v", err)
	}

	listOutput, err := tool.Execute(ctx, json.RawMessage(`{"action":"list_pending"}`), "trace-mutation-list")
	if err != nil {
		t.Fatalf("list_pending: %v", err)
	}
	var listPayload map[string]any
	if err := json.Unmarshal([]byte(listOutput), &listPayload); err != nil {
		t.Fatalf("decode list payload: %v", err)
	}
	items := listPayload["intents"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected one pending intent, got %+v", listPayload)
	}
	first := items[0].(map[string]any)
	if _, ok := first["query"]; ok {
		t.Fatalf("list_pending should not expose query: %+v", first)
	}
	if _, ok := first["variables"]; ok {
		t.Fatalf("list_pending should not expose variables: %+v", first)
	}

	if _, err := tool.Execute(ctx, json.RawMessage(`{
		"action":"discard",
		"intent_id":"`+intent.IntentID+`"
	}`), "trace-mutation-discard"); err != nil {
		t.Fatalf("discard: %v", err)
	}
	if output, err := tool.Execute(ctx, json.RawMessage(`{"action":"list_pending"}`), "trace-mutation-empty"); err != nil {
		t.Fatalf("list_pending empty: %v", err)
	} else if strings.Contains(output, intent.IntentID) {
		t.Fatalf("discarded intent should be hidden from pending list, got %s", output)
	}
}

func testGraphQLMutationRegistry(t *testing.T) *GraphQLSourceRegistry {
	t.Helper()
	return testGraphQLRegistry(t, GraphQLRegistryConfig{
		Sources: []GraphQLSourceConfig{{
			Name:             "crm",
			Endpoint:         "https://crm.test/query",
			SchemaPath:       writeGraphQLSchema(t, "crm"),
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
			Name:          "update_viewer",
			Source:        "crm",
			Domain:        "people",
			RootMutation:  "updateViewer",
			MaxDepth:      4,
			MaxFields:     8,
			MaxRootFields: 1,
			MaxFragments:  2,
		}},
	})
}

func testGraphQLMutationTool(t *testing.T) *GraphQLMutationTool {
	t.Helper()
	return NewGraphQLMutationTool(testGraphQLMutationRegistry(t)).(*GraphQLMutationTool)
}

func testGraphQLMutationBudgetTool(t *testing.T) *GraphQLMutationTool {
	t.Helper()
	return NewGraphQLMutationTool(testGraphQLRegistry(t, GraphQLRegistryConfig{
		Sources: []GraphQLSourceConfig{{
			Name:             "crm",
			Endpoint:         "https://crm.test/query",
			SchemaPath:       writeGraphQLSchema(t, "crm-budget"),
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
			Name:          "update_viewer",
			Source:        "crm",
			Domain:        "people",
			RootMutation:  "updateViewer",
			MaxDepth:      1,
			MaxFields:     8,
			MaxRootFields: 1,
			MaxFragments:  2,
		}},
	})).(*GraphQLMutationTool)
}

func graphQLMutationContext(sess *session.Session, toolCallID string) context.Context {
	ctx := WithSession(context.Background(), sess)
	return WithToolCallID(ctx, toolCallID)
}

func onlyPendingGraphQLMutationIntent(
	t *testing.T,
	sess *session.Session,
) session.PendingGraphQLMutationIntent {
	t.Helper()
	intents := sess.PendingGraphQLMutationIntentsSnapshot()
	if len(intents) != 1 {
		t.Fatalf("expected one intent, got %+v", intents)
	}
	return intents[0]
}
