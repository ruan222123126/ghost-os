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

func TestCodexToCompletionResponseMapsToolCalls(t *testing.T) {
	resp, err := codexToCompletionResponse(codexResponse{
		Status: "completed",
		Output: []codexOutputItem{
			{
				Type:      "function_call",
				ID:        "fc_1",
				Name:      "read_file",
				Arguments: `{"path":"README.md"}`,
			},
		},
		Usage: &codexUsage{InputTokens: 10, OutputTokens: 4, TotalTokens: 14},
	})
	if err != nil {
		t.Fatalf("codexToCompletionResponse returned error: %v", err)
	}
	if resp.FinishReason != FinishToolCalls {
		t.Fatalf("unexpected finish reason: got %q want %q", resp.FinishReason, FinishToolCalls)
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("unexpected tool call count: got %d want 1", len(resp.Message.ToolCalls))
	}
	if resp.Message.ToolCalls[0].ID != "fc_1" {
		t.Fatalf("unexpected tool id: got %q want %q", resp.Message.ToolCalls[0].ID, "fc_1")
	}
	if resp.Message.ToolCalls[0].Name != "read_file" {
		t.Fatalf("unexpected tool name: got %q want %q", resp.Message.ToolCalls[0].Name, "read_file")
	}
	if got := string(resp.Message.ToolCalls[0].Arguments); got != `{"path":"README.md"}` {
		t.Fatalf("unexpected tool arguments: got %q", got)
	}
	if resp.Usage.TotalTokens != 14 {
		t.Fatalf("unexpected total tokens: got %d want 14", resp.Usage.TotalTokens)
	}
}

func TestCodexToCompletionResponseMapsTextAndLengthFinish(t *testing.T) {
	resp, err := codexToCompletionResponse(codexResponse{
		Status: "incomplete",
		IncompleteDetails: &codexIncompleteDetails{
			Reason: "max_output_tokens",
		},
		Output: []codexOutputItem{
			{
				Type: "message",
				Role: "assistant",
				Content: []codexOutputContent{
					{Type: "output_text", Text: "hello"},
					{Type: "output_text", Text: "world"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("codexToCompletionResponse returned error: %v", err)
	}
	if resp.FinishReason != FinishLength {
		t.Fatalf("unexpected finish reason: got %q want %q", resp.FinishReason, FinishLength)
	}
	if resp.Message.Text != "hello\nworld" {
		t.Fatalf("unexpected text: got %q want %q", resp.Message.Text, "hello\nworld")
	}
}

func TestToCodexRequestBuildsFunctionCallAndToolOutputItems(t *testing.T) {
	store := false
	req, err := toCodexRequest("codex-mini-latest", CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "find config"},
			{
				Role: RoleAssistant,
				Text: "I will inspect the repo.",
				ToolCalls: []ToolCall{
					{ID: "call_1", Name: "read_file", Arguments: json.RawMessage(`{"path":"config.toml"}`)},
				},
			},
			{Role: RoleTool, ToolCallID: "call_1", Text: `{"status":"success"}`},
		},
		Tools: []ToolDef{
			{Name: "read_file", Description: "Read a file", Parameters: json.RawMessage(`{"type":"object"}`)},
		},
		ResponseOptions: ResponseOptions{
			PromptCacheKey:       "cache-key",
			PromptCacheRetention: "retain",
			SafetyIdentifier:     "user-1",
			Metadata: map[string]string{
				"channel": "bridge",
			},
			Store: &store,
		},
	})
	if err != nil {
		t.Fatalf("toCodexRequest returned error: %v", err)
	}
	if req.Instructions != "system prompt" {
		t.Fatalf("unexpected instructions: got %q want %q", req.Instructions, "system prompt")
	}
	if len(req.Input) != 3 {
		t.Fatalf("unexpected input count: got %d want 3", len(req.Input))
	}
	if req.Input[0].Type != "message" || req.Input[0].Role != "user" || req.Input[0].Content[0].Type != "input_text" {
		t.Fatalf("unexpected user input item: %+v", req.Input[0])
	}
	if req.ToolChoice != "auto" {
		t.Fatalf("unexpected tool_choice: got %q want %q", req.ToolChoice, "auto")
	}
	if req.ParallelToolCalls {
		t.Fatalf("expected parallel_tool_calls=false, got true")
	}
	if req.Input[1].Type != "function_call" || req.Input[1].Name != "read_file" {
		t.Fatalf("unexpected assistant function call item: %+v", req.Input[1])
	}
	if req.Input[2].Type != "function_call_output" || req.Input[2].CallID != "call_1" {
		t.Fatalf("unexpected tool output item: %+v", req.Input[2])
	}
	if len(req.Tools) != 1 || req.Tools[0].Type != "function" {
		t.Fatalf("unexpected tools payload: %+v", req.Tools)
	}
	if req.Tools[0].Strict {
		t.Fatalf("expected tool strict=false, got true")
	}
	if req.PromptCacheKey != "cache-key" {
		t.Fatalf("unexpected prompt_cache_key: got %q want %q", req.PromptCacheKey, "cache-key")
	}
	if req.PromptCacheRetention != "retain" {
		t.Fatalf("unexpected prompt_cache_retention: got %q want %q", req.PromptCacheRetention, "retain")
	}
	if req.SafetyIdentifier != "user-1" {
		t.Fatalf("unexpected safety_identifier: got %q want %q", req.SafetyIdentifier, "user-1")
	}
	if req.Metadata["channel"] != "bridge" {
		t.Fatalf("unexpected metadata payload: %+v", req.Metadata)
	}
	if req.Store == nil || *req.Store {
		t.Fatalf("unexpected store option: %+v", req.Store)
	}
}

func TestToCodexRequestMapsAssistantTextAsOutputText(t *testing.T) {
	req, err := toCodexRequest("gpt-5.3-codex", CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "hello"},
			{Role: RoleAssistant, Text: "world"},
		},
	})
	if err != nil {
		t.Fatalf("toCodexRequest returned error: %v", err)
	}
	if len(req.Input) != 2 {
		t.Fatalf("unexpected input count: got %d want 2", len(req.Input))
	}
	if req.Input[0].Role != "user" || len(req.Input[0].Content) != 1 || req.Input[0].Content[0].Type != "input_text" {
		t.Fatalf("unexpected user input item: %+v", req.Input[0])
	}
	if req.Input[1].Role != "assistant" || len(req.Input[1].Content) != 1 || req.Input[1].Content[0].Type != "output_text" {
		t.Fatalf("unexpected assistant input item: %+v", req.Input[1])
	}
}

