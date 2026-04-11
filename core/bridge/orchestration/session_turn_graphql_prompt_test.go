package orchestration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

func TestSessionRunnerGraphQLModeKeepsDefaultSystemPrompt(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	registry := tools.NewRegistry()
	registry.Register(tools.NewAskHumanTool())
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "noted"},
			FinishReason: llm.FinishStop,
		}},
	}
	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: bridgeconfig.Config{
				MaxTurns:    3,
				PromptsPath: "",
				Provider:    bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
				ToolSelector: bridgeconfig.ToolSelectorConfig{
					Allowlist: []string{"ask_human"},
				},
				GraphQL: bridgeconfig.GraphQLConfig{
					ToolRuntimeEnabled: true,
				},
			},
			client:       completer,
			registry:     registry,
			systemPrompt: "runtime fallback prompt",
		},
	}, nil, sessionStore, nil)

	if _, _, err := runner.RunTurn(context.Background(), "query viewer", "", "trace-graphql-system-prompt"); err != nil {
		t.Fatalf("run turn: %v", err)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}

	prompt := completer.requests[0].Messages[0].Text
	coreJobIndex := strings.Index(prompt, "## Core Job")
	protocolIndex := strings.Index(prompt, "[System Instruction]")
	if !strings.Contains(prompt, "You are Ghost-OS bridge agent, an AI-driven digital twin execution layer.") {
		t.Fatalf("expected default system prompt to remain, got %q", prompt)
	}
	if coreJobIndex == -1 {
		t.Fatalf("expected default system prompt sections to remain, got %q", prompt)
	}
	if protocolIndex == -1 {
		t.Fatalf("expected tagged tool protocol to be appended, got %q", prompt)
	}
	if protocolIndex < coreJobIndex {
		t.Fatalf("expected tagged protocol to augment the default prompt, got %q", prompt)
	}
	if strings.Contains(prompt, "structured tool schema") {
		t.Fatalf("expected structured tool schema guidance to be removed in graphql mode, got %q", prompt)
	}
	if !strings.Contains(prompt, "Use `ask_human` only when blocked on required user input") {
		t.Fatalf("expected graphql mode to keep ask_human guidance, got %q", prompt)
	}
	if !strings.Contains(prompt, "Tool name: ask_human") {
		t.Fatalf("expected tool id list to include ask_human, got %q", prompt)
	}
	if !strings.Contains(prompt, "strictly use <t:TOOL_ID>JSON_ARGS</t>") {
		t.Fatalf("expected tagged protocol format in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "Parameter format:") {
		t.Fatalf("expected parameter shape section in prompt, got %q", prompt)
	}
	if len(completer.requests[0].Tools) != 0 {
		t.Fatalf("expected graphql mode to hide native tool defs, got %+v", completer.requests[0].Tools)
	}
}

func TestSessionRunnerGraphQLModeRefreshesPromptAfterDynamicLoadInSameTurn(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	registry := tools.NewRegistry()
	cfg := bridgeconfig.Config{
		MaxTurns:    4,
		PromptsPath: "",
		Provider:    bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			AllowlistOnly: true,
			Allowlist:     []string{tools.ToolSearchToolName},
		},
		ToolSearch: bridgeconfig.ToolSearchConfig{
			Enabled:   true,
			IdleTurns: 3,
		},
		GraphQL: bridgeconfig.GraphQLConfig{
			ToolRuntimeEnabled:  true,
			TextSanitizeEnabled: true,
		},
	}
	registry.Register(&graphQLPromptRefreshWebSearchTool{})
	registry.Register(tools.NewToolSearchTool(registry, visibilityOptionsFromConfig(cfg), cfg.ToolSearch.IdleTurns))

	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message:      llm.Message{Role: llm.RoleAssistant, Text: `<t:1>{"action":"load","tool_names":["web_search"]}</t>`},
				FinishReason: llm.FinishStop,
			},
			{
				Message:      llm.Message{Role: llm.RoleAssistant, Text: `<t:2>{"query":"OpenAI API docs"}</t>`},
				FinishReason: llm.FinishStop,
			},
			{
				Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
				FinishReason: llm.FinishStop,
			},
		},
	}
	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg:          cfg,
			client:       completer,
			registry:     registry,
			systemPrompt: "runtime fallback prompt",
		},
	}, nil, sessionStore, nil)

	output, _, err := runner.RunTurn(context.Background(), "find docs", "", "trace-graphql-refresh")
	if err != nil {
		t.Fatalf("run turn: %v", err)
	}
	if output != "done" {
		t.Fatalf("unexpected output: %q", output)
	}
	if len(completer.requests) != 3 {
		t.Fatalf("expected three completion requests, got %d", len(completer.requests))
	}

	firstPrompt := completer.requests[0].Messages[0].Text
	if strings.Contains(firstPrompt, "Tool name: web_search") {
		t.Fatalf("expected first completion prompt to exclude web_search before load, got %q", firstPrompt)
	}
	secondPrompt := completer.requests[1].Messages[0].Text
	if !strings.Contains(secondPrompt, "Tool name: web_search") {
		t.Fatalf("expected second completion prompt to include web_search after load, got %q", secondPrompt)
	}
	if !strings.Contains(secondPrompt, "`web_search` was loaded in this user turn and is available now.") {
		t.Fatalf("expected refreshed dynamic tool state, got %q", secondPrompt)
	}
}

type graphQLPromptRefreshWebSearchTool struct{}

func (*graphQLPromptRefreshWebSearchTool) Name() string {
	return "web_search"
}

func (*graphQLPromptRefreshWebSearchTool) Description() string {
	return "test web search"
}

func (*graphQLPromptRefreshWebSearchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"query":{"type":"string"}
		},
		"required":["query"]
	}`)
}

func (*graphQLPromptRefreshWebSearchTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{ReadOnly: true}
}

func (*graphQLPromptRefreshWebSearchTool) Execute(context.Context, json.RawMessage, string) (string, error) {
	return `{"items":[{"title":"OpenAI API docs"}]}`, nil
}

func visibilityOptionsFromConfig(cfg bridgeconfig.Config) tools.VisibilityOptions {
	return tools.VisibilityOptions{
		ToolSearchEnabled: cfg.ToolSearch.Enabled,
		AllowlistOnly:     cfg.ToolSelector.AllowlistOnly,
		Allowlist:         append([]string(nil), cfg.ToolSelector.Allowlist...),
		Blocklist:         append([]string(nil), cfg.ToolSelector.Blocklist...),
	}
}
