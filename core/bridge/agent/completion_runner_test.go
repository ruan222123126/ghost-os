package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

func TestCompletionRunnerCompleteDoesNotMutateHistory(t *testing.T) {
	completer := newFakeCompleter(&llm.CompletionResponse{
		Message: llm.Message{
			Text: "follow-up",
		},
		FinishReason: llm.FinishStop,
		ConversationState: llm.ConversationState{
			Provider:           llm.ProviderCodex,
			BaseURL:            "https://api.openai.com/v1",
			Model:              "codex-mini-latest",
			PreviousResponseID: "resp_next",
		},
	})
	history := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "hello"},
	})
	history.SetConversationState(llm.ConversationState{
		Provider:           llm.ProviderCodex,
		BaseURL:            "https://api.openai.com/v1",
		Model:              "codex-mini-latest",
		PreviousResponseID: "resp_prev",
	})

	runner := newCompletionRunner(completer, newFakeToolCatalog(), history, llm.ResponseOptions{})
	resp, err := runner.complete(context.Background(), nil, "trace-runner", "", 0)
	if err != nil {
		t.Fatalf("complete returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response but got nil")
	}
	if resp.Message.Role != llm.RoleAssistant {
		t.Fatalf("expected assistant role normalization, got %q", resp.Message.Role)
	}
	if len(history.Messages()) != 2 {
		t.Fatalf("complete should not append assistant message to history, got %d messages", len(history.Messages()))
	}
	if got := history.ConversationState().PreviousResponseID; got != "resp_prev" {
		t.Fatalf("complete should not update conversation state: got %q want %q", got, "resp_prev")
	}
}

func TestCompletionRunnerPassesResponseOptionsToRequest(t *testing.T) {
	store := true
	options := llm.ResponseOptions{
		PromptCacheKey:       " cache-key ",
		PromptCacheRetention: " sticky ",
		SafetyIdentifier:     " user-123 ",
		Metadata: map[string]string{
			"trace": " session-1 ",
		},
		Store: &store,
	}
	completer := newFakeCompleter(&llm.CompletionResponse{
		Message:      llm.Message{Role: llm.RoleAssistant, Text: "ok"},
		FinishReason: llm.FinishStop,
	})
	history := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "hello"},
	})

	runner := newCompletionRunner(completer, newFakeToolCatalog(), history, options)
	if _, err := runner.complete(context.Background(), nil, "trace-runner", "", 0); err != nil {
		t.Fatalf("complete returned error: %v", err)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	got := completer.requests[0].ResponseOptions
	if got.PromptCacheKey != "cache-key" {
		t.Fatalf("unexpected prompt_cache_key: got %q want %q", got.PromptCacheKey, "cache-key")
	}
	if got.PromptCacheRetention != "sticky" {
		t.Fatalf("unexpected prompt_cache_retention: got %q want %q", got.PromptCacheRetention, "sticky")
	}
	if got.SafetyIdentifier != "user-123" {
		t.Fatalf("unexpected safety_identifier: got %q want %q", got.SafetyIdentifier, "user-123")
	}
	if got.Metadata["trace"] != "session-1" {
		t.Fatalf("unexpected metadata map: %+v", got.Metadata)
	}
	if got.Store == nil || !*got.Store {
		t.Fatalf("unexpected store option: %+v", got.Store)
	}
}

func TestCompletionRunnerProjectsInternalMessagesForProvider(t *testing.T) {
	completer := newFakeCompleter(&llm.CompletionResponse{
		Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
		FinishReason: llm.FinishStop,
	})
	history := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "hello"},
		{Role: llm.RoleInternal, Text: "[GRAPHQL_TOOL_RESULT]\n{\"tool\":\"web_search\",\"output\":{\"items\":[{\"title\":\"OpenAI\"}]}}"},
	})

	runner := newCompletionRunner(completer, newFakeToolCatalog(), history, llm.ResponseOptions{})
	if _, err := runner.complete(context.Background(), nil, "trace-runner", "", 0); err != nil {
		t.Fatalf("complete returned error: %v", err)
	}

	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	request := completer.requests[0]
	last := request.Messages[len(request.Messages)-1]
	if last.Role != llm.RoleAssistant {
		t.Fatalf("expected projected assistant role for provider request, got %+v", last)
	}
	if history.Messages()[2].Role != llm.RoleInternal {
		t.Fatalf("expected persisted history role to stay internal, got %+v", history.Messages()[2])
	}
}

