package memory

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	idecision "ghost-os/bridge/memory/internal/decision"
)

type DecisionService struct {
	inner *idecision.DecisionService
}

type decisionColdAdapter struct{ cold *ColdMemory }

type decisionTruthAdapter struct {
	writer *TruthWriter
	mapper *TruthMapper
}

type decisionCounterAdapter struct{ add func(uint64) }

func (c decisionCounterAdapter) Add(v uint64) {
	if c.add != nil {
		c.add(v)
	}
}

func NewDecisionService(config MemoryConfig, cold *ColdMemory) *DecisionService {
	var extractor idecision.DecisionMemoExtractor
	if worker, ok := config.Runtime.Summarizer.(DecisionMemoExtractor); ok {
		extractor = decisionExtractorAdapter{worker: worker}
	}
	inner := idecision.NewDecisionService(idecision.MemoryConfig{
		DecisionEnabled:                config.Decision.Enabled,
		DecisionCaptureOnTurn:          config.Decision.CaptureOnTurn,
		DecisionCaptureOnTurnSet:       config.Decision.CaptureOnTurnSet,
		DecisionPath:                   config.Decision.Path,
		DecisionMaxHits:                config.Decision.MaxHits,
		DecisionMinConfidence:          config.Decision.MinConfidence,
		DecisionMinReuseScore:          config.Decision.MinReuseScore,
		DecisionRecipeEnabled:          config.Decision.Recipe.Enabled,
		DecisionRecipeInterval:         config.Decision.Recipe.Interval,
		DecisionRecipeMinSupport:       config.Decision.Recipe.MinSupport,
		DecisionDebugEnabled:           config.Decision.DebugEnabled,
		RecipeReuseEnabled:             config.Decision.Recipe.ReuseEnabled,
		RecipeExecutionTrackingEnabled: config.Decision.Recipe.ExecutionTrackingEnabled,
		RecipeBackfillEnabled:          config.Decision.Recipe.BackfillEnabled,
		RecipeDefaultEnabled:           config.Decision.Recipe.DefaultEnabled,
		RecipeDefaultGrayPercent:       config.Decision.Recipe.DefaultGrayPercent,
		RecipeMinSelectionConfidence:   config.Decision.Recipe.MinSelectionConfidence,
		RecipeMinSuccessRate:           config.Decision.Recipe.MinSuccessRate,
		RecipeBackfillBatchSize:        config.Decision.Recipe.BackfillBatchSize,
		RecipeBackfillInterval:         config.Decision.Recipe.BackfillInterval,
		WorkerExtractor:                extractor,
	}, decisionColdAdapter{cold: cold})
	return &DecisionService{inner: inner}
}

type decisionExtractorAdapter struct{ worker DecisionMemoExtractor }

func (a decisionExtractorAdapter) ExtractDecisionMemo(input idecision.DecisionCaptureInput) (idecision.DecisionMemo, error) {
	if a.worker == nil {
		return idecision.DecisionMemo{}, nil
	}
	memo, err := a.worker.ExtractDecisionMemo(convertDecisionCaptureInputFromInternal(input))
	if err != nil {
		return idecision.DecisionMemo{}, err
	}
	return convertDecisionValue[DecisionMemo, idecision.DecisionMemo](memo)
}

func (a decisionColdAdapter) ListArchives(timeRange *idecision.TimeRange) ([]idecision.ColdArchive, error) {
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
	return convertDecisionValue[[]ColdArchive, []idecision.ColdArchive](archives)
}

func (a decisionColdAdapter) ListMarkdownNodes() ([]string, error) {
	if a.cold == nil {
		return nil, nil
	}
	return a.cold.ListMarkdownNodes()
}

func (a decisionColdAdapter) LoadMarkdownNode(id string) (idecision.MarkdownNode, error) {
	if a.cold == nil {
		return idecision.MarkdownNode{}, nil
	}
	node, err := a.cold.LoadMarkdownNode(id)
	if err != nil {
		return idecision.MarkdownNode{}, err
	}
	return convertDecisionValue[MarkdownNode, idecision.MarkdownNode](node)
}

