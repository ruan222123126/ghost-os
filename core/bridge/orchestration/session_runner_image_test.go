package orchestration

import (
	"context"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

func TestSessionAgentRunnerRunTurnInputPersistsUserImages(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
			FinishReason: llm.FinishStop,
		}},
	}
	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg:          Config{MaxTurns: 3, PromptsPath: "", Provider: ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"}},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "base system prompt",
		},
	}, nil, sessionStore, nil)

	input := llm.Message{Role: llm.RoleUser, Text: "describe this", Content: []llm.ContentPart{{Type: llm.ContentTypeImage, Image: &llm.ImageContent{URL: "data:image/png;base64,ZmFrZS1pbWFnZQ==", MimeType: "image/png"}}}}
	if _, sessionID, err := runner.RunTurnInput(context.Background(), input, "", "trace-image-input"); err != nil {
		t.Fatalf("run turn input: %v", err)
	} else if len(completer.requests) != 1 || len(completer.requests[0].Messages) < 2 {
		t.Fatalf("unexpected completer requests: %+v", completer.requests)
	} else if got := completer.requests[0].Messages[1]; got.Role != llm.RoleUser || len(got.Content) != 1 || got.Content[0].Image == nil {
		t.Fatalf("expected user image content in model request, got %+v", got)
	} else if loaded, err := sessionStore.Load(sessionID); err != nil {
		t.Fatalf("load session: %v", err)
	} else if len(loaded.Messages) < 2 || len(loaded.Messages[1].Content) != 1 || loaded.Messages[1].Content[0].Image == nil {
		t.Fatalf("expected persisted user image content, got %+v", loaded.Messages)
	} else if loaded.Messages[1].Content[0].Image.URL != "data:image/png;base64,ZmFrZS1pbWFnZQ==" {
		t.Fatalf("unexpected persisted image url: %q", loaded.Messages[1].Content[0].Image.URL)
	}
}
