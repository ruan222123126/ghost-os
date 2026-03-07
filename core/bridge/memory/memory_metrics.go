package memory

import "sync/atomic"

type memoryCounters struct {
	l1Hits               atomic.Uint64
	l2Hits               atomic.Uint64
	l3Hits               atomic.Uint64
	markdownHits         atomic.Uint64
	graphHits            atomic.Uint64
	evolutionRuns        atomic.Uint64
	nodesCreated         atomic.Uint64
	entriesEvolved       atomic.Uint64
	plannerRuns          atomic.Uint64
	plannerErrors        atomic.Uint64
	vectorDocsIndexed    atomic.Uint64
	vectorShadowHits     atomic.Uint64
	shadowQueries        atomic.Uint64
	shadowOverlapMilli   atomic.Uint64
	shadowOnlyCandidates atomic.Uint64
	shadowLatencyMs      atomic.Uint64
	shadowWouldHelpCount atomic.Uint64
	truthEventsWritten   atomic.Uint64
	truthObjectsUpserted atomic.Uint64
	truthClaimsUpserted  atomic.Uint64
	truthErrors          atomic.Uint64
	truthReplays         atomic.Uint64
}

func (m *memoryCounters) snapshot() MemoryMetrics {
	if m == nil {
		return MemoryMetrics{}
	}
	shadowQueries := m.shadowQueries.Load()
	shadowOverlapRate := 0.0
	shadowLatencyMs := uint64(0)
	shadowWouldHelpRate := 0.0
	if shadowQueries > 0 {
		shadowOverlapRate = float64(m.shadowOverlapMilli.Load()) / float64(shadowQueries*1000)
		shadowLatencyMs = m.shadowLatencyMs.Load() / shadowQueries
		shadowWouldHelpRate = float64(m.shadowWouldHelpCount.Load()) / float64(shadowQueries)
	}
	return MemoryMetrics{
		L1Hits:               m.l1Hits.Load(),
		L2Hits:               m.l2Hits.Load(),
		L3Hits:               m.l3Hits.Load(),
		MarkdownHits:         m.markdownHits.Load(),
		GraphHits:            m.graphHits.Load(),
		EvolutionRuns:        m.evolutionRuns.Load(),
		NodesCreated:         m.nodesCreated.Load(),
		EntriesEvolved:       m.entriesEvolved.Load(),
		PlannerRuns:          m.plannerRuns.Load(),
		PlannerErrors:        m.plannerErrors.Load(),
		VectorDocsIndexed:    m.vectorDocsIndexed.Load(),
		VectorShadowHits:     m.vectorShadowHits.Load(),
		ShadowOverlapRate:    shadowOverlapRate,
		ShadowOnlyCandidates: m.shadowOnlyCandidates.Load(),
		ShadowLatencyMs:      shadowLatencyMs,
		ShadowWouldHelpRate:  shadowWouldHelpRate,
		TruthEventsWritten:   m.truthEventsWritten.Load(),
		TruthObjectsUpserted: m.truthObjectsUpserted.Load(),
		TruthClaimsUpserted:  m.truthClaimsUpserted.Load(),
		TruthErrors:          m.truthErrors.Load(),
		TruthReplays:         m.truthReplays.Load(),
	}
}
