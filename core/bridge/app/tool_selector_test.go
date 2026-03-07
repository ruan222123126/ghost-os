package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/llm"
)

type fakeSelectorCompleter struct {
	response       *llm.CompletionResponse
	err            error
	waitForContext bool
	requests       []llm.CompletionRequest
}

func (f *fakeSelectorCompleter) Complete(ctx context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, request)
	if f.waitForContext {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.response, f.err
}

func newSelectorTestConfig() Config {
	return Config{
		ToolSelectorEnabled:    true,
		ToolSelectorMode:       "llm",
		ToolSelectorTimeoutMS:  20,
		ToolSelectorConfidence: 0.75,
		ToolSelectorRecentMsgs: 6,
	}
}

func selectorResponse(text string) *llm.CompletionResponse {
	return &llm.CompletionResponse{Message: llm.Message{Role: llm.RoleAssistant, Text: text}}
}

func TestToolSelector_FallbackOnErrorCases(t *testing.T) {
	tests := []struct {
		name      string
		completer *fakeSelectorCompleter
		cfg       Config
	}{
		{
			name:      "parse error",
			completer: &fakeSelectorCompleter{response: selectorResponse("not-json")},
			cfg:       newSelectorTestConfig(),
		},
		{
			name:      "low confidence",
			completer: &fakeSelectorCompleter{response: selectorResponse(`{"mode":"subset","tools":["read_file"],"confidence":0.40,"reason":"too weak"}`)},
			cfg:       newSelectorTestConfig(),
		},
		{
			name:      "unknown tool",
			completer: &fakeSelectorCompleter{response: selectorResponse(`{"mode":"subset","tools":["ghost_tool","ask_human"],"confidence":0.95,"reason":"bad tool"}`)},
			cfg:       newSelectorTestConfig(),
		},
		{
			name:      "empty list",
			completer: &fakeSelectorCompleter{response: selectorResponse(`{"mode":"subset","tools":[],"confidence":0.95,"reason":"empty"}`)},
			cfg:       newSelectorTestConfig(),
		},
		{
			name:      "timeout",
			completer: &fakeSelectorCompleter{waitForContext: true},
			cfg:       newSelectorTestConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewToolSelector(tt.cfg, tt.completer)
			result := selector.SelectTools(context.Background(), "inspect config.go", nil, "", "trace-1")
			if result.Mode != "all" {
				t.Fatalf("expected fallback mode all, got %q", result.Mode)
			}
			if !result.Fallback {
				t.Fatal("expected fallback=true")
			}
		})
	}
}

func TestToolSelector_AlwaysIncludesAskHuman(t *testing.T) {
	selector := NewToolSelector(newSelectorTestConfig(), &fakeSelectorCompleter{
		response: selectorResponse(`{"mode":"subset","tools":["read_file"],"confidence":0.91,"reason":"focused code read"}`),
	})

	result := selector.SelectTools(context.Background(), "read config.go", nil, "", "trace-2")
	if result.Mode != "subset" || result.Fallback {
		t.Fatalf("expected subset result, got %+v", result)
	}
	if !containsToolName(result.Tools, "ask_human") {
		t.Fatalf("expected ask_human in tools, got %v", result.Tools)
	}
}

func TestToolSelector_AutoExpandsScriptExec(t *testing.T) {
	selector := NewToolSelector(newSelectorTestConfig(), &fakeSelectorCompleter{
		response: selectorResponse(`{"mode":"subset","tools":["script_exec","ask_human"],"confidence":0.95,"reason":"complex script task"}`),
	})

	result := selector.SelectTools(context.Background(), "do a complex scripted edit", nil, "", "trace-3")
	if result.Mode != "subset" || result.Fallback {
		t.Fatalf("expected subset result, got %+v", result)
	}
	for _, name := range []string{"script_exec", "read_file", "search_files", "bash_exec", "ask_human"} {
		if !containsToolName(result.Tools, name) {
			t.Fatalf("expected %q in tools, got %v", name, result.Tools)
		}
	}
}

func TestToolSelector_UsesTimeoutContext(t *testing.T) {
	cfg := newSelectorTestConfig()
	cfg.ToolSelectorTimeoutMS = 10
	completer := &fakeSelectorCompleter{waitForContext: true}
	selector := NewToolSelector(cfg, completer)

	started := time.Now()
	result := selector.SelectTools(context.Background(), "read file", nil, "", "trace-timeout")
	if time.Since(started) > time.Second {
		t.Fatal("selector timeout should return quickly")
	}
	if result.Mode != "all" || !result.Fallback {
		t.Fatalf("expected timeout fallback, got %+v", result)
	}
	if !errors.Is(result.Error, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded error, got %v", result.Error)
	}
}

func TestToolSelector_BuildPromptIncludesDecisionHintSection(t *testing.T) {
	selector := NewToolSelector(newSelectorTestConfig(), &fakeSelectorCompleter{})
	prompt := selector.buildSelectorPrompt("patch config.go", []llm.Message{{Role: llm.RoleUser, Text: "inspect config first"}}, "Similar successful cases used: read_file, apply_diff")
	if !strings.Contains(prompt, "Prior similar experience:\nSimilar successful cases used: read_file, apply_diff") {
		t.Fatalf("expected decision hint section, got %q", prompt)
	}
	if !strings.Contains(prompt, "\nCurrent request:\npatch config.go") {
		t.Fatalf("expected current request section, got %q", prompt)
	}
}

func TestToolSelector_BuildPromptOmitsDecisionHintWhenEmpty(t *testing.T) {
	selector := NewToolSelector(newSelectorTestConfig(), &fakeSelectorCompleter{})
	prompt := selector.buildSelectorPrompt("patch config.go", nil, "")
	if strings.Contains(prompt, "Prior similar experience:") {
		t.Fatalf("expected prompt without decision hint section, got %q", prompt)
	}
}

func TestToolSelector_SelectToolsPassesDecisionHintToWorker(t *testing.T) {
	completer := &fakeSelectorCompleter{response: selectorResponse(`{"mode":"subset","tools":["read_file"],"confidence":0.93,"reason":"focused code read"}`)}
	selector := NewToolSelector(newSelectorTestConfig(), completer)
	hint := "Similar successful cases used: read_file, apply_diff"

	result := selector.SelectTools(context.Background(), "patch config.go", nil, hint, "trace-hint")
	if result.Mode != "subset" || result.Fallback {
		t.Fatalf("expected subset result, got %+v", result)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected one worker request, got %d", len(completer.requests))
	}
	if got := completer.requests[0].Messages[1].Text; !strings.Contains(got, "Prior similar experience:\n"+hint) {
		t.Fatalf("expected worker prompt to include decision hint, got %q", got)
	}
}
