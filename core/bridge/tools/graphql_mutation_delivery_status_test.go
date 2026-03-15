package tools

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"

	"ghost-os/bridge/session"
)

func TestGraphQLMutationToolLogsDeliveryUnknown(t *testing.T) {
	registry := testGraphQLMutationRegistry(t)
	tool := NewGraphQLMutationTool(registry).(*GraphQLMutationTool)
	sess := session.NewSession("system")
	logs := &bytes.Buffer{}
	original := log.Writer()
	log.SetOutput(logs)
	t.Cleanup(func() {
		log.SetOutput(original)
	})

	prepareApprovedMutationIntent(t, tool, sess)
	intent := onlyPendingGraphQLMutationIntent(t, sess)
	tool.httpClient = &http.Client{
		Transport: graphQLRoundTripper(func(*http.Request) (*http.Response, error) {
			return nil, io.ErrUnexpectedEOF
		}),
	}

	_, _ = tool.Execute(graphQLMutationContext(sess, "call-mutation-log"), json.RawMessage(`{
		"action":"commit",
		"intent_id":"`+intent.IntentID+`"
	}`), "trace-mutation-log")

	output := logs.String()
	if !strings.Contains(output, "commit_state=delivery_unknown") || !strings.Contains(output, "delivery_key="+intent.DeliveryKey) {
		t.Fatalf("expected delivery trace fields, got %q", output)
	}
}

func TestGraphQLMutationToolRetryCommitOnlyAllowsApprovedOrUnknown(t *testing.T) {
	registry := testGraphQLMutationRegistry(t)
	tool := NewGraphQLMutationTool(registry).(*GraphQLMutationTool)
	sess := session.NewSession("system")
	prepareApprovedMutationIntent(t, tool, sess)
	intent := onlyPendingGraphQLMutationIntent(t, sess)
	intent.Status = session.GraphQLMutationIntentRejected
	intent.CommitState = ""
	if !sess.ReplacePendingGraphQLMutationIntent(intent) {
		t.Fatalf("replace intent")
	}

	_, err := tool.Execute(graphQLMutationContext(sess, "call-mutation-retry-guard"), json.RawMessage(`{
		"action":"retry_commit",
		"intent_id":"`+intent.IntentID+`"
	}`), "trace-mutation-retry-guard")
	if err == nil || !strings.Contains(err.Error(), "not retryable") {
		t.Fatalf("expected retry guard, got %v", err)
	}
}

func TestGraphQLMutationToolCommitPersistsLatestReceiptErrorSummary(t *testing.T) {
	registry := testGraphQLMutationRegistry(t)
	tool := NewGraphQLMutationTool(registry).(*GraphQLMutationTool)
	sess := session.NewSession("system")
	prepareApprovedMutationIntent(t, tool, sess)
	intent := onlyPendingGraphQLMutationIntent(t, sess)
	tool.httpClient = &http.Client{
		Transport: graphQLRoundTripper(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("connection reset by peer")
		}),
	}

	_, _ = tool.Execute(graphQLMutationContext(sess, "call-mutation-error-summary"), json.RawMessage(`{
		"action":"commit",
		"intent_id":"`+intent.IntentID+`"
	}`), "trace-mutation-error-summary")

	updated := onlyPendingGraphQLMutationIntent(t, sess)
	if !strings.Contains(updated.LastError, "connection reset by peer") {
		t.Fatalf("expected last error summary, got %+v", updated)
	}
}
