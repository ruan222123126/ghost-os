package memory

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	"ghost-os/bridge/llm"
	igraph "ghost-os/bridge/memory/internal/graph"
)

type GraphService struct {
	inner *igraph.GraphService
}

type graphColdAdapter struct {
	cold *ColdMemory
}

func NewGraphService(config MemoryConfig, cold *ColdMemory, summarizer Summarizer) *GraphService {
	inner := igraph.NewGraphService(igraph.MemoryConfig{
		GraphEnabled:          config.Graph.Enabled,
		GraphPath:             config.Graph.Path,
		GraphExtractOnArchive: config.Graph.ExtractOnArchive,
		GraphExtractOnEvolve:  config.Graph.ExtractOnEvolve,
		GraphMaxHops:          config.Graph.MaxHops,
		GraphMaxHits:          config.Graph.MaxHits,
		GraphMinConfidence:    config.Graph.MinConfidence,
		GraphNamespace:        config.Graph.Namespace,
		GraphDebugEnabled:     config.Graph.DebugEnabled,
		EvolutionUseWorker:    config.Warm.EvolutionUseWorker,
	}, graphColdAdapter{cold: cold}, graphSummarizerAdapter{summarizer: summarizer})
	return &GraphService{inner: inner}
}

type graphSummarizerAdapter struct{ summarizer Summarizer }

func (a graphSummarizerAdapter) Summarize(messages []llm.Message) (string, error) {
	if a.summarizer == nil {
		return "", nil
	}
	return a.summarizer.Summarize(messages)
}

func (a graphSummarizerAdapter) ExtractGraphFacts(messages []llm.Message) ([]igraph.GraphFact, error) {
	if extractor, ok := a.summarizer.(GraphFactExtractor); ok {
		return convertGraphSlice[GraphFact, igraph.GraphFact](extractor.ExtractGraphFacts(messages))
	}
	return nil, nil
}

func (a graphColdAdapter) ListArchives(timeRange *igraph.TimeRange) ([]igraph.ColdArchive, error) {
	if a.cold == nil {
		return nil, nil
	}
	var localRange *TimeRange
	if timeRange != nil {
		localRange = &TimeRange{Start: timeRange.Start, End: timeRange.End}
	}
	archives, err := a.cold.ListArchives(localRange)
	if err != nil {
		return nil, err
	}
	return convertGraphValue[[]ColdArchive, []igraph.ColdArchive](archives)
}

func (a graphColdAdapter) ListMarkdownNodes() ([]string, error) {
	if a.cold == nil {
		return nil, nil
	}
	return a.cold.ListMarkdownNodes()
}

func (a graphColdAdapter) LoadMarkdownNode(id string) (igraph.MarkdownNode, error) {
	if a.cold == nil {
		return igraph.MarkdownNode{}, nil
	}
	node, err := a.cold.LoadMarkdownNode(id)
	if err != nil {
		return igraph.MarkdownNode{}, err
	}
	return convertGraphValue[MarkdownNode, igraph.MarkdownNode](node)
}

func (g *GraphService) Enabled() bool {
	return g != nil && g.inner != nil && g.inner.Enabled()
}

func (g *GraphService) IngestArchiveMessages(sessionID string, messages []llm.Message) error {
	if g == nil || g.inner == nil || !g.Enabled() {
		return ErrGraphArchiveIngestDisabled
	}
	if err := g.inner.IngestArchiveMessages(strings.TrimSpace(sessionID), messages); err != nil {
		if errors.Is(err, igraph.ErrArchiveIngestDisabled) {
			return ErrGraphArchiveIngestDisabled
		}
		return err
	}
	return nil
}

func (g *GraphService) IngestMarkdownNode(node MarkdownNode) error {
	if g == nil || g.inner == nil {
		return nil
	}
	converted, err := convertGraphValue[MarkdownNode, igraph.MarkdownNode](node)
	if err != nil {
		return err
	}
	return g.inner.IngestMarkdownNode(converted)
}

func (g *GraphService) SyncObject(object MemoryObject) error {
	_ = object
	return ErrGraphObjectSyncDisabled
}

func (g *GraphService) Retrieve(query MemoryQuery, scope SessionScope) ([]MemoryEntry, []GraphHit, error) {
	if g == nil || g.inner == nil {
		return nil, nil, nil
	}
	entries, hits, err := g.inner.Retrieve(igraph.MemoryQuery{
		TimeRange:       convertGraphTimeRange(query.TimeRange),
		Limit:           query.Limit,
		Keywords:        append([]string(nil), query.Keywords...),
		IncludeGraph:    query.IncludeGraph,
		SemanticQuery:   query.SemanticQuery,
		GraphHops:       query.GraphHops,
		GraphPredicates: append([]string(nil), query.GraphPredicates...),
		GraphDebug:      query.GraphDebug,
	}, igraph.SessionScope{SessionID: scope.SessionID, History: scope.History})
	if err != nil {
		return nil, nil, err
	}
	outEntries, err := convertGraphValue[[]igraph.MemoryEntry, []MemoryEntry](entries)
	if err != nil {
		return nil, nil, err
	}
	outHits, err := convertGraphValue[[]igraph.GraphHit, []GraphHit](hits)
	if err != nil {
		return nil, nil, err
	}
	return outEntries, outHits, nil
}

func (g *GraphService) GraphStats(namespace string) GraphStats {
	if g == nil || g.inner == nil {
		return GraphStats{Namespace: normalizeGraphNamespace(namespace)}
	}
	stats, err := convertGraphValue[igraph.GraphStats, GraphStats](g.inner.GraphStats(namespace))
	if err != nil {
		log.Printf("[MEMORY] graph stats conversion failed: namespace=%s err=%v", strings.TrimSpace(namespace), err)
		return GraphStats{Namespace: normalizeGraphNamespace(namespace)}
	}
	return stats
}

func (g *GraphService) Rebuild(opts GraphRebuildOptions) (GraphRebuildStats, error) {
	if g == nil || g.inner == nil {
		return GraphRebuildStats{}, nil
	}
	stats, err := g.inner.Rebuild(igraph.GraphRebuildOptions{
		Namespace:   opts.Namespace,
		TimeRange:   convertGraphTimeRange(opts.TimeRange),
		MaxSessions: opts.MaxSessions,
		DryRun:      opts.DryRun,
	})
	if err != nil {
		return GraphRebuildStats{}, err
	}
	return convertGraphValue[igraph.GraphRebuildStats, GraphRebuildStats](stats)
}

func convertGraphTimeRange(value *TimeRange) *igraph.TimeRange {
	if value == nil {
		return nil
	}
	return &igraph.TimeRange{Start: value.Start, End: value.End}
}

func convertGraphValue[From any, To any](value From) (To, error) {
	var out To
	payload, err := json.Marshal(value)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return out, err
	}
	return out, nil
}

func convertGraphSlice[From any, To any](value []From, err error) ([]To, error) {
	if err != nil {
		return nil, err
	}
	return convertGraphValue[[]From, []To](value)
}
