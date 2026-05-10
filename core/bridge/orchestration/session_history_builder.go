package orchestration

import (
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
	"ghost-os/bridge/session"
)

type SessionHistoryBuilder = sessionturn.SessionHistoryBuilder
type resolvedHumanQuestion = sessionturn.ResolvedHumanQuestion

func newSessionHistoryBuilder(
	provider bridgeconfig.ProviderConfig,
	systemPrompt string,
	sessionStore *session.Store,
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
