package tools

import "testing"

func TestStaticVisibleToolNames_HidesOnDemandToolsByDefault(t *testing.T) {
	visible := StaticVisibleToolNames([]string{
		"ask_human",
		"script_exec",
		"graphql_query",
		"graphql_schema_lookup",
		"graphql_mutation",
	}, VisibilityOptions{})

	if containsTool(visible, "graphql_query") || containsTool(visible, "graphql_schema_lookup") || containsTool(visible, "graphql_mutation") {
		t.Fatalf("expected on-demand graphql tools to stay hidden, got %v", visible)
	}
	if !containsTool(visible, "script_exec") {
		t.Fatalf("expected static tool to remain visible, got %v", visible)
	}
}

func TestSearchCandidateToolNames_IncludeOnDemandGraphQLTools(t *testing.T) {
	candidates := SearchCandidateToolNames([]string{
		"ask_human",
		"script_exec",
		"graphql_query",
		"graphql_schema_lookup",
		"graphql_mutation",
		"tfind",
	}, nil, VisibilityOptions{
		ToolSearchEnabled: true,
	})

	if !containsTool(candidates, "graphql_query") || !containsTool(candidates, "graphql_schema_lookup") || !containsTool(candidates, "graphql_mutation") {
		t.Fatalf("unexpected tfind candidates: %v", candidates)
	}
}

func containsTool(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
