package tools

import (
	"strings"
	"testing"
)

func TestWebRooterToolDefStaysReadOnlyAndGraphQLVisibleAsQuery(t *testing.T) {
	tool := NewWebRooterTool(WebRooterConfig{})
	def := ToolDefFromTool(tool)
	if !def.Semantics.ReadOnly || def.Semantics.SideEffect {
		t.Fatalf("unexpected web_rooter semantics: %+v", def.Semantics)
	}

	registry := NewRegistry()
	registry.Register(tool)
	schema := BuildGraphQLToolRuntimeSchema(registry)
	if !hasGraphQLToolRuntimeField(schema.QueryFields, webRooterToolName) {
		t.Fatalf("expected web_rooter query field, got %+v", schema.QueryFields)
	}
	if hasGraphQLToolRuntimeField(schema.MutationFields, webRooterToolName) {
		t.Fatalf("web_rooter should not be exposed as mutation: %+v", schema.MutationFields)
	}
}

func TestGetToolMetadataIncludesWebRooterResearchMetadata(t *testing.T) {
	for _, item := range GetToolMetadata() {
		if item.Name != webRooterToolName {
			continue
		}
		if item.Domain != "web" {
			t.Fatalf("unexpected web_rooter domain: %+v", item)
		}
		if got := strings.Join(item.Tags, ","); got != "research,citation,crawl,academic" {
			t.Fatalf("unexpected web_rooter tags: %q", got)
		}
		want := "Citation-rich web research via external web-rooter service."
		if item.ShortDesc != want {
			t.Fatalf("unexpected web_rooter short description: %q", item.ShortDesc)
		}
		return
	}
	t.Fatal("expected web_rooter metadata entry")
}

func hasGraphQLToolRuntimeField(fields []GraphQLToolRuntimeField, name string) bool {
	for _, field := range fields {
		if field.Name == name {
			return true
		}
	}
	return false
}