func TestToCodexRequestRejectsAssistantImageContent(t *testing.T) {
	_, err := toCodexRequest("gpt-5.3-codex", CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Text: "hi"},
			{
				Role: RoleAssistant,
				Content: []ContentPart{{
					Type: ContentTypeImage,
					Image: &ImageContent{
						URL: "https://example.com/a.png",
					},
				}},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for assistant image content, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported image content role") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompleteCodexAppliesClientDefaultResponseOptions(t *testing.T) {
	requestBodies := make([]codexRequest, 0, 1)
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
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_1","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`))
	}))
	defer server.Close()

	store := false
	client := NewClientWithOptions(ClientOptions{
		Provider: ProviderCodex,
		BaseURL:  server.URL,
		Model:    "gpt-5.4",
		ResponseOptions: ResponseOptions{
			PromptCacheKey:       "default-cache-key",
			PromptCacheRetention: "retain-default",
			SafetyIdentifier:     "safe-default",
			Metadata: map[string]string{
				"source": "client-defaults",
			},
			Store: &store,
		},
	})
	client.httpClient = server.Client()

	_, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if len(requestBodies) != 1 {
		t.Fatalf("expected one request, got %d", len(requestBodies))
	}
	body := requestBodies[0]
	if body.PromptCacheKey != "default-cache-key" {
		t.Fatalf("unexpected prompt_cache_key: got %q want %q", body.PromptCacheKey, "default-cache-key")
	}
	if body.PromptCacheRetention != "retain-default" {
		t.Fatalf(
			"unexpected prompt_cache_retention: got %q want %q",
			body.PromptCacheRetention,
			"retain-default",
		)
	}
	if body.SafetyIdentifier != "safe-default" {
		t.Fatalf("unexpected safety_identifier: got %q want %q", body.SafetyIdentifier, "safe-default")
	}
	if body.Metadata["source"] != "client-defaults" {
		t.Fatalf("unexpected metadata: %+v", body.Metadata)
	}
	if body.Store == nil || *body.Store {
		t.Fatalf("unexpected store setting: %+v", body.Store)
	}
}

func TestCompleteCodexRequestOptionsOverrideClientDefaults(t *testing.T) {
	requestBodies := make([]codexRequest, 0, 1)
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
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_1","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`))
	}))
	defer server.Close()

	client := NewClientWithOptions(ClientOptions{
		Provider: ProviderCodex,
		BaseURL:  server.URL,
		Model:    "gpt-5.4",
		ResponseOptions: ResponseOptions{
			PromptCacheKey: "default-cache-key",
		},
	})
	client.httpClient = server.Client()

	_, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "hello"},
		},
		ResponseOptions: ResponseOptions{
			PromptCacheKey: "request-cache-key",
		},
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if len(requestBodies) != 1 {
		t.Fatalf("expected one request, got %d", len(requestBodies))
	}
	if requestBodies[0].PromptCacheKey != "request-cache-key" {
		t.Fatalf(
			"unexpected prompt_cache_key override: got %q want %q",
			requestBodies[0].PromptCacheKey,
			"request-cache-key",
		)
	}
}

