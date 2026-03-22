package tools

import (
	"strings"
	"testing"
)

func TestFormatGraphQLToolRuntimePrompt_IncludesMinimalExamplesForVisibleTools(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewAskHumanTool())
	registry.Register(NewScriptExecTool(nil))
	registry.Register(NewWebSearchTool(WebSearchConfig{}))
	registry.Register(NewToolSearchTool(registry, VisibilityOptions{}, 2))

	prompt := FormatGraphQLToolRuntimePrompt(registry)

	for _, snippet := range []string{
		"Minimal successful examples:",
		"- `ask_human`: include a final custom option so the user can type their own answer.",
		`mutation { ask_human(prompt: "Which environment should I use?", options: [{label: "staging"}, {label: "Other", allow_custom: true}]) }`,
		"- `script_exec`: minimal sandbox execution mutation.",
		`mutation { script_exec(script: "print(\"ok\")") }`,
		"- `tfind`: after `action: \"load\"`, the loaded tool is available next turn, not in the same response.",
		`mutation { tfind(action: "load", tool_names: ["browser_control"]) }`,
		"- `web_search`:",
		`query { web_search(query: "OpenAI API docs") }`,
	} {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to contain %q, got %q", snippet, prompt)
		}
	}
	examples := prompt[strings.Index(prompt, "Minimal successful examples:"):]
	if strings.Contains(examples, `web_search(max_results:`) || strings.Contains(examples, `web_search(provider:`) {
		t.Fatalf("expected web_search example to stay minimal, got %q", examples)
	}
}

func TestFormatGraphQLToolRuntimePrompt_ExamplesRespectCatalogVisibility(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewAskHumanTool())
	registry.Register(NewScriptExecTool(nil))

	prompt := FormatGraphQLToolRuntimePrompt(NewScopedCatalog(registry, []string{AskHumanToolName}))

	if !strings.Contains(prompt, "- `ask_human`:") {
		t.Fatalf("expected visible tool example, got %q", prompt)
	}
	if strings.Contains(prompt, "- `script_exec`:") {
		t.Fatalf("expected hidden tool example to stay hidden, got %q", prompt)
	}
}
