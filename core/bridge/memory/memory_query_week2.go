package memory

import (
	"log"
	"strings"
	"time"
)

func (s *QueryService) queryResultWeek2WithScope(query MemoryQuery, scope SessionScope) (MemoryQueryResult, error) {
	results := make([]MemoryEntry, 0, 32)
	seenIDs := make(map[string]struct{}, 32)
	seenFingerprints := make(map[string]struct{}, 32)
	now := time.Now().UTC()

	collect := func(entries []MemoryEntry) {
		for _, entry := range entries {
			if _, ok := seenIDs[entry.ID]; ok {
				continue
			}
			seenIDs[entry.ID] = struct{}{}
			fingerprint := normalizeRecallText(firstNonEmpty(entry.Summary, entry.Content))
			if fingerprint != "" {
				seenFingerprints[fingerprint] = struct{}{}
			}
			results = append(results, entry)
		}
	}
	collectDistinct := func(entries []MemoryEntry) {
		for _, entry := range entries {
			if _, ok := seenIDs[entry.ID]; ok {
				continue
			}
			fingerprint := normalizeRecallText(firstNonEmpty(entry.Summary, entry.Content))
			if fingerprint != "" {
				if _, ok := seenFingerprints[fingerprint]; ok {
					continue
				}
				seenFingerprints[fingerprint] = struct{}{}
			}
			seenIDs[entry.ID] = struct{}{}
			results = append(results, entry)
		}
	}

	hotEntries := queryHot(scope, query)
	if len(hotEntries) > 0 {
		s.metrics.l1Hits.Add(uint64(len(hotEntries)))
	}
	collect(hotEntries)

	warmQuery := query
	warmQuery.Limit = 0
	warmEntries, err := s.warm.RetrieveCandidates(warmQuery)
	if err != nil {
		return MemoryQueryResult{}, err
	}
	if len(warmEntries) > 0 {
		s.metrics.l2Hits.Add(uint64(len(warmEntries)))
	}
	collect(warmEntries)

	var decisionHits []DecisionHit
	if s.decision != nil && query.IncludeDecision {
		decisionEntries, hits, err := s.decision.Retrieve(query, scope)
		if err != nil {
			log.Printf("[MEMORY] decision recall failed, falling back to other layers: %v", err)
		} else {
			collectDistinct(decisionEntries)
			decisionHits = hits
		}
	}

	var graphHits []GraphHit
	if s.graph != nil && query.IncludeGraph {
		graphEntries, hits, err := s.graph.Retrieve(query, scope)
		if err != nil {
			log.Printf("[MEMORY] graph recall failed, falling back to text layers: %v", err)
		} else {
			if len(graphEntries) > 0 {
				s.metrics.graphHits.Add(uint64(len(graphEntries)))
			}
			collectDistinct(graphEntries)
			graphHits = hits
		}
	}

	coldQuery := query
	coldQuery.Limit = 0
	coldEntries, err := s.cold.Retrieve(coldQuery)
	if err != nil {
		return MemoryQueryResult{}, err
	}
	if len(coldEntries) > 0 {
		s.metrics.l3Hits.Add(uint64(len(coldEntries)))
	}
	collect(coldEntries)

	if query.IncludeMarkdown {
		markdownQuery := query
		markdownQuery.Limit = 0
		markdownEntries, err := s.queryMarkdown(markdownQuery)
		if err != nil {
			return MemoryQueryResult{}, err
		}
		if len(markdownEntries) > 0 {
			s.metrics.markdownHits.Add(uint64(len(markdownEntries)))
		}
		collect(markdownEntries)
	}

	results = rankMemoryEntries(results, query, now, s.scoring)
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	s.recordResultAccess(results, now)
	liveResult := MemoryQueryResult{Entries: results, GraphHits: graphHits, DecisionHits: decisionHits}
	intentPlan, vectorHits, shadowReport := s.runShadowRecall(query, scope, liveResult)
	liveResult.IntentPlan = intentPlan
	liveResult.VectorHits = vectorHits
	liveResult.ShadowReport = shadowReport
	return liveResult, nil
}

func (s *QueryService) recordResultAccess(entries []MemoryEntry, now time.Time) {
	warmHits := make([]string, 0, len(entries))
	decisionMemoHits := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entryLayer(entry) != "warm" {
			if entryLayer(entry) == "decision" {
				memoID, _ := entry.Metadata["memo_id"].(string)
				if strings.TrimSpace(memoID) != "" {
					decisionMemoHits = append(decisionMemoHits, strings.TrimSpace(memoID))
				}
			}
			continue
		}
		warmHits = append(warmHits, entry.ID)
	}
	s.warm.RecordAccess(warmHits, now)
	if s.decision != nil {
		s.decision.RecordMemoAccess(decisionMemoHits, now)
	}
}

func (s *QueryService) truthReadRequested(query MemoryQuery) bool {
	return s.truthReadEnabled || query.IncludeTruth || query.TruthDebug
}
