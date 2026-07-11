package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompleteCodexPrefersStatelessReplayForFollowUpWhenEnabled(t *testing.T) {
	requestBody, resp := completeCodexFollowUp(t, true)

	if resp.Message.Text != "follow-up ok" {
		t.Fatalf("unexpected response text: got %q want %q", resp.Message.Text, "follow-up ok")
	}
	if requestBody.PreviousResponseID != "" {
		t.Fatalf("expected stateless request, got previous_response_id=%q", requestBody.PreviousResponseID)
	}
	assertCodexFollowUpStatelessInput(t, requestBody.Input)
}

func TestCompleteCodexKeepsContinuationForFollowUpWhenStatelessReplayDisabled(t *testing.T) {
	requestBody, _ := completeCodexFollowUp(t, false)

	if requestBody.PreviousResponseID != "resp_prev" {
		t.Fatalf("unexpected previous_response_id: got %q want %q", requestBody.PreviousResponseID, "resp_prev")
	}
	if len(requestBody.Input) != 1 || requestBody.Input[0].Role != "user" {
		t.Fatalf("unexpected continuation input: %+v", requestBody.Input)
	}
	if requestBody.Input[0].Content[0].Text != "second question" {
		t.Fatalf("unexpected continuation text: %+v", requestBody.Input[0])
	}
}

func TestCompleteStreamCodexPrefersStatelessReplayForFollowUpWhenEnabled(t *testing.T) {
	requestBody, resp := streamCodexFollowUp(t, true)

	if resp.Message.Text != "follow-up ok" {
		t.Fatalf("unexpected response text: got %q want %q", resp.Message.Text, "follow-up ok")
	}
	if requestBody.PreviousResponseID != "" {
		t.Fatalf("expected stateless stream request, got previous_response_id=%q", requestBody.PreviousResponseID)
	}
	assertCodexFollowUpStatelessInput(t, requestBody.Input)
}

func completeCodexFollowUp(t *testing.T, statelessRetryEnabled bool) (codexRequest, *CompletionResponse) {
	t.Helper()

	requestBodies := make([]codexRequest, 0, 1)
	server := codexFollowUpServer(t, &requestBodies, false)
	defer server.Close()

	client := NewClientWithOptions(ClientOptions{
		Provider:                   ProviderCodex,
		BaseURL:                    server.URL,
		Model:                      "gpt-5.4",
		CodexStatelessRetryEnabled: statelessRetryEnabled,
	})
	client.httpClient = server.Client()

	resp, err := client.Complete(context.Background(), codexFollowUpCompletionRequest(server.URL))
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if len(requestBodies) != 1 {
		t.Fatalf("unexpected request count: got %d want 1", len(requestBodies))
	}
	return requestBodies[0], resp
}

func streamCodexFollowUp(t *testing.T, statelessRetryEnabled bool) (codexRequest, *CompletionResponse) {
	t.Helper()

	requestBodies := make([]codexRequest, 0, 1)
	server := codexFollowUpServer(t, &requestBodies, true)
	defer server.Close()

	client := NewClientWithOptions(ClientOptions{
		Provider:                   ProviderCodex,
		BaseURL:                    server.URL,
		Model:                      "gpt-5.4",
		CodexStatelessRetryEnabled: statelessRetryEnabled,
	})
	client.httpClient = server.Client()

	resp, err := client.CompleteStream(
		context.Background(),
		codexFollowUpCompletionRequest(server.URL),
		&recordingLLMStreamSink{},
	)
	if err != nil {
		t.Fatalf("CompleteStream returned error: %v", err)
	}
	if len(requestBodies) != 1 {
		t.Fatalf("unexpected request count: got %d want 1", len(requestBodies))
	}
	return requestBodies[0], resp
}

func codexFollowUpServer(
	t *testing.T,
	requestBodies *[]codexRequest,
	stream bool,
) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		defer r.Body.Close()

		var body codexRequest
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		*requestBodies = append(*requestBodies, body)

		if stream {
			writeCodexFollowUpStream(w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"id":"resp_next","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"follow-up ok"}]}]}`,
		))
	}))
}

func writeCodexFollowUpStream(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = w.Write([]byte(strings.Join([]string{
		`event: response.output_text.delta`,
		`data: {"type":"response.output_text.delta","delta":"follow-up ok"}`,
		"",
		`event: response.completed`,
		`data: {"type":"response.completed","response":{"id":"resp_next","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"follow-up ok"}]}]}}`,
		"",
	}, "\n")))
}

func codexFollowUpCompletionRequest(baseURL string) CompletionRequest {
	return CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "hello"},
			{Role: RoleAssistant, Text: "first answer"},
			{Role: RoleUser, Text: "second question"},
		},
		ConversationState: ConversationState{
			Provider:           ProviderCodex,
			BaseURL:            baseURL,
			Model:              "gpt-5.4",
			PreviousResponseID: "resp_prev",
		},
	}
}

func assertCodexFollowUpStatelessInput(t *testing.T, input []codexInputItem) {
	t.Helper()

	if len(input) != 1 {
		t.Fatalf("unexpected stateless input count: got %d want 1", len(input))
	}
	item := input[0]
	if item.Type != "message" || item.Role != "user" {
		t.Fatalf("unexpected stateless input item: %+v", item)
	}
	if len(item.Content) != 1 {
		t.Fatalf("unexpected stateless input content: %+v", item)
	}
	text := item.Content[0].Text
	for _, want := range []string{"User: hello", "Assistant: first answer", "Current request: second question"} {
		if !strings.Contains(text, want) {
			t.Fatalf("stateless input missing %q: %q", want, text)
		}
	}
}
