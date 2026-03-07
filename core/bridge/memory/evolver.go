package memory

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"ghost-os/bridge/llm"
)

// Evolver 收口 warm -> markdown 的后台演化任务与摘要策略。
type Evolver struct {
	warm *WarmMemory
	cold *ColdMemory

	summarizer         Summarizer
	warmTTL            time.Duration
	anchorEnabled      bool
	anchorMinWeight    float64
	evolutionEnabled   bool
	evolutionInterval  time.Duration
	evolutionUseWorker bool
	evolutionBatchSize int
	metrics            *memoryCounters

	dreamStopOnce sync.Once
	dreamStop     chan struct{}
	dreamOnce     sync.Once
	dreamWG       sync.WaitGroup
}

func NewEvolver(config MemoryConfig, warm *WarmMemory, cold *ColdMemory, summarizer Summarizer, metrics *memoryCounters) *Evolver {
	return &Evolver{
		warm:               warm,
		cold:               cold,
		summarizer:         summarizer,
		warmTTL:            config.WarmTTL,
		anchorEnabled:      config.AnchorEnabled,
		anchorMinWeight:    config.AnchorMinWeight,
		evolutionEnabled:   config.EvolutionEnabled,
		evolutionInterval:  config.EvolutionInterval,
		evolutionUseWorker: config.EvolutionUseWorker,
		evolutionBatchSize: config.EvolutionBatchSize,
		metrics:            metrics,
		dreamStop:          make(chan struct{}),
	}
}

func (e *Evolver) Evolve() (EvolutionStats, error) {
	stats := EvolutionStats{}
	e.metrics.evolutionRuns.Add(1)

	entries := e.warm.Snapshot()
	stats.ScannedEntries = len(entries)
	log.Printf("[MEMORY] Evolution started: scanning L2 (%d entries)", stats.ScannedEntries)
	if len(entries) == 0 {
		log.Printf("[MEMORY] Evolution complete: processed %d entries, created %d nodes", stats.ProcessedEntries, stats.NodesCreated)
		return stats, nil
	}

	now := time.Now().UTC()
	minAge := e.warmTTL / 4
	if minAge < 30*time.Minute {
		minAge = 30 * time.Minute
	}

	groups := make(map[string][]MemoryEntry, 8)
	for _, entry := range entries {
		age := now.Sub(entry.Timestamp)
		if age < minAge {
			continue
		}
		if entry.Importance >= 0.7 {
			continue
		}
		groupKey := evolutionGroupKey(entry)
		groups[groupKey] = append(groups[groupKey], entry)
	}

	groupKeys := make([]string, 0, len(groups))
	for key := range groups {
		groupKeys = append(groupKeys, key)
	}
	sort.SliceStable(groupKeys, func(i, j int) bool {
		return latestEntryTimestamp(groups[groupKeys[i]]).After(latestEntryTimestamp(groups[groupKeys[j]]))
	})
	remaining := e.evolutionBatchSize
	if remaining <= 0 {
		remaining = defaultEvolutionBatchSize
	}

	for _, groupKey := range groupKeys {
		if remaining <= 0 {
			break
		}
		group := groups[groupKey]
		if len(group) == 0 {
			continue
		}
		if len(group) > remaining {
			group = cloneEntries(group[:remaining])
		}

		relatedIDs := make([]string, 0, len(group))
		tags := make(map[string]struct{}, 8)
		sessionID := ""
		messages := evolutionMessages(group)
		for _, entry := range group {
			relatedIDs = append(relatedIDs, entry.ID)
			for _, tag := range extractKeywords(entry.Content) {
				tags[tag] = struct{}{}
			}
			if sid, ok := entry.Metadata["session_id"].(string); ok && strings.TrimSpace(sid) != "" {
				sessionID = strings.TrimSpace(sid)
			}
		}

		summary := e.summarizeGroup(messages, group)
		anchors := e.extractGroupAnchors(messages, sessionID, now)
		confidence := evolutionConfidence(group, anchors, summary)
		sourceIDs := append([]string(nil), relatedIDs...)
		node := MarkdownNode{
			ID:         fmt.Sprintf("node_%d", time.Now().UTC().UnixNano()),
			Importance: maxFloat(averageImportance(group), strongestAnchorWeight(anchors)),
			CreatedAt:  now,
			RelatedTo:  append([]string(nil), sourceIDs...),
			Tags:       sortedTagList(tags),
			SessionID:  sessionID,
			Summary:    summary,
			Anchors:    cloneAnchors(anchors),
			SourceIDs:  sourceIDs,
			Confidence: confidence,
			LastSeenAt: latestEntryTimestamp(group),
			Content:    buildStructuredMarkdownContent(groupKey, summary, anchors, group),
		}

		if err := e.cold.SaveMarkdownNode(node); err != nil {
			return stats, err
		}

		for _, entry := range group {
			if shouldRetainWarmEvidence(entry, anchors, e.anchorMinWeight) {
				enriched := cloneEntry(entry)
				enriched.Summary = summary
				enriched.Anchors = mergeAnchorsForEntry(entry, anchors)
				enriched.Confidence = maxFloat(enriched.Confidence, confidence)
				enriched.Source = "dreaming"
				_ = e.warm.Store(enriched)
				continue
			}
			_ = e.warm.Delete(entry.ID)
		}

		stats.ProcessedEntries += len(group)
		remaining -= len(group)
		stats.NodesCreated++
		e.metrics.nodesCreated.Add(1)
		e.metrics.entriesEvolved.Add(uint64(len(group)))
		log.Printf("[MEMORY] Created markdown node: %s (importance=%.2f, related=%d, anchors=%d)", node.ID, node.Importance, len(node.RelatedTo), len(node.Anchors))
	}

	log.Printf("[MEMORY] Evolution complete: processed %d entries, created %d nodes", stats.ProcessedEntries, stats.NodesCreated)
	return stats, nil
}

