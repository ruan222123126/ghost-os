package memoryaug

import "testing"

func TestFilterTurnMessagesDropsUnsupportedRoles(t *testing.T) {
	filtered := filterTurnMessages([]TurnMessage{
		{Role: "user", Text: "Please reply in Chinese by default."},
		{Role: "internal", Text: "[GRAPHQL_EXECUTION_RESULT]\n{\"data\":{\"viewer\":{\"id\":\"1\"}}}"},
		{Role: "assistant", Text: "I will reply in Chinese."},
	})

	if len(filtered) != 2 {
		t.Fatalf("expected only user and assistant messages to remain, got %+v", filtered)
	}
	if filtered[0].Role != "user" || filtered[1].Role != "assistant" {
		t.Fatalf("unexpected filtered roles: %+v", filtered)
	}
}
