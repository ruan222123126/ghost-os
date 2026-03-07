package memory

import "strings"

func (g *GraphService) Rebuild(opts GraphRebuildOptions) (GraphRebuildStats, error) {
	stats := GraphRebuildStats{Namespace: normalizeGraphNamespace(firstNonEmpty(opts.Namespace, g.namespace))}
	if !g.Enabled() || g.cold == nil {
		return stats, nil
	}
	if !opts.DryRun {
		if err := g.store.ResetNamespace(stats.Namespace); err != nil {
			return stats, err
		}
	}

	archives, err := g.cold.ListArchives(opts.TimeRange)
	if err != nil {
		return stats, err
	}
	limit := opts.MaxSessions
	if limit <= 0 || limit > len(archives) {
		limit = len(archives)
	}
	for i := 0; i < limit; i++ {
		archive := archives[i]
		stats.SessionsScanned++
		facts := g.extractor.ExtractFromMessages(archive.Messages, GraphExtractOptions{
			Namespace: stats.Namespace,
			SessionID: strings.TrimSpace(archive.SessionID),
			SourceID:  "archive:" + strings.TrimSpace(archive.SessionID),
			Timestamp: archive.ArchivedAt,
		})
		stats.FactsExtracted += len(facts)
		if opts.DryRun || len(facts) == 0 {
			continue
		}
		applyStats, err := g.store.ApplyFacts(stats.Namespace, facts, g.resolver)
		if err != nil {
			return stats, err
		}
		stats.NodesUpserted += applyStats.NodesUpserted
		stats.EdgesUpserted += applyStats.EdgesUpserted
	}

	nodeIDs, err := g.cold.ListMarkdownNodes()
	if err != nil {
		return stats, err
	}
	for _, nodeID := range nodeIDs {
		node, err := g.cold.LoadMarkdownNode(nodeID)
		if err != nil {
			continue
		}
		ts := markdownNodeTimestamp(node)
		if opts.TimeRange != nil && !opts.TimeRange.Contains(ts) {
			continue
		}
		stats.MarkdownScanned++
		facts := g.extractor.ExtractFromMarkdownNode(node, stats.Namespace)
		stats.FactsExtracted += len(facts)
		if opts.DryRun || len(facts) == 0 {
			continue
		}
		applyStats, err := g.store.ApplyFacts(stats.Namespace, facts, g.resolver)
		if err != nil {
			return stats, err
		}
		stats.NodesUpserted += applyStats.NodesUpserted
		stats.EdgesUpserted += applyStats.EdgesUpserted
	}
	return stats, nil
}