func (e *Evolver) StartDreaming() {
	if !e.evolutionEnabled {
		return
	}
	interval := e.evolutionInterval
	if interval <= 0 {
		interval = defaultEvolutionInterval
	}

	e.dreamOnce.Do(func() {
		e.dreamWG.Add(1)
		go func() {
			defer e.dreamWG.Done()
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-e.dreamStop:
					return
				case <-ticker.C:
					if _, err := e.Evolve(); err != nil {
						log.Printf("[MEMORY] Evolution error: %v", err)
					}
				}
			}
		}()
	})
}

func (e *Evolver) StopDreaming() {
	if e == nil {
		return
	}
	e.dreamStopOnce.Do(func() {
		close(e.dreamStop)
	})
	e.dreamWG.Wait()
}

func (e *Evolver) summarizeGroup(messages []llm.Message, entries []MemoryEntry) string {
	if e.evolutionUseWorker && e.summarizer != nil {
		summary, err := e.summarizer.Summarize(messages)
		if err == nil && strings.TrimSpace(summary) != "" {
			return strings.TrimSpace(summary)
		}
	}

	lines := make([]string, 0, minInt(len(entries), 5)+1)
	lines = append(lines, "# Memory Evolution Summary")
	for i, entry := range entries {
		if i >= 5 {
			break
		}
		lines = append(lines, "- "+summarizeLine(entry.Content, 180))
	}
	return strings.Join(lines, "\n")
}

func (e *Evolver) extractGroupAnchors(messages []llm.Message, sessionID string, detectedAt time.Time) []MemoryAnchor {
	if !e.anchorEnabled {
		return nil
	}
	var worker AnchorExtractor
	if extractor, ok := e.summarizer.(AnchorExtractor); ok {
		worker = extractor
	}
	return extractAnchors(messages, sessionID, detectedAt, e.anchorMinWeight, worker, e.evolutionUseWorker)
}

func evolutionGroupKey(entry MemoryEntry) string {
	if sid, ok := entry.Metadata["session_id"].(string); ok && strings.TrimSpace(sid) != "" {
		return "session:" + strings.TrimSpace(sid)
	}
	keywords := extractKeywords(entry.Content)
	if len(keywords) > 0 {
		return "topic:" + keywords[0]
	}
	return "fallback"
}

func averageImportance(entries []MemoryEntry) float64 {
	if len(entries) == 0 {
		return 0
	}
	sum := 0.0
	for _, entry := range entries {
		importance := entry.Importance
		if importance <= 0 {
			importance = calculateImportance(entry)
		}
		sum += importance
	}
	return clamp01(sum / float64(len(entries)))
}

func sortedTagList(tagSet map[string]struct{}) []string {
	if len(tagSet) == 0 {
		return nil
	}
	out := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		out = append(out, tag)
	}
	sort.Strings(out)
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

