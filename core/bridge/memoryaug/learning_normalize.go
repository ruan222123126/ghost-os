package memoryaug

import (
	"strings"

	"ghost-os/bridge/memorystore"
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

func resolveCandidateMemoryKey(candidate Candidate, memoryType string) string {
	key := memorystore.NormalizeMemoryKey(candidate.MemoryKey)
	if key != "" {
		return key
	}
	return memorystore.DefaultMemoryKey(memoryType, strings.TrimSpace(candidate.Summary))
}