func TestToCodexToolOutputPreservesStructuredPayload(t *testing.T) {
	output := toCodexToolOutput(Message{
		Role: RoleTool,
		Text: `{"status":"success","tool":"read_file","trace_id":"trace","output":"ok","error":""}`,
		Content: []ContentPart{
			{Type: ContentTypeText, Text: "extra"},
		},
	})
	if !strings.Contains(output, `"text":"ok"`) {
		t.Fatalf("expected encoded text payload, got %q", output)
	}
	if !strings.Contains(output, `"type":"text"`) {
		t.Fatalf("expected encoded content payload, got %q", output)
	}
}

func TestToCodexToolOutputUnwrapsEnvelopeError(t *testing.T) {
	output := toCodexToolOutput(Message{
		Role: RoleTool,
		Text: `{"status":"error","tool":"bash_exec","trace_id":"trace","output":"","error":"command failed"}`,
	})
	if output != "command failed" {
		t.Fatalf("unexpected tool output: got %q want %q", output, "command failed")
	}
}

func TestToCodexRequestUsesPreviousResponseIDAndOnlySendsIncrementalMessages(t *testing.T) {
	req, err := toCodexRequest("codex-mini-latest", CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "hello"},
			{Role: RoleAssistant, Text: "hi"},
			{Role: RoleUser, Text: "second turn"},
		},
		ConversationState: ConversationState{
			Provider:           ProviderCodex,
			BaseURL:            "https://api.openai.com/v1",
			Model:              "codex-mini-latest",
			PreviousResponseID: "resp_123",
		},
	})
	if err != nil {
		t.Fatalf("toCodexRequest returned error: %v", err)
	}
	if req.PreviousResponseID != "resp_123" {
		t.Fatalf("unexpected previous_response_id: got %q", req.PreviousResponseID)
	}
	if req.Instructions != "system prompt" {
		t.Fatalf("unexpected instructions: got %q want %q", req.Instructions, "system prompt")
	}
	if len(req.Input) != 1 {
		t.Fatalf("unexpected input count: got %d want 1", len(req.Input))
	}
	if req.Input[0].Type != "message" || req.Input[0].Role != "user" {
		t.Fatalf("unexpected incremental input roles: %+v", req.Input)
	}
	if req.Input[0].Content[0].Text != "second turn" {
		t.Fatalf("unexpected incremental user content: %+v", req.Input[0])
	}
}

func TestCodexIncrementalMessagesSkipsGraphQLToolResultBoundary(t *testing.T) {
	messages := []Message{
		{Role: RoleSystem, Text: "system prompt"},
		{Role: RoleUser, Text: "find project"},
		{Role: RoleAssistant, Text: `mutation { tfind(action: search, query: "project") }`},
		{Role: RoleTool, ToolCallID: "call_1", Text: `{"status":"success","tool":"tfind","output":"{}"}`},
		{Role: RoleAssistant, Text: `[TOOL_TAG_RESULT]
{"status":"success","tool":"tfind"}`},
	}

	got := codexIncrementalMessages(messages)
	if len(got) != 2 {
		t.Fatalf("unexpected incremental message count: got %d want 2", len(got))
	}
	if got[0].Role != RoleTool {
		t.Fatalf("expected incremental first message to stay tool result, got %+v", got[0])
	}
	if got[1].Role != RoleAssistant || !strings.Contains(got[1].Text, "[TOOL_TAG_RESULT]") {
		t.Fatalf("expected incremental second message to keep graphql feedback, got %+v", got[1])
	}
}