func TestCompletionRunnerCompleteReturnsErrorOnNilResponse(t *testing.T) {
	runner := newCompletionRunner(
		newFakeCompleter(nil),
		newFakeToolCatalog(),
		NewHistoryFromMessages([]llm.Message{{Role: llm.RoleUser, Text: "hello"}}),
		llm.ResponseOptions{},
	)

	_, err := runner.complete(context.Background(), nil, "trace-runner", "", 0)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), errCompletionRunnerResponseRequired.Error()) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompletionRunnerRepairsGraphQLTextToolProtocolForProviderRequest(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if err := json.Unmarshal(raw, &requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"done"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	completer := llm.NewClientWithOptions(llm.ClientOptions{
		Provider: llm.ProviderOpenAI,
		BaseURL:  server.URL,
		ChatPath: "/chat/completions",
		Model:    "gpt-4o-mini",
	})
	tool := newStaticTool("web_search", `{"items":[{"title":"OpenAI"}]}`)
	tool.semantics = llm.ToolSemantics{ReadOnly: true}
	history := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "hello"},
		{Role: llm.RoleAssistant, Text: `query { web_search(query: "OpenAI") }`},
		{
			Role:       llm.RoleTool,
			ToolCallID: "graphql-text-call-1",
			Text:       `{"status":"success","tool":"web_search","trace_id":"trace-1","output":"{\"items\":[{\"title\":\"OpenAI\"}]}"}`,
		},
		{Role: llm.RoleInternal, Text: "[GRAPHQL_TOOL_RESULT]\n{\"tool\":\"web_search\",\"output\":{\"items\":[{\"title\":\"OpenAI\"}]}}"},
	})

	runner := newCompletionRunner(completer, newFakeToolCatalog(tool), history, llm.ResponseOptions{})
	if _, err := runner.complete(context.Background(), nil, "trace-runner", "", 1); err != nil {
		t.Fatalf("complete returned error: %v", err)
	}

	messages, ok := requestBody["messages"].([]any)
	if !ok || len(messages) != 5 {
		t.Fatalf("unexpected provider messages payload: %#v", requestBody["messages"])
	}

	assistantMessage, ok := messages[2].(map[string]any)
	if !ok {
		t.Fatalf("unexpected assistant message payload: %#v", messages[2])
	}
	toolCalls, ok := assistantMessage["tool_calls"].([]any)
	if !ok || len(toolCalls) != 1 {
		t.Fatalf("expected repaired tool_calls in assistant message, got %#v", assistantMessage)
	}
	call, ok := toolCalls[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected tool_call payload: %#v", toolCalls[0])
	}
	if got := call["id"]; got != "graphql-text-call-1" {
		t.Fatalf("unexpected tool call id: got %#v want %q", got, "graphql-text-call-1")
	}
	function, ok := call["function"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected function payload: %#v", call["function"])
	}
	if got := function["name"]; got != "web_search" {
		t.Fatalf("unexpected tool name: got %#v want %q", got, "web_search")
	}
	if got := function["arguments"]; got != `{"query":"OpenAI"}` {
		t.Fatalf("unexpected tool arguments: got %#v want %q", got, `{"query":"OpenAI"}`)
	}

	storedMessages := history.Messages()
	if len(storedMessages[2].ToolCalls) != 0 {
		t.Fatalf("expected persisted assistant transcript to remain text-only, got %+v", storedMessages[2])
	}
	if storedMessages[4].Role != llm.RoleInternal {
		t.Fatalf("expected internal message to stay internal in history, got %+v", storedMessages[4])
	}
}

