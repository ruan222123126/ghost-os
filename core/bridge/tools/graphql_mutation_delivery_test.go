package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"ghost-os/bridge/session"
)

func TestGraphQLMutationToolPrepareFreezesDeliveryKeyAndRequestHash(t *testing.T) {
	tool := NewGraphQLMutationTool(testGraphQLRegistry(t, GraphQLRegistryConfig{
		Sources: []GraphQLSourceConfig{{
			Name:             "crm",
			Endpoint:         "https://crm.test/query",
			SchemaPath:       writeGraphQLSchema(t, "crm-variable-path"),
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
			Name:                    "update_viewer",
			Source:                  "crm",
			Domain:                  "people",
			RootMutation:            "updateViewer",
			IdempotencyMode:         graphQLMutationIdempotencyModeVariablePath,
			IdempotencyVariablePath: "input.clientMutationId",
		}},
	})).(*GraphQLMutationTool)
	sess := session.NewSession("system")

	_, err := tool.Execute(
		graphQLMutationContext(sess, "call-mutation-freeze"),
		json.RawMessage(`{
			"action":"prepare",
			"source":"crm",
			"domain":"people",
			"mutation":"mutation UpdateViewer($input: ViewerInput!) { updateViewer(input: $input) { ok } }",
			"variables":{"input":{"id":"user-1","clientMutationId":""}}
		}`),
		"trace-mutation-freeze",
	)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	intent := onlyPendingGraphQLMutationIntent(t, sess)
	if intent.DeliveryKey == "" || intent.RequestHash == "" {
		t.Fatalf("expected frozen delivery metadata, got %+v", intent)
	}
	input := intent.Variables["input"].(map[string]any)
	if input["clientMutationId"] != intent.DeliveryKey {
		t.Fatalf("expected delivery key injection, got %+v", input)
	}
}

func TestGraphQLMutationToolCommitPersistsCommittingBeforeDispatch(t *testing.T) {
	registry := testGraphQLMutationRegistry(t)
	tool := NewGraphQLMutationTool(registry).(*GraphQLMutationTool)
	sess := session.NewSession("system")

	prepareApprovedMutationIntent(t, tool, sess)
	intent := onlyPendingGraphQLMutationIntent(t, sess)
	savedStatuses := make([]string, 0, 2)
	ctx := graphQLMutationContextWithCheckpoint(sess, "call-mutation-commit", graphQLMutationCheckpointFunc(func(saved *session.Session) error {
		current, ok := saved.PendingGraphQLMutationIntent(intent.IntentID)
		if !ok {
			t.Fatalf("expected persisted intent")
		}
		savedStatuses = append(savedStatuses, current.Status)
		return nil
	}))
	tool.httpClient = &http.Client{
		Transport: graphQLRoundTripper(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("Idempotency-Key"); got != intent.DeliveryKey {
				t.Fatalf("unexpected idempotency header: %q", got)
			}
			return newGraphQLResponse(http.StatusOK, `{"data":{"updateViewer":{"ok":true}}}`), nil
		}),
	}

	output, err := tool.Execute(ctx, json.RawMessage(`{
		"action":"commit",
		"intent_id":"`+intent.IntentID+`"
	}`), "trace-mutation-commit-persist")
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if !strings.Contains(output, `"delivery_key":"`+intent.DeliveryKey+`"`) {
		t.Fatalf("expected delivery key in output, got %s", output)
	}
	if len(savedStatuses) != 2 || savedStatuses[0] != session.GraphQLMutationIntentCommitting || savedStatuses[1] != session.GraphQLMutationIntentExecuted {
		t.Fatalf("unexpected checkpoint sequence: %+v", savedStatuses)
	}
	updated := onlyPendingGraphQLMutationIntent(t, sess)
	if updated.AttemptCount != 1 || len(updated.Receipts) != 1 || updated.Receipts[0].State != session.GraphQLMutationIntentExecuted {
		t.Fatalf("expected executed receipt, got %+v", updated)
	}
}