func TestToCodexRequestKeepsPreviousResponseIDWithTrailingGraphQLFeedback(t *testing.T) {
	req, err := toCodexRequest("codex-mini-latest", CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "find project"},
			{Role: RoleAssistant, Text: `mutation { tfind(action: search, query: "project") }`},
			{Role: RoleTool, ToolCallID: "call_1", Text: `{"status":"success","tool":"tfind","output":"{}"}`},
			{Role: RoleAssistant, Text: `[TOOL_TAG_RESULT]
{"status":"success","tool":"tfind"}`},
		},
		ConversationState: ConversationState{
			Provider:           ProviderCodex,
			BaseURL:            "https://api.openai.com/v1",
			Model:              "codex-mini-latest",
			PreviousResponseID: "resp_prev",
		},
	})
	if err != nil {
		t.Fatalf("toCodexRequest returned error: %v", err)
	}
	if req.PreviousResponseID != "resp_prev" {
		t.Fatalf("unexpected previous_response_id: got %q want %q", req.PreviousResponseID, "resp_prev")
	}
	if len(req.Input) != 2 {
		t.Fatalf("unexpected incremental input count: got %d want 2", len(req.Input))
	}
	if req.Input[0].Type != "function_call_output" {
		t.Fatalf("expected first incremental item to be tool output, got %+v", req.Input[0])
	}
	if req.Input[1].Type != "message" || req.Input[1].Role != "assistant" {
		t.Fatalf("expected second incremental item to be assistant feedback, got %+v", req.Input[1])
	}
}

func TestToCodexRequestFallsBackToStatelessReplayForGraphQLTextToolOutput(t *testing.T) {
	req, err := toCodexRequest("codex-mini-latest", CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "find project"},
			{
				Role: RoleAssistant,
				Text: `mutation { tfind(action: search, query: "project") }`,
				ToolCalls: []ToolCall{{
					ID:        "graphql-text-call-1",
					Name:      "tfind",
					Arguments: json.RawMessage(`{"action":"search","query":"project"}`),
				}},
			},
			{Role: RoleTool, ToolCallID: "graphql-text-call-1", Text: `{"status":"success","tool":"tfind","output":"{}"}`},
			{Role: RoleAssistant, Text: `[TOOL_TAG_RESULT]
{"status":"success","tool":"tfind"}`},
		},
		ConversationState: ConversationState{
			Provider:           ProviderCodex,
			BaseURL:            "https://api.openai.com/v1",
			Model:              "codex-mini-latest",
			PreviousResponseID: "resp_prev",
		},
	})
	if err != nil {
		t.Fatalf("toCodexRequest returned error: %v", err)
	}
	if req.PreviousResponseID != "" {
		t.Fatalf("expected previous_response_id cleared for graphql-text tool output, got %q", req.PreviousResponseID)
	}
	if len(req.Input) != 3 {
		t.Fatalf("unexpected stateless replay input count: got %d want 3", len(req.Input))
	}
	if req.Input[0].Type != "message" || req.Input[0].Role != "user" {
		t.Fatalf("unexpected fallback first input: %+v", req.Input[0])
	}
	if req.Input[1].Type != "function_call" || req.Input[1].CallID != "graphql-text-call-1" {
		t.Fatalf("unexpected fallback function_call input: %+v", req.Input[1])
	}
	if req.Input[2].Type != "function_call_output" || req.Input[2].CallID != "graphql-text-call-1" {
		t.Fatalf("unexpected fallback tool output input: %+v", req.Input[2])
	}
}

