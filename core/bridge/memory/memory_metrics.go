package memory

import "sync/atomic"

type memoryCounters struct {
	l1Hits                    atomic.Uint64
	l2Hits                    atomic.Uint64
	l3Hits                    atomic.Uint64
	markdownHits              atomic.Uint64
	graphHits                 atomic.Uint64
	evolutionRuns             atomic.Uint64
	nodesCreated              atomic.Uint64
	entriesEvolved            atomic.Uint64
	plannerRuns               atomic.Uint64
	plannerErrors             atomic.Uint64
	truthQueries              atomic.Uint64
	truthHits                 atomic.Uint64
	truthVerifiedHits         atomic.Uint64
	truthConflictedHits       atomic.Uint64
	hybridRerankRuns          atomic.Uint64
	vectorPromotedHits        atomic.Uint64
	resultConfidenceSumMilli  atomic.Uint64
	resultConfidenceCount     atomic.Uint64
	lowConfidenceFiltered     atomic.Uint64
	contextInjectionFiltered  atomic.Uint64
	vectorDocsIndexed         atomic.Uint64
	vectorShadowHits          atomic.Uint64
	shadowQueries             atomic.Uint64
	shadowOverlapMilli        atomic.Uint64
	shadowOnlyCandidates      atomic.Uint64
	shadowLatencyMs           atomic.Uint64
	shadowWouldHelpCount      atomic.Uint64
	truthEventsWritten        atomic.Uint64
	ledgerEventsWritten       atomic.Uint64
	ledgerShadowMismatchTotal atomic.Uint64
	ledgerReplayQueries       atomic.Uint64
	ledgerReplayLatencyMs     atomic.Uint64
	ledgerBackfillProgress    atomic.Uint64
	truthObjectsUpserted      atomic.Uint64
	truthClaimsUpserted       atomic.Uint64
	truthErrors               atomic.Uint64
	truthReplays              atomic.Uint64
	recipeSelections          atomic.Uint64
	recipeApplied             atomic.Uint64
	recipeSuccess             atomic.Uint64
	recipePartial             atomic.Uint64
	recipeFailure             atomic.Uint64
	recipeHumanBlocked        atomic.Uint64
	recipeDeviations          atomic.Uint64
	recipeFallbacks           atomic.Uint64
	recipeBackfillScanned     atomic.Uint64
	recipeBackfillCreated     atomic.Uint64
	recipeBackfillUpdated     atomic.Uint64
	recipeDefaultGrayHits     atomic.Uint64
}