func (a decisionTruthAdapter) WriteDecisionMemo(memo idecision.DecisionMemo, input idecision.DecisionCaptureInput) error {
	if a.writer == nil || a.mapper == nil || !a.writer.DualWriteEnabled() {
		return nil
	}
	localMemo, err := convertDecisionValue[idecision.DecisionMemo, DecisionMemo](memo)
	if err != nil {
		return err
	}
	localInput := convertDecisionCaptureInputFromInternal(input)
	object := a.mapper.MapDecisionMemo(localMemo, localInput)
	if err := writeTruthObjectShadow(a.writer, truthEventTypeDecisionMemo, object, localInput.TraceID); err != nil {
		return handleTruthShadowWriteError(a.writer, localInput.TraceID, object.ObjectID, err)
	}
	return nil
}

func newDecisionMetricsAdapter(metrics *memoryCounters) *idecision.DecisionMetrics {
	if metrics == nil {
		return nil
	}
	return idecision.NewDecisionMetrics(
		decisionCounterAdapter{add: func(v uint64) { metrics.recipeSelections.Add(v) }},
		decisionCounterAdapter{add: func(v uint64) { metrics.recipeApplied.Add(v) }},
		decisionCounterAdapter{add: func(v uint64) { metrics.recipeSuccess.Add(v) }},
		decisionCounterAdapter{add: func(v uint64) { metrics.recipePartial.Add(v) }},
		decisionCounterAdapter{add: func(v uint64) { metrics.recipeFailure.Add(v) }},
		decisionCounterAdapter{add: func(v uint64) { metrics.recipeHumanBlocked.Add(v) }},
		decisionCounterAdapter{add: func(v uint64) { metrics.recipeDeviations.Add(v) }},
		decisionCounterAdapter{add: func(v uint64) { metrics.recipeFallbacks.Add(v) }},
		decisionCounterAdapter{add: func(v uint64) { metrics.recipeBackfillScanned.Add(v) }},
		decisionCounterAdapter{add: func(v uint64) { metrics.recipeBackfillCreated.Add(v) }},
		decisionCounterAdapter{add: func(v uint64) { metrics.recipeBackfillUpdated.Add(v) }},
		decisionCounterAdapter{add: func(v uint64) { metrics.recipeDefaultGrayHits.Add(v) }},
	)
}

func (d *DecisionService) SetTruthShadow(writer *TruthWriter, mapper *TruthMapper) {
	if d == nil || d.inner == nil {
		return
	}
	d.inner.SetTruthShadow(decisionTruthAdapter{writer: writer, mapper: mapper})
}

func (d *DecisionService) SetMetrics(metrics *memoryCounters) {
	if d == nil || d.inner == nil {
		return
	}
	d.inner.SetMetrics(newDecisionMetricsAdapter(metrics))
}

func (d *DecisionService) Enabled() bool {
	return d != nil && d.inner != nil && d.inner.Enabled()
}

func (d *DecisionService) CaptureOnTurnEnabled() bool {
	return d != nil && d.inner != nil && d.inner.CaptureOnTurnEnabled()
}

func (d *DecisionService) MaxHits() int {
	if d == nil || d.inner == nil {
		return 0
	}
	return d.inner.MaxHits()
}

func (d *DecisionService) DebugEnabled() bool {
	return d != nil && d.inner != nil && d.inner.DebugEnabled()
}

func (d *DecisionService) Stats(namespace string) DecisionStats {
	if d == nil || d.inner == nil {
		return DecisionStats{Namespace: normalizeDecisionNamespace(namespace)}
	}
	stats, err := convertDecisionValue[idecision.DecisionStats, DecisionStats](d.inner.Stats(namespace))
	if err != nil {
		return DecisionStats{Namespace: normalizeDecisionNamespace(namespace)}
	}
	return stats
}

func (d *DecisionService) ListMemos(namespace string) []DecisionMemo {
	if d == nil || d.inner == nil {
		return nil
	}
	memos, err := convertDecisionValue[[]idecision.DecisionMemo, []DecisionMemo](d.inner.ListMemos(namespace))
	if err != nil {
		return nil
	}
	return memos
}

func (d *DecisionService) ListRecipes(namespace string) []DecisionRecipe {
	if d == nil || d.inner == nil {
		return nil
	}
	recipes, err := convertDecisionValue[[]idecision.DecisionRecipe, []DecisionRecipe](d.inner.ListRecipes(namespace))
	if err != nil {
		return nil
	}
	return recipes
}

func (d *DecisionService) RecordMemoAccess(ids []string, accessedAt time.Time) {
	if d != nil && d.inner != nil {
		d.inner.RecordMemoAccess(ids, accessedAt)
	}
}

