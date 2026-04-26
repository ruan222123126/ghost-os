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

	resp, err := client.CompleteStream(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Text: "Read config file"},
		},
	}, sink)
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

func TestCompleteStreamOpenAIReasoningDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"id":"chatcmpl-r1","choices":[{"index":0,"delta":{"role":"assistant","reasoning_content":"Analyzing"},"finish_reason":null}]}`,
			"",
			`data: {"id":"chatcmpl-r1","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":"stop"}]}`,
			"",
			`data: [DONE]`,
			"",
		}, "\n")))
	}))
	defer server.Close()

	client := newStreamTestClient(server, ProviderOpenAI)
	sink := &recordingLLMStreamSink{}

	resp, err := client.CompleteStream(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Text: "say hello"},
		},
	}, sink)
	if err != nil {
		t.Fatalf("CompleteStream returned error: %v", err)
	}
	if resp.Message.Text != "Hello" {
		t.Fatalf("unexpected text: got %q want %q", resp.Message.Text, "Hello")
	}
	if got := string(resp.Message.ReasoningContent); got != `"Analyzing"` {
		t.Fatalf("unexpected reasoning_content: got %q want %q", got, `"Analyzing"`)
	}
	if len(sink.deltas) != 2 {
		t.Fatalf("unexpected delta count: got %d want 2", len(sink.deltas))
	}
	if sink.deltas[0].Kind != DeltaKindThinking || sink.deltas[0].Thinking != "Analyzing" {
		t.Fatalf("unexpected first delta: %+v", sink.deltas[0])
	}
	if sink.deltas[1].Kind != DeltaKindText || sink.deltas[1].Text != "Hello" {
		t.Fatalf("unexpected second delta: %+v", sink.deltas[1])
	}
}

func TestCompleteStreamOpenAIReasoningDeltaRejectsUnsupportedPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"id":"chatcmpl-r2","choices":[{"index":0,"delta":{"role":"assistant","reasoning_content":{"foo":"bar"}},"finish_reason":null}]}`,
			"",
			`data: [DONE]`,
			"",
		}, "\n")))
	}))
	defer server.Close()

	client := newStreamTestClient(server, ProviderOpenAI)
	sink := &recordingLLMStreamSink{}

	_, err := client.CompleteStream(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Text: "status"},
		},
	}, sink)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "unsupported openai stream reasoning_content") {
		t.Fatalf("unexpected error: %v", err)
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

	resp, err := client.CompleteStream(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Text: "Say hello"},
		},
	}, sink)
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

	resp, err := client.CompleteStream(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Text: "read config"},
		},
	}, sink)
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

func TestCompleteStreamAnthropicThinkingDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: message_start`,
			`data: {"type":"message_start","message":{"id":"msg_t1","role":"assistant","content":[],"stop_reason":"","usage":{"input_tokens":9,"output_tokens":0}}}`,
			"",
			`event: content_block_start`,
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"Analyzing"}}`,
			"",
			`event: content_block_delta`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":" step"}}`,
			"",
			`event: content_block_stop`,
			`data: {"type":"content_block_stop","index":0}`,
			"",
			`event: content_block_start`,
			`data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`,
			"",
			`event: content_block_delta`,
			`data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Done"}}`,
			"",
			`event: content_block_stop`,
			`data: {"type":"content_block_stop","index":1}`,
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

	resp, err := client.CompleteStream(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Text: "status"},
		},
	}, sink)
	if err != nil {
		t.Fatalf("CompleteStream returned error: %v", err)
	}
	if resp.Message.Text != "Done" {
		t.Fatalf("unexpected text: got %q want %q", resp.Message.Text, "Done")
	}
	if len(sink.deltas) != 3 {
		t.Fatalf("unexpected delta count: got %d want 3", len(sink.deltas))
	}
	if sink.deltas[0].Kind != DeltaKindThinking || sink.deltas[0].Thinking != "Analyzing" {
		t.Fatalf("unexpected first delta: %+v", sink.deltas[0])
	}
	if sink.deltas[1].Kind != DeltaKindThinking || sink.deltas[1].Thinking != " step" {
		t.Fatalf("unexpected second delta: %+v", sink.deltas[1])
	}
	if sink.deltas[2].Kind != DeltaKindText || sink.deltas[2].Text != "Done" {
		t.Fatalf("unexpected third delta: %+v", sink.deltas[2])
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

	resp, err := client.CompleteStream(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Text: "say hello"},
		},
	}, sink)
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