func TestGraphQLMutationToolTimeoutBecomesDeliveryUnknownAndRetryReusesDeliveryKey(t *testing.T) {
	registry := testGraphQLMutationRegistry(t)
	tool := NewGraphQLMutationTool(registry).(*GraphQLMutationTool)
	sess := session.NewSession("system")

	prepareApprovedMutationIntent(t, tool, sess)
	intent := onlyPendingGraphQLMutationIntent(t, sess)
	ctx := graphQLMutationContext(sess, "call-mutation-retry")
	tool.httpClient = &http.Client{
		Transport: graphQLRoundTripper(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("Idempotency-Key"); got != intent.DeliveryKey {
				t.Fatalf("unexpected idempotency header: %q", got)
			}
			return nil, context.DeadlineExceeded
		}),
	}

	if _, err := tool.Execute(ctx, json.RawMessage(`{
		"action":"commit",
		"intent_id":"`+intent.IntentID+`"
	}`), "trace-mutation-timeout"); err == nil {
		t.Fatal("expected commit timeout error")
	}

	unknown := onlyPendingGraphQLMutationIntent(t, sess)
	if unknown.Status != session.GraphQLMutationIntentDeliveryUnknown {
		t.Fatalf("expected delivery_unknown state, got %+v", unknown)
	}
	if unknown.AttemptCount != 1 || len(unknown.Receipts) != 1 {
		t.Fatalf("expected first receipt, got %+v", unknown)
	}
	if unknown.Receipts[0].DeliveryKey != intent.DeliveryKey || unknown.RequestHash != intent.RequestHash {
		t.Fatalf("expected frozen delivery metadata, got %+v", unknown)
	}

	tool.httpClient = &http.Client{
		Transport: graphQLRoundTripper(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("Idempotency-Key"); got != intent.DeliveryKey {
				t.Fatalf("retry changed idempotency header: %q", got)
			}
			return newGraphQLResponse(http.StatusOK, `{"data":{"updateViewer":{"ok":true}}}`), nil
		}),
	}
	if _, err := tool.Execute(ctx, json.RawMessage(`{
		"action":"retry_commit",
		"intent_id":"`+intent.IntentID+`"
	}`), "trace-mutation-retry"); err != nil {
		t.Fatalf("retry_commit: %v", err)
	}

	executed := onlyPendingGraphQLMutationIntent(t, sess)
	if executed.Status != session.GraphQLMutationIntentExecuted || executed.AttemptCount != 2 {
		t.Fatalf("expected executed retry, got %+v", executed)
	}
	if len(executed.Receipts) != 2 {
		t.Fatalf("expected two receipts, got %+v", executed.Receipts)
	}
	if executed.DeliveryKey != intent.DeliveryKey || executed.RequestHash != intent.RequestHash {
		t.Fatalf("expected frozen delivery metadata after retry, got %+v", executed)
	}
}

func TestGraphQLMutationToolStatusAndRetryGuards(t *testing.T) {
	registry := testGraphQLMutationRegistry(t)
	tool := NewGraphQLMutationTool(registry).(*GraphQLMutationTool)
	sess := session.NewSession("system")

	prepareApprovedMutationIntent(t, tool, sess)
	intent := onlyPendingGraphQLMutationIntent(t, sess)
	tool.httpClient = &http.Client{
		Transport: graphQLRoundTripper(func(*http.Request) (*http.Response, error) {
			return newGraphQLResponse(http.StatusOK, `{"data":{"updateViewer":{"ok":true}}}`), nil
		}),
	}
	ctx := graphQLMutationContext(sess, "call-mutation-status")
	if _, err := tool.Execute(ctx, json.RawMessage(`{
		"action":"commit",
		"intent_id":"`+intent.IntentID+`"
	}`), "trace-mutation-status-commit"); err != nil {
		t.Fatalf("commit: %v", err)
	}

	statusOutput, err := tool.Execute(ctx, json.RawMessage(`{
		"action":"status",
		"intent_id":"`+intent.IntentID+`"
	}`), "trace-mutation-status")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(statusOutput), &payload); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if payload["status"] != session.GraphQLMutationIntentExecuted || payload["attempt_count"] != float64(1) {
		t.Fatalf("unexpected status payload: %+v", payload)
	}
	if _, err := tool.Execute(ctx, json.RawMessage(`{
		"action":"retry_commit",
		"intent_id":"`+intent.IntentID+`"
	}`), "trace-mutation-status-retry"); err == nil || !strings.Contains(err.Error(), "already executed") {
		t.Fatalf("expected executed retry guard, got %v", err)
	}
}

func prepareApprovedMutationIntent(
	t *testing.T,
	tool *GraphQLMutationTool,
	sess *session.Session,
) {
	t.Helper()
	_, err := tool.Execute(
		graphQLMutationContext(sess, "call-mutation-prepare"),
		json.RawMessage(`{
			"action":"prepare",
			"source":"crm",
			"domain":"people",
			"mutation":"mutation ApprovedMutation($input: ViewerInput!) { updateViewer(input: $input) { ok } }",
			"variables":{"input":{"id":"user-1"}}
		}`),
		"trace-mutation-prepare-approved",
	)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	intent := onlyPendingGraphQLMutationIntent(t, sess)
	if !sess.SetHumanAnswer(intent.QuestionID, "Approve") {
		t.Fatalf("expected human approval")
	}
}

func graphQLMutationContextWithCheckpoint(
	sess *session.Session,
	toolCallID string,
	checkpoint SessionCheckpoint,
) context.Context {
	ctx := WithSession(context.Background(), sess)
	ctx = WithSessionCheckpoint(ctx, checkpoint)
	return WithToolCallID(ctx, toolCallID)
}
