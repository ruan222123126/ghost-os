package webrooter

import (
	"strings"
	"testing"
)

func TestNormalizeResponse_UsesDataFieldsFromNestedPayload(t *testing.T) {
	payload := map[string]any{
		"success": true,
		"content": "ok",
		"data": map[string]any{
			"citations":       []any{map[string]any{"id": "W1"}},
			"references_text": "[W1] https://openai.com/docs",
			"comparison":      map[string]any{"summary": "kept"},
			"mindsearch_compat": map[string]any{
				"enabled": true,
			},
		},
		"urls":     []any{"https://openai.com/docs"},
		"error":    nil,
		"metadata": map[string]any{"source": "web-rooter"},
	}

	result, err := NormalizeResponse("research", payload)
	if err != nil {
		t.Fatalf("NormalizeResponse: %v", err)
	}
	if len(result.Citations) != 1 {
		t.Fatalf("expected one citation, got %+v", result.Citations)
	}
	if result.ReferencesText != "[W1] https://openai.com/docs" {
		t.Fatalf("unexpected references text: %q", result.ReferencesText)
	}
}

func TestNormalizeResponse_RejectsUpstreamSuccessFalseWithoutDataObjectRequirement(t *testing.T) {
	payload := map[string]any{
		"success":  false,
		"content":  "fetch failed",
		"data":     nil,
		"urls":     []any{},
		"error":    "connection reset",
		"metadata": map[string]any{},
	}

	_, err := NormalizeResponse("fetch", payload)
	if err == nil || !strings.Contains(err.Error(), "upstream returned success=false") {
		t.Fatalf("expected upstream failure, got %v", err)
	}
}

func TestNormalizeResponse_RequiresDataObjectWhenSuccessIsTrue(t *testing.T) {
	payload := map[string]any{
		"success":  true,
		"content":  "ok",
		"data":     nil,
		"urls":     []any{},
		"error":    nil,
		"metadata": map[string]any{},
	}

	_, err := NormalizeResponse("research", payload)
	if err == nil || !strings.Contains(err.Error(), `response field "data" must be an object`) {
		t.Fatalf("expected data object type error, got %v", err)
	}
}
