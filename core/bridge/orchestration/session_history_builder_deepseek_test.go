package orchestration

import (
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

func TestSessionHistoryBuilderKeepsLongHistoryWhenProviderContextWindowIsConfigured(t *testing.T) {
	sess := session.NewSession("system")
	for i := 0; i < 12; i++ {
		sess.AddMessage(llm.Message{
			Role: llm.RoleUser,
			Text: strings.Repeat("browser tool replay payload ", 120),
		})
	}

	before := sess.Messages
	builder := newSessionHistoryBuilder(
		bridgeconfig.ProviderConfig{
			Type:                llm.ProviderCustom,
			Model:               "deepseek-v4-pro",
			ContextWindowTokens: 1000000,
		},
		"system",
		nil,
		3,
		false,
		"",
	)
	history := builder.BuildHistory(sess)
	after := history.Messages()

	if len(after) != len(before) {
		t.Fatalf("unexpected pruned message count: got %d want %d", len(after), len(before))
	}
	if after[len(after)-1].Text != before[len(before)-1].Text {
		t.Fatalf("expected last message to remain intact")
	}
}
