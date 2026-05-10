package orchestration

import (
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
	bridgesession "ghost-os/bridge/session"
)

func buildSessionTurnDraftPayload(
	sess *bridgesession.Session,
	includeDraft bool,
) *sessionTurnDraft {
	return sessionturn.BuildSessionTurnDraftPayload(sess, includeDraft)
}

func buildAssistantDraftSessionMessage(sess *bridgesession.Session) (sessionMessage, bool) {
	return sessionturn.BuildAssistantDraftSessionMessage(sess)
}
