package memoryaug

import (
	"strings"
)

func normalizeLearnInput(input LearnFromTurnInput, settings Settings) LearnFromTurnInput {
	userScope := strings.TrimSpace(input.UserScope)
	if userScope == "" {
		userScope = settings.UserScopeID
	}
	return LearnFromTurnInput{
		SessionID: strings.TrimSpace(input.SessionID),
		UserScope: userScope,
		TraceID:   strings.TrimSpace(input.TraceID),
		Messages:  append([]TurnMessage(nil), input.Messages...),
	}
}
