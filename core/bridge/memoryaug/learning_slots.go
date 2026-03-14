package memoryaug

import (
	"strings"

	"ghost-os/bridge/memorystore"
)

func (s *learningService) acceptsCandidate(candidate Candidate) bool {
	return candidate.Confidence >= s.settings.MinConfidence &&
		memorystore.NormalizeMemoryKey(candidate.MemoryKey) != ""
}

func (s *learningService) normalizeCandidate(
	candidate Candidate,
	input LearnFromTurnInput,
) (memorystore.MemoryEntry, []string, bool) {
	spec, ok := slotSpecForKey(candidate.MemoryKey)
	if !ok {
		return memorystore.MemoryEntry{}, nil, false
	}
	value := normalizeSlotValue(spec, candidate)
	if value == "" {
		return memorystore.MemoryEntry{}, nil, false
	}
	return memorystore.MemoryEntry{
		ScopeType:  s.resolveSlotScope(spec),
		ScopeID:    s.resolveSlotScopeID(spec, input),
		SourceKind: memorystore.SourceKindLearned,
		MemoryType: spec.MemoryType,
		MemoryKey:  spec.Key,
		Summary:    spec.Summary,
		Content:    renderSlotContent(spec, value),
		Metadata:   buildSlotMetadata(value, candidate.Reason),
		Confidence: candidate.Confidence,
		Status:     memorystore.MemoryStatusActive,
	}, normalizeIDs(candidate.SupersedesID), true
}

func (s *learningService) resolveSlotScope(spec SlotSpec) string {
	if spec.DefaultScope == memorystore.ScopeTypeSession && s.settings.SessionScopeEnabled {
		return memorystore.ScopeTypeSession
	}
	if s.settings.UserScopeEnabled {
		return memorystore.ScopeTypeUser
	}
	return memorystore.ScopeTypeSession
}

func (s *learningService) resolveSlotScopeID(spec SlotSpec, input LearnFromTurnInput) string {
	if s.resolveSlotScope(spec) == memorystore.ScopeTypeSession {
		return input.SessionID
	}
	return input.UserScope
}

func candidateLabel(candidate Candidate) string {
	if key := memorystore.NormalizeMemoryKey(candidate.MemoryKey); key != "" {
		return key
	}
	if summary := strings.TrimSpace(candidate.Summary); summary != "" {
		return summary
	}
	return strings.TrimSpace(candidate.Content)
}
