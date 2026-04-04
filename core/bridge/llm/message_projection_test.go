package llm

import "testing"

func TestProjectMessagesForCompletionKeepsStoredInternalRoleUntouched(t *testing.T) {
	input := []Message{
		{Role: RoleSystem, Text: "system"},
		{Role: RoleInternal, Text: "[TOOL_TAG_RESULT]\n{\"tool\":\"web_search\",\"output\":{\"items\":[{\"title\":\"OpenAI\"}]}}"},
	}

	projected := ProjectMessagesForCompletion(input)
	if len(projected) != 2 {
		t.Fatalf("unexpected projected length: got %d", len(projected))
	}
	if projected[1].Role != RoleAssistant {
		t.Fatalf("expected provider-safe assistant role, got %+v", projected[1])
	}
	if input[1].Role != RoleInternal {
		t.Fatalf("expected source slice to remain internal, got %+v", input[1])
	}
}
