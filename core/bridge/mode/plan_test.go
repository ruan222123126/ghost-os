package mode

import (
	"errors"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
)

func TestValidateResponseRejectsToolCalls(t *testing.T) {
	_, _, err := ValidateResponse(&llm.CompletionResponse{
		Message:      llm.Message{ToolCalls: []llm.ToolCall{{ID: "c1", Name: "script_exec"}}},
		FinishReason: llm.FinishToolCalls,
	})
	if !errors.Is(err, ErrToolCallsForbidden) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateResponseRejectsToolTagSyntax(t *testing.T) {
	_, _, err := ValidateResponse(&llm.CompletionResponse{
		Message:      llm.Message{Text: "<t:1>{}\n</t>"},
		FinishReason: llm.FinishStop,
	})
	if !errors.Is(err, ErrToolTagForbidden) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildSystemPromptIncludesTemplateSections(t *testing.T) {
	prompt := BuildSystemPrompt("base")
	if !strings.Contains(prompt, TemplateIntent) || !strings.Contains(prompt, TemplateTasks) {
		t.Fatalf("missing template sections: %q", prompt)
	}
	if !strings.Contains(prompt, MainExecutorLiteral) {
		t.Fatalf("missing main executor literal: %q", prompt)
	}
}
