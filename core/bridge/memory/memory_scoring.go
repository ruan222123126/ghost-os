package memory

import (
	"math"
	"sort"
	"strings"
	"time"
)

const (
	defaultTemporalDecayHalfLife = 72 * time.Hour
	defaultAnchorMinWeight       = 0.65
	defaultEvolutionBatchSize    = 20
)

type memoryScoringConfig struct {
	TemporalDecayEnabled  bool
	TemporalDecayHalfLife time.Duration
	AnchorEnabled         bool
	AnchorMinWeight       float64
}

type scoredMemoryEntry struct {
	Entry MemoryEntry
	Score float64
}

func defaultMemoryScoringConfig() memoryScoringConfig {
	return memoryScoringConfig{
		TemporalDecayEnabled:  true,
		TemporalDecayHalfLife: defaultTemporalDecayHalfLife,
		AnchorEnabled:         true,
		AnchorMinWeight:       defaultAnchorMinWeight,
	}
}

func newMemoryScoringConfig(config MemoryConfig) memoryScoringConfig {
	out := defaultMemoryScoringConfig()
	out.TemporalDecayEnabled = config.Warm.TemporalDecayEnabled
	if config.Warm.TemporalDecayHalfLife > 0 {
		out.TemporalDecayHalfLife = config.Warm.TemporalDecayHalfLife
	}
	out.AnchorEnabled = config.Warm.AnchorEnabled
	if config.Warm.AnchorMinWeight > 0 {
		out.AnchorMinWeight = clamp01(config.Warm.AnchorMinWeight)
	}
	return out
}

func rankMemoryEntries(entries []MemoryEntry, query MemoryQuery, now time.Time, config memoryScoringConfig) []MemoryEntry {
	if len(entries) == 0 {
		return nil
	}

	scored := make([]scoredMemoryEntry, 0, len(entries))
	for _, entry := range entries {
		scored = append(scored, scoredMemoryEntry{
			Entry: cloneEntry(entry),
			Score: memoryScore(entry, query, now, config),
		})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		if scored[i].Entry.Importance != scored[j].Entry.Importance {
			return scored[i].Entry.Importance > scored[j].Entry.Importance
		}
		if scored[i].Entry.Timestamp.Equal(scored[j].Entry.Timestamp) {
			return scored[i].Entry.ID < scored[j].Entry.ID
		}
		return scored[i].Entry.Timestamp.After(scored[j].Entry.Timestamp)
	})
	out := make([]MemoryEntry, 0, len(scored))
	for _, item := range scored {
		out = append(out, cloneEntry(item.Entry))
	}
	return out
}

func memoryScore(entry MemoryEntry, query MemoryQuery, now time.Time, config memoryScoringConfig) float64 {
	relevance := semanticRelevanceScore(entry, query)
	importance := entry.Importance
	if importance <= 0 {
		importance = calculateImportance(entry)
	}
	temporal := temporalDecayScore(entry, now, config)
	anchor := anchorBoostScore(entry, query, now, config)
	access := accessBoostScore(entry)
	layerBias := memoryLayerBias(entry)
	base := 0.35 + importance*0.25 + temporal*0.2 + anchor*0.15 + access*0.05
	return relevance*base + layerBias
}