func (d *DecisionService) CaptureTurn(input DecisionCaptureInput) error {
	if d == nil || d.inner == nil {
		return nil
	}
	return d.inner.CaptureTurn(convertDecisionCaptureInputToInternal(input))
}

func (d *DecisionService) BuildSelectorHint(query MemoryQuery, scope SessionScope) (string, []DecisionHit, error) {
	if d == nil || d.inner == nil {
		return "", nil, nil
	}
	hint, hits, err := d.inner.BuildSelectorHint(convertDecisionQuery(query), convertDecisionScope(scope))
	if err != nil {
		return "", nil, err
	}
	outHits, err := convertDecisionValue[[]idecision.DecisionHit, []DecisionHit](hits)
	if err != nil {
		return "", nil, err
	}
	return hint, outHits, nil
}

func (d *DecisionService) Retrieve(query MemoryQuery, scope SessionScope) ([]MemoryEntry, []DecisionHit, error) {
	if d == nil || d.inner == nil {
		return nil, nil, nil
	}
	entries, hits, err := d.inner.Retrieve(convertDecisionQuery(query), convertDecisionScope(scope))
	if err != nil {
		return nil, nil, err
	}
	outEntries, err := convertDecisionValue[[]idecision.MemoryEntry, []MemoryEntry](entries)
	if err != nil {
		return nil, nil, err
	}
	outHits, err := convertDecisionValue[[]idecision.DecisionHit, []DecisionHit](hits)
	if err != nil {
		return nil, nil, err
	}
	return outEntries, outHits, nil
}

func (d *DecisionService) Rebuild(opts DecisionRebuildOptions) (DecisionRebuildStats, error) {
	if d == nil || d.inner == nil {
		return DecisionRebuildStats{Namespace: normalizeDecisionNamespace(opts.Namespace)}, nil
	}
	stats, err := d.inner.Rebuild(convertDecisionRebuildOptions(opts))
	if err != nil {
		return DecisionRebuildStats{}, err
	}
	return convertDecisionValue[idecision.DecisionRebuildStats, DecisionRebuildStats](stats)
}

func (d *DecisionService) DistillAll(namespace string) (DecisionDistillStats, error) {
	if d == nil || d.inner == nil {
		return DecisionDistillStats{Namespace: distillStatsNamespace(namespace)}, nil
	}
	stats, err := d.inner.DistillAll(namespace)
	if err != nil {
		return DecisionDistillStats{}, err
	}
	return convertDecisionValue[idecision.DecisionDistillStats, DecisionDistillStats](stats)
}

func (d *DecisionService) StartDistiller() {
	if d != nil && d.inner != nil {
		d.inner.StartDistiller()
	}
}
func (d *DecisionService) StopDistiller() {
	if d != nil && d.inner != nil {
		d.inner.StopDistiller()
	}
}
func (d *DecisionService) HasDistiller() bool {
	return d != nil && d.inner != nil && d.inner.HasDistiller()
}

func (d *DecisionService) SelectRecipeReuse(query MemoryQuery, scope SessionScope, hits []DecisionHit, entries []MemoryEntry, plan *QueryIntentPlan, rerankReport *HybridRerankReport, now time.Time) (*DecisionRecipe, *RecipeSelectionReport, *RecipeAdvisory) {
	if d == nil || d.inner == nil {
		return nil, nil, nil
	}
	convertedHits, err := convertDecisionValue[[]DecisionHit, []idecision.DecisionHit](hits)
	if err != nil {
		return nil, nil, nil
	}
	convertedEntries, err := convertDecisionValue[[]MemoryEntry, []idecision.MemoryEntry](entries)
	if err != nil {
		return nil, nil, nil
	}
	var convertedPlan *idecision.QueryIntentPlan
	if plan != nil {
		value, err := convertDecisionValue[QueryIntentPlan, idecision.QueryIntentPlan](*plan)
		if err != nil {
			return nil, nil, nil
		}
		convertedPlan = &value
	}
	var convertedReport *idecision.HybridRerankReport
	if rerankReport != nil {
		value, err := convertDecisionValue[HybridRerankReport, idecision.HybridRerankReport](*rerankReport)
		if err != nil {
			return nil, nil, nil
		}
		convertedReport = &value
	}
	recipe, report, advisory := d.inner.SelectRecipeReuse(convertDecisionQuery(query), convertDecisionScope(scope), convertedHits, convertedEntries, convertedPlan, convertedReport, now)
	var outRecipe *DecisionRecipe
	var outReport *RecipeSelectionReport
	var outAdvisory *RecipeAdvisory
	if recipe != nil {
		value, err := convertDecisionValue[idecision.DecisionRecipe, DecisionRecipe](*recipe)
		if err == nil {
			outRecipe = &value
		}
	}
	if report != nil {
		value, err := convertDecisionValue[idecision.RecipeSelectionReport, RecipeSelectionReport](*report)
		if err == nil {
			outReport = &value
		}
	}
	if advisory != nil {
		value, err := convertDecisionValue[idecision.RecipeAdvisory, RecipeAdvisory](*advisory)
		if err == nil {
			outAdvisory = &value
		}
	}
	return outRecipe, outReport, outAdvisory
}

