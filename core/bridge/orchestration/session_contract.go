package orchestration

import (
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
	bridgesession "ghost-os/bridge/session"
)

type SessionHistoryBuilder = sessionturn.SessionHistoryBuilder
type messageProjectionOptions = sessionturn.ProjectionOptions
type resolvedHumanQuestion = sessionturn.ResolvedHumanQuestion

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

func buildSessionTurnDraftPayload(
	sess *bridgesession.Session,
	includeDraft bool,
) *sessionTurnDraft {
	return sessionturn.BuildSessionTurnDraftPayload(sess, includeDraft)
}

func buildAssistantDraftSessionMessage(sess *bridgesession.Session) (sessionMessage, bool) {
	return sessionturn.BuildAssistantDraftSessionMessage(sess)
}

func newSessionHistoryBuilder(
	provider bridgeconfig.ProviderConfig,
	systemPrompt string,
	sessionStore *bridgesession.Store,
	idleTurns int,
	microcompactEnabled bool,
	traceID string,
) *SessionHistoryBuilder {
	return sessionturn.NewSessionHistoryBuilder(
		provider,
		systemPrompt,
		sessionStore,
		idleTurns,
		microcompactEnabled,
		traceID,
	)
}

func agentMessageForResolvedHumanTool(
	toolCallID string,
	toolName string,
	traceID string,
	output string,
) llm.Message {
	return sessionturn.AgentMessageForResolvedHumanTool(toolCallID, toolName, traceID, output)
}

func messagesWithSystemPrompt(messages []llm.Message, systemPrompt string) []llm.Message {
	return sessionturn.MessagesWithSystemPrompt(messages, systemPrompt)
}

func projectMessagesForModel(messages []llm.Message, options messageProjectionOptions) []llm.Message {
	return sessionturn.ProjectMessagesForModel(messages, options)
}

func sanitizeToolProtocolMessages(messages []llm.Message) []llm.Message {
	return sessionturn.SanitizeToolProtocolMessages(messages)
}

func estimateMessagesTokens(messages []llm.Message) int {
	return sessionturn.EstimateMessagesTokens(messages)
}
