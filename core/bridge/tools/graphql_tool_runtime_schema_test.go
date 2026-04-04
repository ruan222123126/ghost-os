package tools

import (
	"strings"
	"testing"
)

func TestFormatGraphQLToolRuntimePrompt_UsesTaggedToolProtocol(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewAskHumanTool())
	registry.Register(NewScriptExecTool(nil))

	prompt := FormatGraphQLToolRuntimePrompt(registry)

	for _, snippet := range []string{
		"[System Instruction]",
		"strictly use <t:TOOL_ID>JSON_ARGS</t>",
		"ID: 1",
		"Tool name:",
		"Parameter format:",
		"<t:1>",
	} {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to contain %q, got %q", snippet, prompt)
		}
	}
	if strings.Contains(prompt, "mutation {") || strings.Contains(prompt, "GraphQL Tool Call Protocol") {
		t.Fatalf("expected prompt to avoid GraphQL mutation protocol wording, got %q", prompt)
	}
}

func TestFormatGraphQLToolRuntimePrompt_RespectsCatalogVisibility(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewAskHumanTool())
	registry.Register(NewScriptExecTool(nil))

	prompt := FormatGraphQLToolRuntimePrompt(NewScopedCatalog(registry, []string{AskHumanToolName}))

	if !strings.Contains(prompt, "ask_human") {
		t.Fatalf("expected visible tool in prompt, got %q", prompt)
	}
	if strings.Contains(prompt, "script_exec") {
		t.Fatalf("expected hidden tool to stay hidden, got %q", prompt)
	}
}

func TestFormatGraphQLToolRuntimePrompt_HandlesEmptyCatalog(t *testing.T) {
	prompt := FormatGraphQLToolRuntimePrompt(NewRegistry())
	if !strings.Contains(prompt, "No tools are available for this turn.") {
		t.Fatalf("expected empty-catalog hint, got %q", prompt)
	}
}
