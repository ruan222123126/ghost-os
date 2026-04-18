package orchestration

import (
	"strings"

	"ghost-os/bridge/llm"
	bridgesession "ghost-os/bridge/session"
)

func buildAssistantDraftSessionMessage(sess *bridgesession.Session) (sessionMessage, bool) {
	if sess == nil || sess.AssistantDraft == nil {
		return sessionMessage{}, false
	}
	if strings.TrimSpace(sess.AssistantDraft.Text) == "" {
		return sessionMessage{}, false
	}
	return sessionMessage{
		Index:      sess.MessageCount,
		Role:       string(llm.RoleAssistant),
		Text:       sess.AssistantDraft.Text,
		InProgress: true,
	}, true
}