func minInt(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func evolutionMessages(entries []MemoryEntry) []llm.Message {
	if len(entries) == 0 {
		return nil
	}
	messages := make([]llm.Message, 0, len(entries))
	for _, entry := range entries {
		role := llm.RoleUser
		if entry.Metadata != nil {
			if rawRole, ok := entry.Metadata["role"].(string); ok {
				switch strings.ToLower(strings.TrimSpace(rawRole)) {
				case string(llm.RoleAssistant):
					role = llm.RoleAssistant
				case string(llm.RoleTool):
					role = llm.RoleTool
				case string(llm.RoleSystem):
					role = llm.RoleSystem
				}
			}
		}
		messages = append(messages, llm.Message{Role: role, Text: summarizeLine(entry.Content, 500)})
	}
	return messages
}

func latestEntryTimestamp(entries []MemoryEntry) time.Time {
	latest := time.Time{}
	for _, entry := range entries {
		if entry.Timestamp.After(latest) {
			latest = entry.Timestamp
		}
	}
	return latest.UTC()
}

func strongestAnchorWeight(anchors []MemoryAnchor) float64 {
	strongest := 0.0
	for _, anchor := range anchors {
		if anchor.Weight > strongest {
			strongest = anchor.Weight
		}
	}
	return clamp01(strongest)
}

func evolutionConfidence(entries []MemoryEntry, anchors []MemoryAnchor, summary string) float64 {
	confidence := 0.55
	if strings.TrimSpace(summary) != "" {
		confidence += 0.1
	}
	for _, entry := range entries {
		if entry.Confidence > confidence {
			confidence = entry.Confidence
		}
	}
	if strongest := strongestAnchorWeight(anchors); strongest > confidence {
		confidence = strongest
	}
	if confidence < 0.7 {
		confidence = 0.7
	}
	return clamp01(confidence)
}

func buildStructuredMarkdownContent(groupKey string, summary string, anchors []MemoryAnchor, entries []MemoryEntry) string {
	lines := []string{"# Memory Node", "", "## Summary", strings.TrimSpace(summary)}
	if strings.TrimSpace(groupKey) != "" {
		lines = append(lines, "", "- Group: `"+strings.TrimSpace(groupKey)+"`")
	}
	if len(anchors) > 0 {
		lines = append(lines, "", "## Anchors")
		for _, anchor := range anchors {
			line := fmt.Sprintf("- `%s:%s` %s (weight=%.2f)", anchor.Type, anchor.Key, anchor.Value, anchor.Weight)
			if strings.TrimSpace(anchor.Reason) != "" {
				line += " — " + summarizeLine(anchor.Reason, 120)
			}
			lines = append(lines, line)
		}
	}
	lines = append(lines, "", "## Evidence")
	for i, entry := range entries {
		if i >= 3 {
			break
		}
		lines = append(lines, fmt.Sprintf("- `%s`: %s", entry.ID, summarizeLine(entry.Content, 180)))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func shouldRetainWarmEvidence(entry MemoryEntry, anchors []MemoryAnchor, minWeight float64) bool {
	if entry.Importance >= 0.55 || entry.AccessCount >= 2 {
		return true
	}
	for _, anchor := range filterAnchorsByWeight(anchors, minWeight) {
		if !persistentAnchorType(anchor.Type) {
			continue
		}
		text := searchableEntryText(entry)
		if strings.Contains(text, strings.ToLower(anchor.Value)) || strings.Contains(text, strings.ToLower(anchor.Key)) {
			return true
		}
	}
	return false
}

func mergeAnchorsForEntry(entry MemoryEntry, anchors []MemoryAnchor) []MemoryAnchor {
	if len(anchors) == 0 {
		return cloneAnchors(entry.Anchors)
	}
	text := searchableEntryText(entry)
	merged := cloneAnchors(entry.Anchors)
	for _, anchor := range anchors {
		if strings.Contains(text, strings.ToLower(anchor.Value)) || strings.Contains(text, strings.ToLower(anchor.Key)) {
			merged = append(merged, anchor)
		}
	}
	if len(merged) == 0 && len(entry.Anchors) == 0 && len(anchors) > 0 {
		merged = append(merged, anchors[0])
	}
	return normalizeAnchors(merged)
}

func maxFloat(a, b float64) float64 {
	if a >= b {
		return a
	}
	return b
}
