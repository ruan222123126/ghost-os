package orchestration

import (
	"context"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

func TestSessionRunnerGraphQLModeKeepsDefaultSystemPrompt(t *testing.T) {
	sessionStore := newTempSessionStore(t)
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
			},
			client:       completer,
			registry:     tools.NewRegistry(),
			graphQL:      &tools.GraphQLSourceRegistry{},
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
	protocolIndex := strings.Index(prompt, "GraphQL Text Protocol:")
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
}