func TestCompletionRunnerRepairsGraphQLTextBatchToolProtocolForProviderRequest(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if err := json.Unmarshal(raw, &requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"done"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	completer := llm.NewClientWithOptions(llm.ClientOptions{
		Provider: llm.ProviderOpenAI,
		BaseURL:  server.URL,
		ChatPath: "/chat/completions",
		Model:    "gpt-4o-mini",
	})
	webSearchTool := newStaticTool("web_search", `{"items":[{"title":"OpenAI"}]}`)
	webSearchTool.semantics = llm.ToolSemantics{ReadOnly: true}
	scriptExecTool := newStaticTool("script_exec", `{"status":"ok"}`)
	history := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "hello"},
		{Role: llm.RoleAssistant, Text: `mutation { web_search(query: "OpenAI") } mutation { script_exec(script: "print('ok')") }`},
		{
			Role:       llm.RoleTool,
			ToolCallID: "graphql-text-call-1",
			Text:       `{"status":"success","tool":"web_search","trace_id":"trace-1","output":"{\"items\":[{\"title\":\"OpenAI\"}]}"}`,
		},
		{Role: llm.RoleInternal, Text: "[GRAPHQL_TOOL_RESULT]\n{\"tool\":\"web_search\",\"output\":{\"items\":[{\"title\":\"OpenAI\"}]}}"},
		{
			Role:       llm.RoleTool,
			ToolCallID: "graphql-text-call-2",
			Text:       `{"status":"success","tool":"script_exec","trace_id":"trace-1","output":"{\"status\":\"ok\"}"}`,
		},
		{Role: llm.RoleInternal, Text: "[GRAPHQL_TOOL_RESULT]\n{\"tool\":\"script_exec\",\"output\":{\"status\":\"ok\"}}"},
	})

	runner := newCompletionRunner(completer, newFakeToolCatalog(webSearchTool, scriptExecTool), history, llm.ResponseOptions{})
	if _, err := runner.complete(context.Background(), nil, "trace-runner", "", 1); err != nil {
		t.Fatalf("complete returned error: %v", err)
	}

	messages, ok := requestBody["messages"].([]any)
	if !ok || len(messages) != 7 {
		t.Fatalf("unexpected provider messages payload: %#v", requestBody["messages"])
	}

	assistantMessage, ok := messages[2].(map[string]any)
	if !ok {
		t.Fatalf("unexpected assistant message payload: %#v", messages[2])
	}
	toolCalls, ok := assistantMessage["tool_calls"].([]any)
	if !ok || len(toolCalls) != 2 {
		t.Fatalf("expected repaired batch tool_calls in assistant message, got %#v", assistantMessage)
	}
	firstCall, ok := toolCalls[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected first tool_call payload: %#v", toolCalls[0])
	}
	secondCall, ok := toolCalls[1].(map[string]any)
	if !ok {
		t.Fatalf("unexpected second tool_call payload: %#v", toolCalls[1])
	}
	if got := firstCall["id"]; got != "graphql-text-call-1" {
		t.Fatalf("unexpected first tool call id: got %#v want %q", got, "graphql-text-call-1")
	}
	if got := secondCall["id"]; got != "graphql-text-call-2" {
		t.Fatalf("unexpected second tool call id: got %#v want %q", got, "graphql-text-call-2")
	}
	firstFunction, ok := firstCall["function"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected first function payload: %#v", firstCall["function"])
	}
	secondFunction, ok := secondCall["function"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected second function payload: %#v", secondCall["function"])
	}
	if got := firstFunction["name"]; got != "web_search" {
		t.Fatalf("unexpected first tool name: got %#v want %q", got, "web_search")
	}
	if got := secondFunction["name"]; got != "script_exec" {
		t.Fatalf("unexpected second tool name: got %#v want %q", got, "script_exec")
	}
	if got := firstFunction["arguments"]; got != `{"query":"OpenAI"}` {
		t.Fatalf("unexpected first tool arguments: got %#v want %q", got, `{"query":"OpenAI"}`)
	}
	if got := secondFunction["arguments"]; got != `{"script":"print('ok')"}` {
		t.Fatalf("unexpected second tool arguments: got %#v want %q", got, `{"script":"print('ok')"}`)
	}
}

func TestCompletionRunnerRepairsGraphQLTextToolProtocolWithHiddenCatalog(t *testing.T) {
	completer := newFakeCompleter(&llm.CompletionResponse{
		Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
		FinishReason: llm.FinishStop,
	})
	tool := newStaticTool("web_search", `{"items":[{"title":"OpenAI"}]}`)
	tool.semantics = llm.ToolSemantics{ReadOnly: true}
	baseCatalog := newFakeToolCatalog(tool)
	history := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "hello"},
		{Role: llm.RoleAssistant, Text: `query { web_search(query: "OpenAI") }`},
		{
			Role:       llm.RoleTool,
			ToolCallID: "graphql-text-call-1",
			Text:       `{"status":"success","tool":"web_search","trace_id":"trace-1","output":"{\"items\":[{\"title\":\"OpenAI\"}]}"}`,
		},
		{Role: llm.RoleInternal, Text: "[GRAPHQL_TOOL_RESULT]\n{\"tool\":\"web_search\",\"output\":{\"items\":[{\"title\":\"OpenAI\"}]}}"},
	})

	runner := newCompletionRunner(completer, tools.NewStructuredToolHiddenCatalog(baseCatalog), history, llm.ResponseOptions{})
	if _, err := runner.complete(context.Background(), nil, "trace-runner", "", 1); err != nil {
		t.Fatalf("complete returned error: %v", err)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	request := completer.requests[0]
	if len(request.Tools) != 0 {
		t.Fatalf("expected hidden catalog to keep native tool defs hidden, got %+v", request.Tools)
	}
	if len(request.Messages) < 4 || len(request.Messages[2].ToolCalls) != 1 {
		t.Fatalf("expected repaired tool call in provider request, got %+v", request.Messages)
	}
	if got := request.Messages[2].ToolCalls[0].Name; got != "web_search" {
		t.Fatalf("unexpected repaired tool name: %q", got)
	}
}
