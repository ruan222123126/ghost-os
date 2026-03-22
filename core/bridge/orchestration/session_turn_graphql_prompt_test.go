package orchestration

import (
	"context"
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
			cfg: Config{
				MaxTurns:    3,
				PromptsPath: "",
				Provider:    ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
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
	protocolIndex := strings.Index(prompt, "GraphQL Tool Call Protocol:")
	if !strings.Contains(prompt, "You are Ghost-OS bridge agent, an AI-driven digital twin execution layer.") {
		t.Fatalf("expected default system prompt to remain, got %q", prompt)
	}
	if coreJobIndex == -1 {
		t.Fatalf("expected default system prompt sections to remain, got %q", prompt)
	}
	if protocolIndex == -1 {
		t.Fatalf("expected GraphQL protocol to be appended, got %q", prompt)
	}
	if protocolIndex < coreJobIndex {
		t.Fatalf("expected GraphQL protocol to augment the default prompt, got %q", prompt)
	}
	if strings.Contains(prompt, "structured tool schema") {
		t.Fatalf("expected structured tool schema guidance to be removed in graphql mode, got %q", prompt)
	}
	if !strings.Contains(prompt, "Use `ask_human` only when blocked on required user input") {
		t.Fatalf("expected graphql mode to keep ask_human guidance, got %q", prompt)
	}
	if !strings.Contains(prompt, "ask_human(") {
		t.Fatalf("expected graphql schema summary to include ask_human field, got %q", prompt)
	}
	if len(completer.requests[0].Tools) != 0 {
		t.Fatalf("expected graphql mode to hide native tool defs, got %+v", completer.requests[0].Tools)
	}
}
