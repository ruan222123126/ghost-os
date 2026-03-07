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
	return MemoryMetrics{
		L1Hits:               m.l1Hits.Load(),
		L2Hits:               m.l2Hits.Load(),
		L3Hits:               m.l3Hits.Load(),
		MarkdownHits:         m.markdownHits.Load(),
		GraphHits:            m.graphHits.Load(),
		EvolutionRuns:        m.evolutionRuns.Load(),
		NodesCreated:         m.nodesCreated.Load(),
		EntriesEvolved:       m.entriesEvolved.Load(),
		TruthEventsWritten:   m.truthEventsWritten.Load(),
		TruthObjectsUpserted: m.truthObjectsUpserted.Load(),
		TruthClaimsUpserted:  m.truthClaimsUpserted.Load(),
		TruthErrors:          m.truthErrors.Load(),
		TruthReplays:         m.truthReplays.Load(),
	}
}