func (m *memoryCounters) snapshot() MemoryMetrics {
	if m == nil {
		return MemoryMetrics{}
	}
	shadowQueries := m.shadowQueries.Load()
	shadowOverlapRate := 0.0
	shadowLatencyMs := uint64(0)
	shadowWouldHelpRate := 0.0
	ledgerReplayLatencyMs := uint64(0)
	avgResultConfidence := 0.0
	recipeSuccessRate := 0.0
	recipePartialRate := 0.0
	recipeFailureRate := 0.0
	recipeHumanBlockedRate := 0.0
	recipeDeviationRate := 0.0
	recipeFallbackRate := 0.0
	recipeDefaultGrayHitRate := 0.0
	if shadowQueries > 0 {
		shadowOverlapRate = float64(m.shadowOverlapMilli.Load()) / float64(shadowQueries*1000)
		shadowLatencyMs = m.shadowLatencyMs.Load() / shadowQueries
		shadowWouldHelpRate = float64(m.shadowWouldHelpCount.Load()) / float64(shadowQueries)
	}
	if count := m.resultConfidenceCount.Load(); count > 0 {
		avgResultConfidence = float64(m.resultConfidenceSumMilli.Load()) / float64(count*1000)
	}
	if replayQueries := m.ledgerReplayQueries.Load(); replayQueries > 0 {
		ledgerReplayLatencyMs = m.ledgerReplayLatencyMs.Load() / replayQueries
	}
	recipeApplied := m.recipeApplied.Load()
	if recipeApplied > 0 {
		recipeSuccessRate = float64(m.recipeSuccess.Load()) / float64(recipeApplied)
		recipePartialRate = float64(m.recipePartial.Load()) / float64(recipeApplied)
		recipeFailureRate = float64(m.recipeFailure.Load()) / float64(recipeApplied)
		recipeHumanBlockedRate = float64(m.recipeHumanBlocked.Load()) / float64(recipeApplied)
		recipeDeviationRate = float64(m.recipeDeviations.Load()) / float64(recipeApplied)
	}
	recipeSelections := m.recipeSelections.Load()
	if denominator := recipeSelections + m.recipeFallbacks.Load(); denominator > 0 {
		recipeFallbackRate = float64(m.recipeFallbacks.Load()) / float64(denominator)
	}
	if recipeSelections > 0 {
		recipeDefaultGrayHitRate = float64(m.recipeDefaultGrayHits.Load()) / float64(recipeSelections)
	}
	return MemoryMetrics{
		L1Hits:                        m.l1Hits.Load(),
		L2Hits:                        m.l2Hits.Load(),
		L3Hits:                        m.l3Hits.Load(),
		MarkdownHits:                  m.markdownHits.Load(),
		GraphHits:                     m.graphHits.Load(),
		EvolutionRuns:                 m.evolutionRuns.Load(),
		NodesCreated:                  m.nodesCreated.Load(),
		EntriesEvolved:                m.entriesEvolved.Load(),
		PlannerRuns:                   m.plannerRuns.Load(),
		PlannerErrors:                 m.plannerErrors.Load(),
		TruthQueries:                  m.truthQueries.Load(),
		TruthHits:                     m.truthHits.Load(),
		TruthVerifiedHits:             m.truthVerifiedHits.Load(),
		TruthConflictedHits:           m.truthConflictedHits.Load(),
		HybridRerankRuns:              m.hybridRerankRuns.Load(),
		VectorPromotedHits:            m.vectorPromotedHits.Load(),
		AvgResultConfidence:           avgResultConfidence,
		LowConfidenceFiltered:         m.lowConfidenceFiltered.Load(),
		ContextInjectionFiltered:      m.contextInjectionFiltered.Load(),
		VectorDocsIndexed:             m.vectorDocsIndexed.Load(),
		VectorShadowHits:              m.vectorShadowHits.Load(),
		ShadowOverlapRate:             shadowOverlapRate,
		ShadowOnlyCandidates:          m.shadowOnlyCandidates.Load(),
		ShadowLatencyMs:               shadowLatencyMs,
		ShadowWouldHelpRate:           shadowWouldHelpRate,
		TruthEventsWritten:            m.truthEventsWritten.Load(),
		LedgerEventsWritten:           m.ledgerEventsWritten.Load(),
		LedgerShadowMismatchTotal:     m.ledgerShadowMismatchTotal.Load(),
		LedgerReplayLatencyMs:         ledgerReplayLatencyMs,
		LedgerBackfillProgress:        m.ledgerBackfillProgress.Load(),
		TruthObjectsUpserted:          m.truthObjectsUpserted.Load(),
		TruthClaimsUpserted:           m.truthClaimsUpserted.Load(),
		TruthErrors:                   m.truthErrors.Load(),
		TruthReplays:                  m.truthReplays.Load(),
		RecipeSelectedCount:           recipeSelections,
		RecipeAppliedCount:            recipeApplied,
		RecipeSuccessRate:             recipeSuccessRate,
		RecipePartialRate:             recipePartialRate,
		RecipeFailureRate:             recipeFailureRate,
		RecipeHumanBlockedRate:        recipeHumanBlockedRate,
		RecipeDeviationRate:           recipeDeviationRate,
		RecipeFallbackRate:            recipeFallbackRate,
		RecipeBackfillSessionsScanned: m.recipeBackfillScanned.Load(),
		RecipeBackfillCreated:         m.recipeBackfillCreated.Load(),
		RecipeBackfillUpdated:         m.recipeBackfillUpdated.Load(),
		RecipeDefaultGrayHitRate:      recipeDefaultGrayHitRate,
	}
}
