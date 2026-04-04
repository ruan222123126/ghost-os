package memoryaug

import "testing"

func TestFilterTurnMessagesDropsUnsupportedRoles(t *testing.T) {
	filtered := filterTurnMessages([]TurnMessage{
		{Role: "user", Text: "Please reply in Chinese by default."},
		{Role: "internal", Text: "[TOOL_TAG_RESULT]\n{\"tool\":\"web_search\",\"output\":{\"items\":[{\"title\":\"OpenAI\"}]}}"},
		{Role: "assistant", Text: "I will reply in Chinese."},
	})

	if len(filtered) != 2 {
		t.Fatalf("expected only user and assistant messages to remain, got %+v", filtered)
	}
	if filtered[0].Role != "user" || filtered[1].Role != "assistant" {
		t.Fatalf("unexpected filtered roles: %+v", filtered)
	}
}