func TestCompleteStreamCodexToolCallSequence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: response.output_item.added`,
			`data: {"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"read_file","arguments":""}}`,
			"",
			`event: response.function_call_arguments.delta`,
			`data: {"type":"response.function_call_arguments.delta","output_index":0,"delta":"{\"path\":\"con"}`,
			"",
			`event: response.function_call_arguments.delta`,
			`data: {"type":"response.function_call_arguments.delta","output_index":0,"delta":"fig.toml\"}"}`,
			"",
			`event: response.output_item.done`,
			`data: {"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"read_file","arguments":"{\"path\":\"config.toml\"}"}}`,
			"",
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"id":"resp_1","status":"completed","output":[{"type":"function_call","id":"fc_1","call_id":"call_1","name":"read_file","arguments":"{\"path\":\"config.toml\"}"}],"usage":{"input_tokens":8,"output_tokens":4,"total_tokens":12}}}`,
			"",
		}, "\n")))
	}))
	defer server.Close()

	client := NewClientWithOptions(ClientOptions{
		Provider:                   ProviderCodex,
		BaseURL:                    server.URL,
		Model:                      "test-model",
		CodexStatelessRetryEnabled: true,
	})
	client.httpClient = server.Client()
	sink := &recordingLLMStreamSink{}

	resp, err := client.CompleteStream(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Text: "read config.toml"},
		},
	}, sink)
	if err != nil {
		t.Fatalf("CompleteStream returned error: %v", err)
	}
	if resp.FinishReason != FinishToolCalls {
		t.Fatalf("unexpected finish reason: got %q want %q", resp.FinishReason, FinishToolCalls)
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("unexpected tool call count: got %d want 1", len(resp.Message.ToolCalls))
	}
	if got := string(resp.Message.ToolCalls[0].Arguments); got != `{"path":"config.toml"}` {
		t.Fatalf("unexpected tool args: got %q", got)
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

func TestCompleteStreamCodexTextDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: response.output_text.delta`,
			`data: {"type":"response.output_text.delta","delta":"Hel"}`,
			"",
			`event: response.output_text.delta`,
			`data: {"type":"response.output_text.delta","delta":"lo"}`,
			"",
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"id":"resp_2","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Hello"}]}],"usage":{"input_tokens":5,"output_tokens":2,"total_tokens":7}}}`,
			"",
		}, "\n")))
	}))
	defer server.Close()

	client := NewClientWithOptions(ClientOptions{
		Provider:                   ProviderCodex,
		BaseURL:                    server.URL,
		Model:                      "test-model",
		CodexStatelessRetryEnabled: true,
	})
	client.httpClient = server.Client()
	sink := &recordingLLMStreamSink{}

	resp, err := client.CompleteStream(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Text: "say hello"},
		},
	}, sink)
	if err != nil {
		t.Fatalf("CompleteStream returned error: %v", err)
	}
	if resp.Message.Text != "Hello" {
		t.Fatalf("unexpected text: got %q want %q", resp.Message.Text, "Hello")
	}
	if resp.FinishReason != FinishStop {
		t.Fatalf("unexpected finish reason: got %q want %q", resp.FinishReason, FinishStop)
	}
	if len(sink.deltas) != 2 || sink.deltas[0].Text != "Hel" || sink.deltas[1].Text != "lo" {
		t.Fatalf("unexpected deltas: %+v", sink.deltas)
	}
}

func TestCompleteStreamCodexThinkingDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: response.reasoning_summary_text.delta`,
			`data: {"type":"response.reasoning_summary_text.delta","delta":"Analyzing"}`,
			"",
			`event: response.reasoning_text.delta`,
			`data: {"type":"response.reasoning_text.delta","delta":" detail"}`,
			"",
			`event: response.output_text.delta`,
			`data: {"type":"response.output_text.delta","delta":"Done"}`,
			"",
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"id":"resp_t2","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Done"}]}],"usage":{"input_tokens":5,"output_tokens":2,"total_tokens":7}}}`,
			"",
		}, "\n")))
	}))
	defer server.Close()

	client := NewClientWithOptions(ClientOptions{
		Provider:                   ProviderCodex,
		BaseURL:                    server.URL,
		Model:                      "test-model",
		CodexStatelessRetryEnabled: true,
	})
	client.httpClient = server.Client()
	sink := &recordingLLMStreamSink{}

	resp, err := client.CompleteStream(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Text: "status"},
		},
	}, sink)
	if err != nil {
		t.Fatalf("CompleteStream returned error: %v", err)
	}
	if resp.Message.Text != "Done" {
		t.Fatalf("unexpected text: got %q want %q", resp.Message.Text, "Done")
	}
	if len(sink.deltas) != 3 {
		t.Fatalf("unexpected delta count: got %d want 3", len(sink.deltas))
	}
	if sink.deltas[0].Kind != DeltaKindThinking || sink.deltas[0].Thinking != "Analyzing" {
		t.Fatalf("unexpected first delta: %+v", sink.deltas[0])
	}
	if sink.deltas[1].Kind != DeltaKindThinking || sink.deltas[1].Thinking != " detail" {
		t.Fatalf("unexpected second delta: %+v", sink.deltas[1])
	}
	if sink.deltas[2].Kind != DeltaKindText || sink.deltas[2].Text != "Done" {
		t.Fatalf("unexpected third delta: %+v", sink.deltas[2])
	}
}

func TestCompleteStreamCodexFallsBackToStatelessReplayAfterContinuation400(t *testing.T) {
	requestBodies := make([]codexRequest, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		defer r.Body.Close()

		var body codexRequest
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBodies = append(requestBodies, body)

		if len(requestBodies) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"上游服务异常，请联系管理员","type":"upstream_error","code":"upstream_error"}}`))
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: response.output_text.delta`,
			`data: {"type":"response.output_text.delta","delta":"done"}`,
			"",
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"id":"resp_2","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"done"}]}],"usage":{"input_tokens":9,"output_tokens":2,"total_tokens":11}}}`,
			"",
		}, "\n")))
	}))
	defer server.Close()

	client := NewClientWithOptions(ClientOptions{
		Provider:                   ProviderCodex,
		BaseURL:                    server.URL,
		Model:                      "test-model",
		CodexStatelessRetryEnabled: true,
	})
	client.httpClient = server.Client()
	sink := &recordingLLMStreamSink{}

	resp, err := client.CompleteStream(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "Need your approval"},
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{ID: "call_1", Name: "ask_human", Arguments: json.RawMessage(`{"question":"continue?"}`)},
				},
			},
			{Role: RoleTool, ToolCallID: "call_1", Text: `{"answer":"ai"}`},
		},
		ConversationState: ConversationState{
			Provider:           ProviderCodex,
			BaseURL:            server.URL,
			Model:              "test-model",
			PreviousResponseID: "resp_prev",
		},
	}, sink)
	if err != nil {
		t.Fatalf("CompleteStream returned error: %v", err)
	}
	if resp.Message.Text != "done" {
		t.Fatalf("unexpected response text: got %q want %q", resp.Message.Text, "done")
	}
	if len(sink.deltas) != 1 || sink.deltas[0].Kind != DeltaKindText || sink.deltas[0].Text != "done" {
		t.Fatalf("unexpected deltas: %+v", sink.deltas)
	}
	if len(requestBodies) != 2 {
		t.Fatalf("unexpected request count: got %d want 2", len(requestBodies))
	}
	if requestBodies[0].PreviousResponseID != "" {
		t.Fatalf("first request should proactively clear previous_response_id, got %q", requestBodies[0].PreviousResponseID)
	}
	if requestBodies[1].PreviousResponseID != "" {
		t.Fatalf("second request should clear previous_response_id, got %q", requestBodies[1].PreviousResponseID)
	}
	if len(requestBodies[0].Input) != 3 {
		t.Fatalf("unexpected continuation payload: %+v", requestBodies[0].Input)
	}
	if requestBodies[0].Input[0].Type != "message" || requestBodies[0].Input[0].Role != "user" {
		t.Fatalf("unexpected first replay user item: %+v", requestBodies[0].Input[0])
	}
	if requestBodies[0].Input[1].Type != "function_call" || requestBodies[0].Input[2].Type != "function_call_output" {
		t.Fatalf("unexpected first replay tail: %+v", requestBodies[0].Input)
	}
	if len(requestBodies[1].Input) != 3 {
		t.Fatalf("unexpected stateless replay input count: got %d want 3", len(requestBodies[1].Input))
	}
	if requestBodies[1].Input[0].Type != "message" || requestBodies[1].Input[0].Role != "user" {
		t.Fatalf("unexpected stateless synthetic user item: %+v", requestBodies[1].Input[0])
	}
	if requestBodies[1].Input[1].Type != "function_call" || requestBodies[1].Input[2].Type != "function_call_output" {
		t.Fatalf("unexpected stateless replay tail: %+v", requestBodies[1].Input)
	}
}

