package guiagent

import (
	"strings"
	"testing"
)

func TestSystemPromptDefinesActionTypeContract(t *testing.T) {
	for _, snippet := range []string{
		`fields "thought" and "action"`,
		`use field "type"`,
		`"action":{"type":"finished"}`,
		`Never use {"action":{"action":"..."}}.`,
	} {
		if !strings.Contains(systemPrompt, snippet) {
			t.Fatalf("expected system prompt to contain %q, got %q", snippet, systemPrompt)
		}
	}
}
