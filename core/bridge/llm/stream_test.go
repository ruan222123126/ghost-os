package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type recordingLLMStreamSink struct {
	deltas []LLMDelta
}

func (r *recordingLLMStreamSink) OnDelta(_ context.Context, delta LLMDelta) error {
	r.deltas = append(r.deltas, delta)
	return nil
}

func newStreamTestClient(server *httptest.Server, provider Provider) *Client {
	client := NewClientWithOptions(ClientOptions{
		Provider: provider,
		BaseURL:  server.URL,
		Model:    "test-model",
	})
	client.httpClient = server.Client()
	return client
}

func TestStreamJSONParsesSSEFrames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"n\":1}\n\n"))
		_, _ = w.Write([]byte("data: {\"n\":2}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := newStreamTestClient(server, ProviderOpenAI)
	lines := make([]string, 0, 2)
	err := client.streamJSON(context.Background(), "/chat/completions", map[string]any{"stream": true}, nil, func(line []byte) error {
		lines = append(lines, string(line))
		return nil
	})
	if err != nil {
		t.Fatalf("streamJSON returned error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("unexpected line count: got %d want %d", len(lines), 2)
	}
	if lines[0] != "{\"n\":1}" || lines[1] != "{\"n\":2}" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}

func TestCompleteStreamOpenAITextDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"id":"chatcmpl-1","choices":[{"index":0,"delta":{"role":"assistant","content":"Hel"},"finish_reason":null}]}`,
			"",
			`data: {"id":"chatcmpl-1","choices":[{"index":0,"delta":{"content":"lo"},"finish_reason":"stop"}]}`,
			"",
			`data: [DONE]`,
			"",
		}, "\n")))
	}))
	defer server.Close()

	client := newStreamTestClient(server, ProviderOpenAI)
	sink := &recordingLLMStreamSink{}

	resp, err := client.CompleteStream(context.Background(), CompletionRequest{}, sink)
	if err != nil {
		t.Fatalf("CompleteStream returned error: %v", err)
	}
	if resp.Message.Text != "Hello" {
		t.Fatalf("unexpected text: got %q want %q", resp.Message.Text, "Hello")
	}
	if resp.FinishReason != FinishStop {
		t.Fatalf("unexpected finish reason: got %q want %q", resp.FinishReason, FinishStop)
	}
	if len(sink.deltas) != 2 {
		t.Fatalf("unexpected delta count: got %d want %d", len(sink.deltas), 2)
	}
	if sink.deltas[0].Kind != DeltaKindText || sink.deltas[0].Text != "Hel" {
		t.Fatalf("unexpected first delta: %+v", sink.deltas[0])
	}
	if sink.deltas[1].Text != "lo" {
		t.Fatalf("unexpected second delta: %+v", sink.deltas[1])
	}
}

func TestCompleteStreamOpenAIToolCallSequence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"id":"chatcmpl-2","choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"web_search","arguments":"{\"query\":\"gol"}}]},"finish_reason":null}]}`,
			"",
			`data: {"id":"chatcmpl-2","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ang\"}"}}],"content":""},"finish_reason":"tool_calls"}]}`,
			"",
			`data: [DONE]`,
			"",
		}, "\n")))
	}))
	defer server.Close()

	client := newStreamTestClient(server, ProviderOpenAI)
	sink := &recordingLLMStreamSink{}

	resp, err := client.CompleteStream(context.Background(), CompletionRequest{}, sink)
	if err != nil {
		t.Fatalf("CompleteStream returned error: %v", err)
	}
	if resp.FinishReason != FinishToolCalls {
		t.Fatalf("unexpected finish reason: got %q want %q", resp.FinishReason, FinishToolCalls)
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("unexpected tool call count: got %d want %d", len(resp.Message.ToolCalls), 1)
	}
	if resp.Message.ToolCalls[0].Name != "web_search" {
		t.Fatalf("unexpected tool name: got %q want %q", resp.Message.ToolCalls[0].Name, "web_search")
	}
	if got, want := string(resp.Message.ToolCalls[0].Arguments), `{"query":"golang"}`; got != want {
		t.Fatalf("unexpected tool args: got %q want %q", got, want)
	}
	wantKinds := []DeltaKind{
		DeltaKindToolCallStart,
		DeltaKindToolCallDelta,
		DeltaKindToolCallDelta,
		DeltaKindToolCallEnd,
	}
	if len(sink.deltas) != len(wantKinds) {
		t.Fatalf("unexpected delta count: got %d want %d", len(sink.deltas), len(wantKinds))
	}
	for index, want := range wantKinds {
		if sink.deltas[index].Kind != want {
			t.Fatalf("unexpected delta[%d]: got %q want %q", index, sink.deltas[index].Kind, want)
		}
	}
}

func TestCompleteStreamAnthropicTextDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: message_start`,
			`data: {"type":"message_start","message":{"id":"msg_1","role":"assistant","content":[],"stop_reason":"","usage":{"input_tokens":12,"output_tokens":0}}}`,
			"",
			`event: content_block_start`,
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			"",
			`event: content_block_delta`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			"",
			`event: content_block_delta`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}`,
			"",
			`event: content_block_stop`,
			`data: {"type":"content_block_stop","index":0}`,
			"",
			`event: message_delta`,
			`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":2}}`,
			"",
			`event: message_stop`,
			`data: {"type":"message_stop"}`,
			"",
		}, "\n")))
	}))
	defer server.Close()

	client := newStreamTestClient(server, ProviderAnthropic)
	sink := &recordingLLMStreamSink{}

	resp, err := client.CompleteStream(context.Background(), CompletionRequest{}, sink)
	if err != nil {
		t.Fatalf("CompleteStream returned error: %v", err)
	}
	if resp.Message.Text != "Hello world" {
		t.Fatalf("unexpected text: got %q want %q", resp.Message.Text, "Hello world")
	}
	if resp.FinishReason != FinishStop {
		t.Fatalf("unexpected finish reason: got %q want %q", resp.FinishReason, FinishStop)
	}
	if len(sink.deltas) != 2 || sink.deltas[0].Text != "Hello" || sink.deltas[1].Text != " world" {
		t.Fatalf("unexpected deltas: %+v", sink.deltas)
	}
}

func TestCompleteStreamAnthropicToolCallSequence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: message_start`,
			`data: {"type":"message_start","message":{"id":"msg_2","role":"assistant","content":[],"stop_reason":"","usage":{"input_tokens":8,"output_tokens":0}}}`,
			"",
			`event: content_block_start`,
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"web_search","input":{}}}`,
			"",
			`event: content_block_delta`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"query\":\"gol"}}`,
			"",
			`event: content_block_delta`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"ang\"}"}}`,
			"",
			`event: content_block_stop`,
			`data: {"type":"content_block_stop","index":0}`,
			"",
			`event: message_delta`,
			`data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":3}}`,
			"",
			`event: message_stop`,
			`data: {"type":"message_stop"}`,
			"",
		}, "\n")))
	}))
	defer server.Close()

	client := newStreamTestClient(server, ProviderAnthropic)
	sink := &recordingLLMStreamSink{}

	resp, err := client.CompleteStream(context.Background(), CompletionRequest{}, sink)
	if err != nil {
		t.Fatalf("CompleteStream returned error: %v", err)
	}
	if resp.FinishReason != FinishToolCalls {
		t.Fatalf("unexpected finish reason: got %q want %q", resp.FinishReason, FinishToolCalls)
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("unexpected tool call count: got %d want %d", len(resp.Message.ToolCalls), 1)
	}
	if got, want := string(resp.Message.ToolCalls[0].Arguments), `{"query":"golang"}`; got != want {
		t.Fatalf("unexpected tool args: got %q want %q", got, want)
	}
	wantKinds := []DeltaKind{
		DeltaKindToolCallStart,
		DeltaKindToolCallDelta,
		DeltaKindToolCallDelta,
		DeltaKindToolCallEnd,
	}
	if len(sink.deltas) != len(wantKinds) {
		t.Fatalf("unexpected delta count: got %d want %d", len(sink.deltas), len(wantKinds))
	}
	for index, want := range wantKinds {
		if sink.deltas[index].Kind != want {
			t.Fatalf("unexpected delta[%d]: got %q want %q", index, sink.deltas[index].Kind, want)
		}
	}
}