func TestCompleteCodexClearsPreviousResponseIDForGraphQLTextToolOutput(t *testing.T) {
	requestBodies := make([]codexRequest, 0, 1)
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
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_1","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}]}`))
	}))
	defer server.Close()

	client := NewClientWithOptions(ClientOptions{
		Provider: ProviderCodex,
		BaseURL:  server.URL,
		Model:    "gpt-5.4",
	})
	client.httpClient = server.Client()

	_, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "find project"},
			{
				Role: RoleAssistant,
				Text: `mutation { tfind(action: search, query: "project") }`,
				ToolCalls: []ToolCall{{
					ID:        "graphql-text-call-1",
					Name:      "tfind",
					Arguments: json.RawMessage(`{"action":"search","query":"project"}`),
				}},
			},
			{Role: RoleTool, ToolCallID: "graphql-text-call-1", Text: `{"status":"success","tool":"tfind","output":"{}"}`},
			{Role: RoleAssistant, Text: `[TOOL_TAG_RESULT]
{"status":"success","tool":"tfind"}`},
		},
		ConversationState: ConversationState{
			Provider:           ProviderCodex,
			BaseURL:            server.URL,
			Model:              "gpt-5.4",
			PreviousResponseID: "resp_prev",
		},
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if len(requestBodies) != 1 {
		t.Fatalf("expected one request, got %d", len(requestBodies))
	}
	if requestBodies[0].PreviousResponseID != "" {
		t.Fatalf(
			"expected previous_response_id to be cleared for graphql-text tool output, got %q",
			requestBodies[0].PreviousResponseID,
		)
	}
	if len(requestBodies[0].Input) != 3 {
		t.Fatalf("unexpected input count: got %d want 3", len(requestBodies[0].Input))
	}
	if requestBodies[0].Input[1].Type != "function_call" || requestBodies[0].Input[2].Type != "function_call_output" {
		t.Fatalf("unexpected replay input: %+v", requestBodies[0].Input)
	}
}

func TestToCodexRequestClearsPreviousResponseIDWhenIncrementalInputIsEmpty(t *testing.T) {
	req, err := toCodexRequest("codex-mini-latest", CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "hello"},
			{Role: RoleAssistant, Text: "hi"},
		},
		ConversationState: ConversationState{
			Provider:           ProviderCodex,
			BaseURL:            "https://api.openai.com/v1",
			Model:              "codex-mini-latest",
			PreviousResponseID: "resp_prev",
		},
	})
	if err != nil {
		t.Fatalf("toCodexRequest returned error: %v", err)
	}
	if req.PreviousResponseID != "" {
		t.Fatalf("expected empty previous_response_id when incremental input is empty, got %q", req.PreviousResponseID)
	}
	if len(req.Input) == 0 {
		t.Fatal("expected non-empty stateless input fallback")
	}
	if req.Input[0].Type != "message" || req.Input[0].Role != "user" {
		t.Fatalf("unexpected fallback first input: %+v", req.Input[0])
	}
}

func TestToCodexRequestKeepsAssistantMessageWithoutToolCalls(t *testing.T) {
	req, err := toCodexRequest("codex-mini-latest", CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "hello"},
			{Role: RoleAssistant, Text: "plain answer"},
			{Role: RoleUser, Text: "follow up"},
		},
	})
	if err != nil {
		t.Fatalf("toCodexRequest returned error: %v", err)
	}
	if len(req.Input) != 3 {
		t.Fatalf("unexpected input count: got %d want 3", len(req.Input))
	}
	if req.Input[1].Type != "message" || req.Input[1].Role != "assistant" || req.Input[1].Content[0].Text != "plain answer" {
		t.Fatalf("unexpected assistant replay item: %+v", req.Input[1])
	}
}

func TestToCodexRequestSanitizesToolSchemaForStrictGateways(t *testing.T) {
	req, err := toCodexRequest("codex-mini-latest", CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "hello"},
		},
		Tools: []ToolDef{
			{
				Name:        "paginate",
				Description: "Pagination helper",
				Parameters: json.RawMessage(`{
					"properties":{
						"page":{"type":"integer"},
						"filters":{"additionalProperties":true}
					},
					"required":["page"]
				}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("toCodexRequest returned error: %v", err)
	}
	if len(req.Tools) != 1 {
		t.Fatalf("unexpected tools count: got %d want 1", len(req.Tools))
	}
	if got := req.Tools[0].Parameters["type"]; got != "object" {
		t.Fatalf("unexpected sanitized top-level type: got %#v want %q", got, "object")
	}
	properties, ok := req.Tools[0].Parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected sanitized properties object, got %#v", req.Tools[0].Parameters["properties"])
	}
	if got, exists := properties["type"]; exists {
		t.Fatalf("expected no synthetic properties.type entry, got %#v", got)
	}
	page, ok := properties["page"].(map[string]any)
	if !ok {
		t.Fatalf("expected page property object, got %#v", properties["page"])
	}
	if got := page["type"]; got != "number" {
		t.Fatalf("expected integer to normalize to number, got %#v", got)
	}
	filters, ok := properties["filters"].(map[string]any)
	if !ok {
		t.Fatalf("expected filters property object, got %#v", properties["filters"])
	}
	if got := filters["type"]; got != "object" {
		t.Fatalf("expected filters type to infer object, got %#v", got)
	}
	if got := filters["properties"]; got == nil {
		t.Fatal("expected inferred object schema to include empty properties")
	}
}