func (d *DecisionService) debugUpsertMemo(memo DecisionMemo) (bool, error) {
	if d == nil || d.inner == nil {
		return false, nil
	}
	converted, err := convertDecisionValue[DecisionMemo, idecision.DecisionMemo](memo)
	if err != nil {
		return false, err
	}
	return d.inner.UpsertMemo(converted)
}

func (d *DecisionService) debugUpsertRecipe(recipe DecisionRecipe) (bool, error) {
	if d == nil || d.inner == nil {
		return false, nil
	}
	converted, err := convertDecisionValue[DecisionRecipe, idecision.DecisionRecipe](recipe)
	if err != nil {
		return false, err
	}
	return d.inner.UpsertRecipe(converted)
}

func (d *DecisionService) debugUpsertRecipeRun(run RecipeRun) (bool, error) {
	if d == nil || d.inner == nil {
		return false, nil
	}
	converted, err := convertDecisionValue[RecipeRun, idecision.RecipeRun](run)
	if err != nil {
		return false, err
	}
	return d.inner.UpsertRecipeRun(converted)
}

func (d *DecisionService) debugListRecipeRuns(namespace string) []RecipeRun {
	if d == nil || d.inner == nil {
		return nil
	}
	runs, err := convertDecisionValue[[]idecision.RecipeRun, []RecipeRun](d.inner.ListRecipeRuns(namespace))
	if err != nil {
		return nil
	}
	return runs
}

func (d *DecisionService) debugRecipe(id string) (DecisionRecipe, bool) {
	if d == nil || d.inner == nil {
		return DecisionRecipe{}, false
	}
	recipe, ok := d.inner.Recipe(id)
	if !ok {
		return DecisionRecipe{}, false
	}
	value, err := convertDecisionValue[idecision.DecisionRecipe, DecisionRecipe](recipe)
	if err != nil {
		return DecisionRecipe{}, false
	}
	return value, true
}

func (d *DecisionService) debugExplainMemoLineage(id string) (MemoLineageExplanation, bool) {
	if d == nil || d.inner == nil {
		return MemoLineageExplanation{}, false
	}
	value, ok := d.inner.ExplainMemoLineage(id)
	if !ok {
		return MemoLineageExplanation{}, false
	}
	converted, err := convertDecisionValue[idecision.MemoLineageExplanation, MemoLineageExplanation](value)
	if err != nil {
		return MemoLineageExplanation{}, false
	}
	return converted, true
}

func (d *DecisionService) debugExplainRecipeLineage(id string) (RecipeLineageExplanation, bool) {
	if d == nil || d.inner == nil {
		return RecipeLineageExplanation{}, false
	}
	value, ok := d.inner.ExplainRecipeLineage(id)
	if !ok {
		return RecipeLineageExplanation{}, false
	}
	converted, err := convertDecisionValue[idecision.RecipeLineageExplanation, RecipeLineageExplanation](value)
	if err != nil {
		return RecipeLineageExplanation{}, false
	}
	return converted, true
}

func (d *DecisionService) debugExplainRecipeRunLineage(id string) (RecipeRunLineageExplanation, bool) {
	if d == nil || d.inner == nil {
		return RecipeRunLineageExplanation{}, false
	}
	value, ok := d.inner.ExplainRecipeRunLineage(id)
	if !ok {
		return RecipeRunLineageExplanation{}, false
	}
	converted, err := convertDecisionValue[idecision.RecipeRunLineageExplanation, RecipeRunLineageExplanation](value)
	if err != nil {
		return RecipeRunLineageExplanation{}, false
	}
	return converted, true
}

