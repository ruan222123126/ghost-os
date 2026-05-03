package orchestration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

func TestCompletionPromptRefreshLoadsSkillContextSameRun(t *testing.T) {
	projectRoot := t.TempDir()
	skillDir := filepath.Join(projectRoot, ".agents", "skills", "release")
	writePromptRefreshSkillFile(t, skillDir, "release_flow", "release skill", "Release body.")

	sess := session.NewSession("")
	sess.AdvanceToolTurn(3)
	registry := tools.NewRegistry()
	registry.Register(tools.NewToolSearchTool(
		registry,
		tools.VisibilityOptions{ToolSearchEnabled: true},
		3,
		tools.ToolSearchOptions{ProjectRoot: projectRoot},
	))
	catalog := tools.NewScopedCatalog(registry, []string{"sfind"})
	cfg := bridgeconfig.Config{
		MaxTurns:    3,
		ProjectRoot: projectRoot,
		PromptsDir:  filepath.Join(t.TempDir(), "prompts"),
		ToolSearch:  bridgeconfig.ToolSearchConfig{Enabled: true, IdleTurns: 3},
	}
	prompt, err := bridgeruntime.BuildSystemPromptForSession(cfg, catalog, sess, cfg.ToolSearch.IdleTurns)
	if err != nil {
		t.Fatalf("BuildSystemPromptForSession: %v", err)
	}

	completer := &promptRefreshCompleter{
		responses: []*llm.CompletionResponse{
			promptRefreshToolCall("call-1", "sfind", `{"action":"load","skill_names":["release_flow"]}`),
			promptRefreshStop("done"),
		},
	}
	deps := agentRuntimeDependencies{cfg: cfg, client: completer, registry: registry}
	runAgent := newSessionTurnPreparer(nil, nil, nil, nil, nil).buildTurnAgent(
		deps,
		catalog,
		sess,
		agent.NewHistory(prompt),
	)

	ctx := tools.WithSession(context.Background(), sess)
	if _, err := runAgent.RunMessageWithTraceID(ctx, llm.Message{Role: llm.RoleUser, Text: "load it"}, "trace"); err != nil {
		t.Fatalf("RunMessageWithTraceID: %v", err)
	}
	if len(completer.requests) != 2 {
		t.Fatalf("expected two completion requests, got %d", len(completer.requests))
	}
	if !strings.Contains(completer.requests[1].Messages[0].Text, "Release body.") {
		t.Fatalf("expected refreshed prompt to include loaded skill body, got %q", completer.requests[1].Messages[0].Text)
	}
}

type promptRefreshCompleter struct {
	requests  []llm.CompletionRequest
	responses []*llm.CompletionResponse
}

func (f *promptRefreshCompleter) Complete(
	_ context.Context,
	request llm.CompletionRequest,
) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, request)
	index := len(f.requests) - 1
	return f.responses[index], nil
}

func promptRefreshToolCall(id string, name string, arguments string) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		FinishReason: llm.FinishToolCalls,
		Message: llm.Message{
			Role: llm.RoleAssistant,
			ToolCalls: []llm.ToolCall{
				{ID: id, Name: name, Arguments: json.RawMessage(arguments)},
			},
		},
	}
}

func promptRefreshStop(text string) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		FinishReason: llm.FinishStop,
		Message: llm.Message{
			Role: llm.RoleAssistant,
			Text: text,
		},
	}
}

func writePromptRefreshSkillFile(
	t *testing.T,
	dir string,
	name string,
	description string,
	body string,
) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}
}