func TestToCodexRequestDoesNotInjectTypeIntoPropertiesContainer(t *testing.T) {
	req, err := toCodexRequest("codex-mini-latest", CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "hello"},
		},
		Tools: []ToolDef{
			{
				Name:        "screen_control",
				Description: "browser",
				Parameters: json.RawMessage(`{
					"type":"object",
					"properties":{
						"action":{"type":"string"},
						"params":{
							"type":"object",
							"properties":{
								"url":{"type":"string"}
							},
							"additionalProperties":false
						}
					},
					"required":["action"],
					"additionalProperties":false
				}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("toCodexRequest returned error: %v", err)
	}
	if len(req.Tools) != 1 {
		t.Fatalf("unexpected tools count: got %d want 1", len(req.Tools))
	}

	rootProps, ok := req.Tools[0].Parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected root properties object, got %#v", req.Tools[0].Parameters["properties"])
	}
	if got, exists := rootProps["type"]; exists {
		t.Fatalf("expected no synthetic root properties.type entry, got %#v", got)
	}
	paramsSchema, ok := rootProps["params"].(map[string]any)
	if !ok {
		t.Fatalf("expected params schema object, got %#v", rootProps["params"])
	}
	nestedProps, ok := paramsSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested properties object, got %#v", paramsSchema["properties"])
	}
	if got, exists := nestedProps["type"]; exists {
		t.Fatalf("expected no synthetic nested properties.type entry, got %#v", got)
	}
}

func TestCompleteCodexFallsBackToStatelessReplayAfterContinuation400(t *testing.T) {
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

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_2","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"done"}]}]}`))
	}))
	defer server.Close()

	client := NewClientWithOptions(ClientOptions{
		Provider:                   ProviderCodex,
		BaseURL:                    server.URL,
		Model:                      "gpt-5.4",
		CodexStatelessRetryEnabled: true,
	})
	client.httpClient = server.Client()

	resp, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "list files"},
			{
				Role: RoleAssistant,
				Text: "I will inspect the repo.",
				ToolCalls: []ToolCall{
					{ID: "call_1", Name: "bash_exec", Arguments: json.RawMessage(`{"command":"pwd && ls -la"}`)},
				},
			},
			{Role: RoleTool, ToolCallID: "call_1", Text: "/repo\nAGENTS.md"},
		},
		ConversationState: ConversationState{
			Provider:           ProviderCodex,
			BaseURL:            server.URL,
			Model:              "gpt-5.4",
			PreviousResponseID: "resp_prev",
		},
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if resp.Message.Text != "done" {
		t.Fatalf("unexpected response text: got %q want %q", resp.Message.Text, "done")
	}
	if len(requestBodies) != 2 {
		t.Fatalf("unexpected request count: got %d want 2", len(requestBodies))
	}
	if requestBodies[0].PreviousResponseID != "" {
		t.Fatalf("unexpected first previous_response_id: got %q", requestBodies[0].PreviousResponseID)
	}
	if len(requestBodies[0].Input) != 3 {
		t.Fatalf("unexpected first request input: %+v", requestBodies[0].Input)
	}
	if requestBodies[0].Input[0].Type != "message" || requestBodies[0].Input[0].Role != "user" {
		t.Fatalf("unexpected first replay user item: %+v", requestBodies[0].Input[0])
	}
	if requestBodies[0].Input[1].Type != "function_call" || requestBodies[0].Input[2].Type != "function_call_output" {
		t.Fatalf("unexpected first replay tail: %+v", requestBodies[0].Input)
	}
	if requestBodies[1].PreviousResponseID != "" {
		t.Fatalf("unexpected fallback previous_response_id: got %q", requestBodies[1].PreviousResponseID)
	}
	if len(requestBodies[1].Input) != 3 {
		t.Fatalf("unexpected fallback input count: got %d want 3", len(requestBodies[1].Input))
	}
	if requestBodies[1].Input[0].Type != "message" || requestBodies[1].Input[0].Role != "user" {
		t.Fatalf("unexpected fallback first input: %+v", requestBodies[1].Input[0])
	}
	if got := requestBodies[1].Input[0].Content[0].Text; got != "list files" && !strings.Contains(got, "Current request: list files") {
		t.Fatalf("unexpected fallback user prompt: %+v", requestBodies[1].Input[0])
	}
	if requestBodies[1].Input[1].Type != "function_call" {
		t.Fatalf("unexpected fallback function call item: %+v", requestBodies[1].Input[1])
	}
	if requestBodies[1].Input[2].Type != "function_call_output" {
		t.Fatalf("unexpected fallback tool output item: %+v", requestBodies[1].Input[2])
	}
}

