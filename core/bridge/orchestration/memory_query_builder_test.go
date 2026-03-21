package orchestration

import (
	"strings"
	"testing"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
)

func TestBuildMemoryRecallQuerySkipsInternalMessages(t *testing.T) {
	history := agent.NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleUser, Text: "Reply in Chinese."},
		{Role: llm.RoleInternal, Text: "[GRAPHQL_EXECUTION_RESULT]\n{\"data\":{\"viewer\":{\"id\":\"1\"}}}"},
		{Role: llm.RoleAssistant, Text: "I will reply in Chinese."},
	})

	query := buildMemoryRecallQuery(history, "continue")
	if strings.Contains(query, "GRAPHQL_EXECUTION_RESULT") {
		t.Fatalf("internal graphql execution feedback should not appear in recall query, got %q", query)
	}
	if !strings.Contains(query, "Reply in Chinese.") || !strings.Contains(query, "I will reply in Chinese.") {
		t.Fatalf("expected user and assistant context in recall query, got %q", query)
	}
}
