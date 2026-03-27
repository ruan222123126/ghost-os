package tools

import (
	"strings"
	"testing"
)

func TestFormatGraphQLToolRuntimePrompt_IncludesMinimalExamplesForVisibleTools(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewAskHumanTool())
	registry.Register(NewMemoryManageTool(nil))
	registry.Register(NewScriptExecTool(nil))
	registry.Register(NewWebSearchTool(WebSearchConfig{}))
	registry.Register(NewWebRooterTool(WebRooterConfig{}))
	registry.Register(NewToolSearchTool(registry, VisibilityOptions{}, 2))

	prompt := FormatGraphQLToolRuntimePrompt(registry)

	for _, snippet := range []string{
		"Minimal successful examples:",
		"Prefer copying the closest minimal successful example",
		"- `ask_human`: include a final custom option so the user can type their own answer.",
		`mutation { ask_human(prompt: "Which environment should I use?", options: [{label: "staging"}, {label: "Other", allow_custom: true}]) }`,
		"- `memory_manage`: use stable URIs; for `update` or `delete`, discover the exact URI first with `read` or `list`.",
		`mutation { memory_manage(operation: create, uri: "user://preferences/editor", content: "Prefer vim keybindings") }`,
		"- `script_exec`: minimal sandbox execution mutation.",
		`mutation { script_exec(script: "print(\"ok\")") }`,
		"- `tfind`: after `action: load`, the loaded tool is available in the same user turn on the next completion.",
		`mutation { tfind(action: load, tool_names: ["browser_control"]) }`,
		"- `web_search`:",
		`mutation { web_search(query: "OpenAI API docs") }`,
		"- `web_rooter`: single read-only action wrapper for the pinned web-rooter HTTP service.",
		`mutation { web_rooter(action: internet_search, params: {query: "OpenAI API docs", num_results: 5, auto_crawl: false}) }`,
	} {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to contain %q, got %q", snippet, prompt)
		}
	}
	if strings.Contains(prompt, "type Query {") {
		t.Fatalf("expected query block to be removed, got %q", prompt)
	}
	examples := prompt[strings.Index(prompt, "Minimal successful examples:"):]
	if strings.Contains(examples, `web_search(max_results:`) || strings.Contains(examples, `web_search(provider:`) {
		t.Fatalf("expected web_search example to stay minimal, got %q", examples)
	}
	if strings.Contains(examples, `web_rooter(action: internet_search)`) {
		t.Fatalf("expected web_rooter example to stay explicit, got %q", examples)
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

func TestFormatGraphQLToolRuntimePrompt_UsesSingleMutationBlock(t *testing.T) {
	tests := []struct {
		name     string
		registry *Registry
		want     string
	}{
		{
			name:     "empty",
			registry: NewRegistry(),
			want:     "No GraphQL tools are available in Mutation for this turn.",
		},
		{
			name: "with-tool",
			registry: func() *Registry {
				registry := NewRegistry()
				registry.Register(NewAskHumanTool())
				return registry
			}(),
			want: "type Mutation {",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			prompt := FormatGraphQLToolRuntimePrompt(testCase.registry)
			if !strings.Contains(prompt, testCase.want) {
				t.Fatalf("expected prompt to contain %q, got %q", testCase.want, prompt)
			}
			if strings.Contains(prompt, "type Query {") || strings.Contains(prompt, "_empty: JSON") {
				t.Fatalf("expected prompt to stay mutation-only without placeholders, got %q", prompt)
			}
		})
	}
}