func (d *DecisionService) debugDistillerStopCh() <-chan struct{} {
	if d == nil || d.inner == nil {
		return nil
	}
	return d.inner.DistillerStopCh()
}

func convertDecisionQuery(query MemoryQuery) idecision.MemoryQuery {
	metadata := cloneMetadata(query.Metadata)
	if metadata == nil {
		metadata = make(map[string]any, 2)
	}
	if strings.TrimSpace(query.Namespace) != "" {
		metadata["namespace"] = strings.TrimSpace(query.Namespace)
	}
	if strings.TrimSpace(query.WorkspaceID) != "" {
		metadata["workspace_id"] = strings.TrimSpace(query.WorkspaceID)
	}
	return idecision.MemoryQuery{
		TimeRange:         convertDecisionTimeRange(query.TimeRange),
		Limit:             query.Limit,
		Keywords:          append([]string(nil), query.Keywords...),
		Metadata:          metadata,
		SemanticQuery:     query.SemanticQuery,
		MinConfidence:     query.MinConfidence,
		IncludeDecision:   query.IncludeDecision,
		DecisionDebug:     query.DecisionDebug,
		DecisionReuseOnly: query.DecisionReuseOnly,
		DecisionTypes:     append([]string(nil), query.DecisionTypes...),
		EnvironmentStrict: query.EnvironmentStrict,
		MinReuseScore:     query.MinReuseScore,
		Environment:       convertDecisionEnvPtr(query.Environment),
	}
}

func convertDecisionScope(scope SessionScope) idecision.SessionScope {
	return idecision.SessionScope{SessionID: scope.SessionID, Environment: convertDecisionEnvPtr(scope.Environment)}
}

func convertDecisionRebuildOptions(opts DecisionRebuildOptions) idecision.DecisionRebuildOptions {
	return idecision.DecisionRebuildOptions{
		Namespace:        opts.Namespace,
		TimeRange:        convertDecisionTimeRange(opts.TimeRange),
		MaxSessions:      opts.MaxSessions,
		BatchSize:        opts.BatchSize,
		CursorCheckpoint: opts.CursorCheckpoint,
		DryRun:           opts.DryRun,
		IncludeRecipes:   opts.IncludeRecipes,
		RebuildMemos:     opts.RebuildMemos,
		ResetNamespace:   opts.ResetNamespace,
	}
}

func convertDecisionTimeRange(value *TimeRange) *idecision.TimeRange {
	if value == nil {
		return nil
	}
	return &idecision.TimeRange{Start: value.Start, End: value.End}
}

func convertDecisionEnvPtr(value *DecisionEnvFingerprint) *idecision.DecisionEnvFingerprint {
	if value == nil {
		return nil
	}
	converted, err := convertDecisionValue[DecisionEnvFingerprint, idecision.DecisionEnvFingerprint](*value)
	if err != nil {
		return nil
	}
	return &converted
}

func cloneMetadata(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func convertDecisionValue[From any, To any](value From) (To, error) {
	var out To
	payload, err := json.Marshal(value)
	if err != nil {
		log.Printf("[MEMORY] decision conversion marshal failed: from=%T err=%v", value, err)
		return out, err
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		log.Printf("[MEMORY] decision conversion unmarshal failed: from=%T to=%T err=%v", value, out, err)
		return out, err
	}
	return out, nil
}

func convertDecisionCaptureInputToInternal(input DecisionCaptureInput) idecision.DecisionCaptureInput {
	converted, err := convertDecisionValue[DecisionCaptureInput, idecision.DecisionCaptureInput](input)
	if err != nil {
		return idecision.DecisionCaptureInput{}
	}
	converted.RecentHistory = llm.CloneMessages(input.RecentHistory)
	converted.NewMessages = llm.CloneMessages(input.NewMessages)
	return converted
}

func convertDecisionCaptureInputFromInternal(input idecision.DecisionCaptureInput) DecisionCaptureInput {
	converted, err := convertDecisionValue[idecision.DecisionCaptureInput, DecisionCaptureInput](input)
	if err != nil {
		return DecisionCaptureInput{}
	}
	converted.RecentHistory = llm.CloneMessages(input.RecentHistory)
	converted.NewMessages = llm.CloneMessages(input.NewMessages)
	return converted
}