func entryTextMatchesQuery(entry MemoryEntry, query MemoryQuery) bool {
	phrase := strings.ToLower(strings.TrimSpace(query.SemanticQuery))
	terms := queryTerms(query)
	if phrase == "" && len(terms) == 0 {
		return true
	}
	text := searchableEntryText(entry)
	if text == "" {
		return false
	}
	if phrase != "" && strings.Contains(text, phrase) {
		return true
	}
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func semanticRelevanceScore(entry MemoryEntry, query MemoryQuery) float64 {
	phrase := strings.ToLower(strings.TrimSpace(query.SemanticQuery))
	terms := queryTerms(query)
	if phrase == "" && len(terms) == 0 {
		return 1
	}
	text := searchableEntryText(entry)
	if text == "" {
		return 0
	}

	phraseHit := 0.0
	if phrase != "" && strings.Contains(text, phrase) {
		phraseHit = 1
	}

	matched := 0
	for _, term := range terms {
		if strings.Contains(text, term) {
			matched++
		}
	}
	termScore := 0.0
	if len(terms) > 0 {
		termScore = float64(matched) / float64(len(terms))
	}

	return clamp01(0.15 + termScore*0.65 + phraseHit*0.2)
}

func temporalDecayScore(entry MemoryEntry, now time.Time, config memoryScoringConfig) float64 {
	if !config.TemporalDecayEnabled && !queryRequestsRecencyBoost(entry) {
		return clamp01(1 + entry.FreshnessBoost*0.2)
	}
	if entry.Timestamp.IsZero() {
		return clamp01(1 + entry.FreshnessBoost*0.2)
	}

	age := now.UTC().Sub(entry.Timestamp.UTC())
	if age < 0 {
		age = 0
	}
	halfLife := config.TemporalDecayHalfLife
	if halfLife <= 0 {
		halfLife = defaultTemporalDecayHalfLife
	}
	floor := 0.0
	for _, anchor := range activeAnchors(entry.Anchors, now) {
		if anchor.Weight > 0 && anchor.Weight < config.AnchorMinWeight {
			continue
		}
		switch anchor.Type {
		case MemoryAnchorEmotion:
			if halfLife > defaultTemporalDecayHalfLife/2 {
				halfLife = defaultTemporalDecayHalfLife / 2
			}
		case MemoryAnchorPreference:
			if halfLife < defaultTemporalDecayHalfLife*4 {
				halfLife = defaultTemporalDecayHalfLife * 4
			}
			if floor < 0.15 {
				floor = 0.15
			}
		case MemoryAnchorIdentity:
			if halfLife < defaultTemporalDecayHalfLife*6 {
				halfLife = defaultTemporalDecayHalfLife * 6
			}
			if floor < 0.35 {
				floor = 0.35
			}
		case MemoryAnchorAvoidance, MemoryAnchorConstraint:
			if halfLife < defaultTemporalDecayHalfLife*12 {
				halfLife = defaultTemporalDecayHalfLife * 12
			}
			if floor < 0.85 {
				floor = 0.85
			}
		}
	}
	temporal := math.Exp2(-age.Hours() / halfLife.Hours())
	if temporal < floor {
		temporal = floor
	}
	return clamp01(temporal + entry.FreshnessBoost*0.2)
}

func anchorBoostScore(entry MemoryEntry, query MemoryQuery, now time.Time, config memoryScoringConfig) float64 {
	if !config.AnchorEnabled {
		return 0
	}
	anchors := activeAnchors(entry.Anchors, now)
	if len(anchors) == 0 {
		return 0
	}
	phrase := strings.ToLower(strings.TrimSpace(query.SemanticQuery))
	terms := queryTerms(query)
	anchorTypes := normalizedAnchorTypes(query.AnchorTypes)
	best := 0.0
	for _, anchor := range anchors {
		if anchor.Weight > 0 && anchor.Weight < config.AnchorMinWeight {
			continue
		}
		if len(anchorTypes) > 0 {
			if _, ok := anchorTypes[anchor.Type]; !ok {
				continue
			}
		}
		anchorText := searchableAnchorText(anchor)
		if anchorText == "" {
			continue
		}
		matched := 0
		for _, term := range terms {
			if strings.Contains(anchorText, term) {
				matched++
			}
		}
		if phrase != "" && strings.Contains(anchorText, phrase) {
			matched += 2
		}
		if matched == 0 && len(anchorTypes) == 0 {
			continue
		}
		termFactor := 0.6
		if len(terms) > 0 {
			termFactor = clamp01(float64(matched) / float64(len(terms)+1))
		}
		score := clamp01(anchor.Weight * (0.5 + termFactor*0.5))
		switch anchor.Type {
		case MemoryAnchorAvoidance, MemoryAnchorConstraint:
			score = clamp01(score + 0.1)
		case MemoryAnchorPreference, MemoryAnchorIdentity:
			score = clamp01(score + 0.05)
		}
		if score > best {
			best = score
		}
	}
	return best
}

func accessBoostScore(entry MemoryEntry) float64 {
	if entry.AccessCount <= 0 {
		return 0
	}
	return clamp01(math.Log1p(float64(entry.AccessCount)) / math.Log(10))
}

func memoryLayerBias(entry MemoryEntry) float64 {
	switch entryLayer(entry) {
	case "hot":
		return 0.035
	case "warm":
		return 0.025
	case "decision":
		return 0.01
	case "cold":
		return 0.012
	case "graph":
		return 0.008
	case "markdown":
		return 0.01
	default:
		return 0
	}
}

func searchableEntryText(entry MemoryEntry) string {
	parts := make([]string, 0, 4+len(entry.Anchors))
	if content := strings.TrimSpace(entry.Content); content != "" {
		parts = append(parts, strings.ToLower(content))
	}
	if summary := strings.TrimSpace(entry.Summary); summary != "" {
		parts = append(parts, strings.ToLower(summary))
	}
	if source := strings.TrimSpace(entry.Source); source != "" {
		parts = append(parts, strings.ToLower(source))
	}
	for _, anchor := range entry.Anchors {
		if text := searchableAnchorText(anchor); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

func searchableAnchorText(anchor MemoryAnchor) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(anchor.Type)),
		strings.ToLower(strings.TrimSpace(anchor.Key)),
		strings.ToLower(strings.TrimSpace(anchor.Value)),
		strings.ToLower(strings.TrimSpace(anchor.Reason)),
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func queryTerms(query MemoryQuery) []string {
	terms := make([]string, 0, len(query.Keywords)+8)
	terms = append(terms, query.Keywords...)
	if semantic := strings.TrimSpace(query.SemanticQuery); semantic != "" {
		terms = append(terms, strings.Fields(strings.ToLower(semantic))...)
	}
	seen := make(map[string]struct{}, len(terms))
	out := make([]string, 0, len(terms))
	for _, term := range terms {
		trimmed := strings.ToLower(strings.Trim(strings.TrimSpace(term), ",.!?:;()[]{}\"'"))
		if len(trimmed) < 2 {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func normalizedAnchorTypes(anchorTypes []string) map[string]struct{} {
	if len(anchorTypes) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(anchorTypes))
	for _, anchorType := range anchorTypes {
		trimmed := strings.ToLower(strings.TrimSpace(anchorType))
		if trimmed == "" {
			continue
		}
		out[trimmed] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func entryLayer(entry MemoryEntry) string {
	if entry.Metadata != nil {
		if layer, ok := entry.Metadata["layer"].(string); ok {
			return strings.ToLower(strings.TrimSpace(layer))
		}
	}
	return strings.ToLower(strings.TrimSpace(entry.Source))
}

func queryRequestsRecencyBoost(entry MemoryEntry) bool {
	return entry.FreshnessBoost > 0
}
