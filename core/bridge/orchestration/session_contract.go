package orchestration

import (
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
	bridgesession "ghost-os/bridge/session"
)

func buildSessionMetadataPayload(summary bridgesession.SessionMetadata) sessionMetadata {
	return sessionturn.BuildSessionMetadataPayload(summary)
}

func buildSessionDetailPayload(
	sess *bridgesession.Session,
	page bridgesession.MessagePage,
	includeDraft bool,
) sessionDetail {
	return sessionturn.BuildSessionDetailPayload(sess, page, includeDraft)
}

func buildSessionMessagePagePayload(page bridgesession.MessagePage) sessionMessagePage {
	return sessionturn.BuildSessionMessagePagePayload(page)
}

func buildSessionMessagePayload(index int, message llm.Message) sessionMessage {
	return sessionturn.BuildSessionMessagePayload(index, message)
}