func TestCompleteCodexFallsBackToStatelessReplayAfterMissingFunctionCallOutputError(t *testing.T) {
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
			_, _ = w.Write([]byte(`{"error":{"message":"No tool call found for function call output with call_id call_1.","type":"invalid_request_error","code":null}}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_2","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"done"}]}]}`))
	}))
	defer server.Close()

	client := NewClientWithOptions(ClientOptions{
		Provider:                   ProviderCodex,
		BaseURL:                    server.URL,
		Model:                      "gpt-5.4",
		CodexStatelessRetryEnabled: true,
	})
	client.httpClient = server.Client()

	resp, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "list files"},
			{
				Role: RoleAssistant,
				Text: "I will inspect the repo.",
				ToolCalls: []ToolCall{
					{ID: "call_1", Name: "bash_exec", Arguments: json.RawMessage(`{"command":"pwd && ls -la"}`)},
				},
			},
			{Role: RoleTool, ToolCallID: "call_1", Text: "/repo\nAGENTS.md"},
		},
		ConversationState: ConversationState{
			Provider:           ProviderCodex,
			BaseURL:            server.URL,
			Model:              "gpt-5.4",
			PreviousResponseID: "resp_prev",
		},
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if resp.Message.Text != "done" {
		t.Fatalf("unexpected response text: got %q want %q", resp.Message.Text, "done")
	}
	if len(requestBodies) != 2 {
		t.Fatalf("unexpected request count: got %d want 2", len(requestBodies))
	}
	if requestBodies[0].PreviousResponseID != "" {
		t.Fatalf("unexpected first previous_response_id: got %q", requestBodies[0].PreviousResponseID)
	}
	if len(requestBodies[0].Input) != 3 {
		t.Fatalf("unexpected first request input: %+v", requestBodies[0].Input)
	}
	if requestBodies[0].Input[0].Type != "message" || requestBodies[0].Input[0].Role != "user" {
		t.Fatalf("unexpected first replay user item: %+v", requestBodies[0].Input[0])
	}
	if requestBodies[0].Input[1].Type != "function_call" || requestBodies[0].Input[2].Type != "function_call_output" {
		t.Fatalf("unexpected first replay tail: %+v", requestBodies[0].Input)
	}
	if requestBodies[1].PreviousResponseID != "" {
		t.Fatalf("unexpected fallback previous_response_id: got %q", requestBodies[1].PreviousResponseID)
	}
	if len(requestBodies[1].Input) != 3 {
		t.Fatalf("unexpected fallback input count: got %d want 3", len(requestBodies[1].Input))
	}
	if requestBodies[1].Input[1].Type != "function_call" || requestBodies[1].Input[2].Type != "function_call_output" {
		t.Fatalf("unexpected fallback replay tail: %+v", requestBodies[1].Input)
	}
}