func TestCompleteStreamCodexFallsBackToStatelessReplayAfterMissingFunctionCallOutputError(t *testing.T) {
	requestBodies := make([]codexRequest, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		defer r.Body.Close()

		var body codexRequest
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBodies = append(requestBodies, body)

		if len(requestBodies) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"No tool call found for function_call_output with call_id call_1.","type":"invalid_request_error","code":null}}`))
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`event: response.output_text.delta`,
			`data: {"type":"response.output_text.delta","delta":"done"}`,
			"",
			`event: response.completed`,
			`data: {"type":"response.completed","response":{"id":"resp_2","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"done"}]}],"usage":{"input_tokens":9,"output_tokens":2,"total_tokens":11}}}`,
			"",
		}, "\n")))
	}))
	defer server.Close()

	client := NewClientWithOptions(ClientOptions{
		Provider:                   ProviderCodex,
		BaseURL:                    server.URL,
		Model:                      "test-model",
		CodexStatelessRetryEnabled: true,
	})
	client.httpClient = server.Client()
	sink := &recordingLLMStreamSink{}

	resp, err := client.CompleteStream(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "Need your approval"},
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{ID: "call_1", Name: "ask_human", Arguments: json.RawMessage(`{"question":"continue?"}`)},
				},
			},
			{Role: RoleTool, ToolCallID: "call_1", Text: `{"answer":"ai"}`},
		},
		ConversationState: ConversationState{
			Provider:           ProviderCodex,
			BaseURL:            server.URL,
			Model:              "test-model",
			PreviousResponseID: "resp_prev",
		},
	}, sink)
	if err != nil {
		t.Fatalf("CompleteStream returned error: %v", err)
	}
	if resp.Message.Text != "done" {
		t.Fatalf("unexpected response text: got %q want %q", resp.Message.Text, "done")
	}
	if len(requestBodies) != 2 {
		t.Fatalf("unexpected request count: got %d want 2", len(requestBodies))
	}
	if requestBodies[0].PreviousResponseID != "" {
		t.Fatalf("first request should proactively clear previous_response_id, got %q", requestBodies[0].PreviousResponseID)
	}
	if requestBodies[1].PreviousResponseID != "" {
		t.Fatalf("second request should clear previous_response_id, got %q", requestBodies[1].PreviousResponseID)
	}
	if len(requestBodies[0].Input) != 3 {
		t.Fatalf("unexpected first stateless replay input count: got %d want 3", len(requestBodies[0].Input))
	}
	if requestBodies[0].Input[1].Type != "function_call" || requestBodies[0].Input[2].Type != "function_call_output" {
		t.Fatalf("unexpected first stateless replay tail: %+v", requestBodies[0].Input)
	}
	if len(requestBodies[1].Input) != 3 {
		t.Fatalf("unexpected stateless replay input count: got %d want 3", len(requestBodies[1].Input))
	}
	if requestBodies[1].Input[1].Type != "function_call" || requestBodies[1].Input[2].Type != "function_call_output" {
		t.Fatalf("unexpected stateless replay tail: %+v", requestBodies[1].Input)
	}
}

func TestCompleteStreamCodexDoesNotFallbackToStatelessReplayByDefault(t *testing.T) {
	requestBodies := make([]codexRequest, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		defer r.Body.Close()

		var body codexRequest
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBodies = append(requestBodies, body)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"upstream_error","type":"upstream_error","code":"upstream_error"}}`))
	}))
	defer server.Close()

	client := newStreamTestClient(server, ProviderCodex)
	sink := &recordingLLMStreamSink{}

	_, err := client.CompleteStream(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "hello"},
		},
		ConversationState: ConversationState{
			Provider:           ProviderCodex,
			BaseURL:            server.URL,
			Model:              "test-model",
			PreviousResponseID: "resp_prev",
		},
	}, sink)
	if err == nil || !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected upstream 400 error without fallback, got %v", err)
	}
	if len(requestBodies) != 1 {
		t.Fatalf("expected one request without fallback, got %d", len(requestBodies))
	}
}
