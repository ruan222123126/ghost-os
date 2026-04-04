package orchestration

import (
	"testing"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
)

func TestBuildPlannerRecentMessagesSkipsInternalMessages(t *testing.T) {
	history := agent.NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleUser, Text: "Reply in Chinese."},
		{Role: llm.RoleInternal, Text: "[TOOL_TAG_RESULT]\n{\"tool\":\"web_search\"}"},
		{Role: llm.RoleAssistant, Text: "I will reply in Chinese."},
	})

	messages := buildPlannerRecentMessages(history, "continue")
	if len(messages) != 3 {
		t.Fatalf("expected recent user/assistant context plus current message, got %+v", messages)
	}
	if messages[0].Text != "Reply in Chinese." || messages[1].Text != "I will reply in Chinese." {
		t.Fatalf("unexpected planner messages: %+v", messages)
	}
}