func TestCompleteCodexFallsBackToStatelessReplayForPlainFollowUpTurn(t *testing.T) {
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

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_3","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"follow-up ok"}]}]}`))
	}))
	defer server.Close()

	client := NewClientWithOptions(ClientOptions{
		Provider:                   ProviderCodex,
		BaseURL:                    server.URL,
		Model:                      "gpt-5.4",
		CodexStatelessRetryEnabled: true,
	})
	client.httpClient = server.Client()

	resp, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "hello"},
			{Role: RoleAssistant, Text: "first answer"},
			{Role: RoleUser, Text: "second question"},
		},
		ConversationState: ConversationState{
			Provider:           ProviderCodex,
			BaseURL:            server.URL,
			Model:              "gpt-5.4",
			PreviousResponseID: "resp_prev",
		},
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if resp.Message.Text != "follow-up ok" {
		t.Fatalf("unexpected response text: got %q want %q", resp.Message.Text, "follow-up ok")
	}
	if len(requestBodies) != 2 {
		t.Fatalf("unexpected request count: got %d want 2", len(requestBodies))
	}
	if requestBodies[0].PreviousResponseID != "resp_prev" {
		t.Fatalf("unexpected first previous_response_id: got %q", requestBodies[0].PreviousResponseID)
	}
	if len(requestBodies[0].Input) != 1 || requestBodies[0].Input[0].Role != "user" {
		t.Fatalf("unexpected first request input: %+v", requestBodies[0].Input)
	}
	if requestBodies[1].PreviousResponseID != "" {
		t.Fatalf("unexpected fallback previous_response_id: got %q", requestBodies[1].PreviousResponseID)
	}
	if len(requestBodies[1].Input) != 1 {
		t.Fatalf("unexpected fallback input count: got %d want 1", len(requestBodies[1].Input))
	}
	if requestBodies[1].Input[0].Type != "message" || requestBodies[1].Input[0].Role != "user" {
		t.Fatalf("unexpected fallback user item: %+v", requestBodies[1].Input[0])
	}
	if !strings.Contains(requestBodies[1].Input[0].Content[0].Text, "Assistant: first answer") {
		t.Fatalf("unexpected fallback user prompt: %+v", requestBodies[1].Input[0])
	}
}

func TestToCodexRequestWithOptionsStatelessCompressesHistoryIntoSyntheticUserPrompt(t *testing.T) {
	req, err := toCodexRequestWithOptions("codex-mini-latest", CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "first"},
			{Role: RoleAssistant, Text: "tool preamble", ToolCalls: []ToolCall{{ID: "call_1", Name: "list_files", Arguments: json.RawMessage(`{"path":"."}`)}}},
			{Role: RoleTool, ToolCallID: "call_1", Text: "Directory: ."},
			{Role: RoleAssistant, Text: "first answer"},
			{Role: RoleUser, Text: "second"},
		},
	}, codexRequestOptions{ForceStateless: true})
	if err != nil {
		t.Fatalf("toCodexRequestWithOptions returned error: %v", err)
	}
	if len(req.Input) != 1 {
		t.Fatalf("unexpected input count: got %d want 1", len(req.Input))
	}
	if req.Input[0].Type != "message" || req.Input[0].Role != "user" {
		t.Fatalf("unexpected input item: %+v", req.Input[0])
	}
	text := req.Input[0].Content[0].Text
	if !strings.Contains(text, "User: first") || !strings.Contains(text, "Assistant: first answer") || !strings.Contains(text, "Current request: second") {
		t.Fatalf("unexpected synthetic prompt: %q", text)
	}
}

func TestCompleteCodexDoesNotFallbackToStatelessReplayByDefault(t *testing.T) {
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

	client := NewClientWithOptions(ClientOptions{
		Provider: ProviderCodex,
		BaseURL:  server.URL,
		Model:    "gpt-5.4",
	})
	client.httpClient = server.Client()

	_, err := client.Complete(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Text: "system prompt"},
			{Role: RoleUser, Text: "hello"},
		},
		ConversationState: ConversationState{
			Provider:           ProviderCodex,
			BaseURL:            server.URL,
			Model:              "gpt-5.4",
			PreviousResponseID: "resp_prev",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected upstream 400 error without fallback, got %v", err)
	}
	if len(requestBodies) != 1 {
		t.Fatalf("expected one request without fallback, got %d", len(requestBodies))
	}
}
