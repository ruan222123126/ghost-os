package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/memory"
	"ghost-os/bridge/session"
)

const memoryWorkerTimeout = 30 * time.Second

func memoryManagerConfigFromStore(store *ConfigStore, sessionStore *session.Store) memory.MemoryConfig {
	summarizer := newMemoryWorkerSummarizer(store)
	if store != nil {
		if cfg, err := loadConfigWithRuntime(store.RuntimeConfig()); err == nil {
			return memoryManagerConfigFromAppConfig(cfg, sessionStore, summarizer)
		}
	}
	if cfg, err := LoadConfig(); err == nil {
		return memoryManagerConfigFromAppConfig(cfg, sessionStore, summarizer)
	}
	recipeReuseEnabled, recipeReuseEnabledSet := memoryDecisionRecipeReuseEnabledFromEnv()
	recipeExecutionTrackingEnabled, recipeExecutionTrackingEnabledSet := memoryDecisionRecipeExecutionTrackingEnabledFromEnv()
	recipeBackfillEnabled, recipeBackfillEnabledSet := memoryDecisionRecipeBackfillEnabledFromEnv()
	recipeDefaultEnabled, recipeDefaultEnabledSet := memoryDecisionRecipeDefaultEnabledFromEnv()
	return memory.MemoryConfig{
		WarmCapacity:                      agentWarmMemoryCapacity,
		WarmPath:                          memoryWarmPathFromEnv(),
		ColdBaseDir:                       memoryColdPathFromEnv(),
		AutoRecallEnabled:                 memoryAutoRecallEnabledFromEnv(),
		AutoRecallLimit:                   memoryAutoRecallLimitFromEnv(),
		WarmTTL:                           memoryWarmTTLFromEnv(),
		TemporalDecayEnabled:              memoryTemporalDecayEnabledFromEnv(),
		TemporalDecayHalfLife:             configuredTemporalHalfLife(memoryTemporalDecayEnabledFromEnv(), memoryTemporalDecayHalfLifeFromEnv()),
		AnchorEnabled:                     memoryAnchorEnabledFromEnv(),
		AnchorMinWeight:                   configuredAnchorMinWeight(memoryAnchorEnabledFromEnv(), memoryAnchorMinWeightFromEnv()),
		EvolutionInterval:                 memoryEvolutionIntervalFromEnv(),
		EvolutionEnabled:                  memoryEvolutionEnabledFromEnv(),
		EvolutionUseWorker:                memoryEvolutionUseWorkerFromEnv(),
		EvolutionBatchSize:                memoryEvolutionBatchSizeFromEnv(),
		GraphEnabled:                      memoryGraphEnabledFromEnv(),
		GraphPath:                         memoryGraphPathFromEnv(),
		GraphExtractOnArchive:             memoryGraphExtractOnArchiveFromEnv(),
		GraphExtractOnEvolve:              memoryGraphExtractOnEvolveFromEnv(),
		GraphMaxHops:                      memoryGraphMaxHopsFromEnv(),
		GraphMaxHits:                      memoryGraphMaxHitsFromEnv(),
		GraphMinConfidence:                memoryGraphMinConfidenceFromEnv(),
		GraphNamespace:                    memoryGraphNamespaceFromEnv(),
		GraphDebugEnabled:                 memoryGraphDebugEnabledFromEnv(),
		DecisionEnabled:                   memoryDecisionEnabledFromEnv(),
		DecisionCaptureOnTurn:             memoryDecisionCaptureOnTurnFromEnv(),
		DecisionCaptureOnTurnSet:          true,
		DecisionPath:                      memoryDecisionPathFromEnv(),
		DecisionMaxHits:                   memoryDecisionMaxHitsFromEnv(),
		DecisionMinConfidence:             memoryDecisionMinConfidenceFromEnv(),
		DecisionMinReuseScore:             memoryDecisionMinReuseScoreFromEnv(),
		DecisionRecipeEnabled:             memoryDecisionRecipeEnabledFromEnv(),
		DecisionRecipeInterval:            memoryDecisionRecipeIntervalFromEnv(),
		DecisionRecipeMinSupport:          memoryDecisionRecipeMinSupportFromEnv(),
		DecisionDebugEnabled:              memoryDecisionDebugEnabledFromEnv(),
		RecipeReuseEnabled:                recipeReuseEnabled,
		RecipeReuseEnabledSet:             recipeReuseEnabledSet,
		RecipeExecutionTrackingEnabled:    recipeExecutionTrackingEnabled,
		RecipeExecutionTrackingEnabledSet: recipeExecutionTrackingEnabledSet,
		RecipeBackfillEnabled:             recipeBackfillEnabled,
		RecipeBackfillEnabledSet:          recipeBackfillEnabledSet,
		RecipeDefaultEnabled:              recipeDefaultEnabled,
		RecipeDefaultEnabledSet:           recipeDefaultEnabledSet,
		RecipeDefaultGrayPercent:          memoryDecisionRecipeDefaultGrayPercentFromEnv(),
		RecipeMinSelectionConfidence:      memoryDecisionRecipeMinSelectionConfidenceFromEnv(),
		RecipeMinSuccessRate:              memoryDecisionRecipeMinSuccessRateFromEnv(),
		RecipeBackfillBatchSize:           memoryDecisionRecipeBackfillBatchSizeFromEnv(),
		RecipeBackfillInterval:            memoryDecisionRecipeBackfillIntervalFromEnv(),
		SessionStore:                      sessionStore,
		Summarizer:                        summarizer,
	}
}

func memoryManagerConfigFromAppConfig(cfg Config, sessionStore *session.Store, summarizer memory.Summarizer) memory.MemoryConfig {
	if !shouldEnableMemoryWorker(cfg) {
		summarizer = nil
	}
	return memory.MemoryConfig{
		WarmCapacity:                      agentWarmMemoryCapacity,
		WarmPath:                          cfg.MemoryWarmPath,
		ColdBaseDir:                       cfg.MemoryColdPath,
		AutoRecallEnabled:                 cfg.MemoryAutoRecallEnabled,
		AutoRecallLimit:                   cfg.MemoryAutoRecallLimit,
		WarmTTL:                           cfg.MemoryWarmTTL,
		TemporalDecayEnabled:              cfg.MemoryTemporalDecayEnabled,
		TemporalDecayHalfLife:             configuredTemporalHalfLife(cfg.MemoryTemporalDecayEnabled, cfg.MemoryTemporalDecayHalfLife),
		AnchorEnabled:                     cfg.MemoryAnchorEnabled,
		AnchorMinWeight:                   configuredAnchorMinWeight(cfg.MemoryAnchorEnabled, cfg.MemoryAnchorMinWeight),
		EvolutionInterval:                 cfg.MemoryEvolutionInterval,
		EvolutionEnabled:                  cfg.MemoryEvolutionEnabled,
		EvolutionUseWorker:                cfg.MemoryEvolutionUseWorker,
		EvolutionBatchSize:                cfg.MemoryEvolutionBatchSize,
		GraphEnabled:                      cfg.MemoryGraphEnabled,
		GraphPath:                         cfg.MemoryGraphPath,
		GraphExtractOnArchive:             cfg.MemoryGraphExtractOnArchive,
		GraphExtractOnEvolve:              cfg.MemoryGraphExtractOnEvolve,
		GraphMaxHops:                      cfg.MemoryGraphMaxHops,
		GraphMaxHits:                      cfg.MemoryGraphMaxHits,
		GraphMinConfidence:                cfg.MemoryGraphMinConfidence,
		GraphNamespace:                    cfg.MemoryGraphNamespace,
		GraphDebugEnabled:                 cfg.MemoryGraphDebugEnabled,
		DecisionEnabled:                   cfg.MemoryDecisionEnabled,
		DecisionCaptureOnTurn:             cfg.MemoryDecisionCaptureOnTurn,
		DecisionCaptureOnTurnSet:          true,
		DecisionPath:                      cfg.MemoryDecisionPath,
		DecisionMaxHits:                   cfg.MemoryDecisionMaxHits,
		DecisionMinConfidence:             cfg.MemoryDecisionMinConfidence,
		DecisionMinReuseScore:             cfg.MemoryDecisionMinReuseScore,
		DecisionRecipeEnabled:             cfg.MemoryDecisionRecipeEnabled,
		DecisionRecipeInterval:            cfg.MemoryDecisionRecipeInterval,
		DecisionRecipeMinSupport:          cfg.MemoryDecisionRecipeMinSupport,
		DecisionDebugEnabled:              cfg.MemoryDecisionDebugEnabled,
		RecipeReuseEnabled:                cfg.MemoryDecisionRecipeReuseEnabled,
		RecipeReuseEnabledSet:             cfg.MemoryDecisionRecipeReuseEnabledSet,
		RecipeExecutionTrackingEnabled:    cfg.MemoryDecisionRecipeExecutionTrackingEnabled,
		RecipeExecutionTrackingEnabledSet: cfg.MemoryDecisionRecipeExecutionTrackingEnabledSet,
		RecipeBackfillEnabled:             cfg.MemoryDecisionRecipeBackfillEnabled,
		RecipeBackfillEnabledSet:          cfg.MemoryDecisionRecipeBackfillEnabledSet,
		RecipeDefaultEnabled:              cfg.MemoryDecisionRecipeDefaultEnabled,
		RecipeDefaultEnabledSet:           cfg.MemoryDecisionRecipeDefaultEnabledSet,
		RecipeDefaultGrayPercent:          cfg.MemoryDecisionRecipeDefaultGrayPercent,
		RecipeMinSelectionConfidence:      cfg.MemoryDecisionRecipeMinSelectionConfidence,
		RecipeMinSuccessRate:              cfg.MemoryDecisionRecipeMinSuccessRate,
		RecipeBackfillBatchSize:           cfg.MemoryDecisionRecipeBackfillBatchSize,
		RecipeBackfillInterval:            cfg.MemoryDecisionRecipeBackfillInterval,
		SessionStore:                      sessionStore,
		Summarizer:                        summarizer,
	}
}

func shouldEnableMemoryWorker(cfg Config) bool {
	if cfg.MemoryEvolutionUseWorker {
		return true
	}
	return cfg.MemoryDecisionEnabled && cfg.MemoryDecisionCaptureOnTurn
}

func configuredTemporalHalfLife(enabled bool, halfLife time.Duration) time.Duration {
	if !enabled {
		return -1
	}
	return halfLife
}

func configuredAnchorMinWeight(enabled bool, minWeight float64) float64 {
	if !enabled {
		return -1
	}
	return minWeight
}

type memoryWorkerSummarizer struct {
	store *ConfigStore
}

func newMemoryWorkerSummarizer(store *ConfigStore) *memoryWorkerSummarizer {
	return &memoryWorkerSummarizer{store: store}
}

func (s *memoryWorkerSummarizer) Summarize(messages []llm.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}
	client, cfg, err := s.workerClient()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), memoryWorkerTimeout)
	defer cancel()
	resp, err := client.Complete(ctx, llm.CompletionRequest{Messages: []llm.Message{
		{Role: llm.RoleSystem, Text: "You are Ghost-OS memory worker. Summarize conversation fragments into a concise, behavior-relevant memory summary. Return only the summary text. Do not include chain-of-thought."},
		{Role: llm.RoleUser, Text: fmt.Sprintf("Worker model: %s\n\nConversation fragments:\n%s\n\nWrite a compact summary that preserves stable preferences, constraints, and important facts.", effectiveWorkerModel(cfg), renderMemoryWorkerMessages(messages))},
	}})
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(resp.Message.Text)
	if text == "" {
		return "", fmt.Errorf("memory summarizer returned empty content")
	}
	return text, nil
}

func (s *memoryWorkerSummarizer) ExtractAnchors(messages []llm.Message) ([]memory.MemoryAnchor, error) {
	if len(messages) == 0 {
		return nil, nil
	}
	client, cfg, err := s.workerClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), memoryWorkerTimeout)
	defer cancel()
	resp, err := client.Complete(ctx, llm.CompletionRequest{Messages: []llm.Message{
		{Role: llm.RoleSystem, Text: "You are Ghost-OS memory worker. Extract only stable user anchors. Return strict JSON array with objects: type,key,value,weight,reason. Allowed type values: preference,avoidance,constraint,identity,emotion. Do not include markdown or explanation."},
		{Role: llm.RoleUser, Text: fmt.Sprintf("Worker model: %s\n\nConversation fragments:\n%s\n\nReturn only JSON.", effectiveWorkerModel(cfg), renderMemoryWorkerMessages(messages))},
	}})
	if err != nil {
		return nil, err
	}
	text := stripJSONCodeFence(strings.TrimSpace(resp.Message.Text))
	if text == "" {
		return nil, nil
	}
	var anchors []memory.MemoryAnchor
	if err := json.Unmarshal([]byte(text), &anchors); err != nil {
		return nil, fmt.Errorf("decode memory anchors: %w", err)
	}
	return anchors, nil
}

func (s *memoryWorkerSummarizer) ExtractGraphFacts(messages []llm.Message) ([]memory.GraphFact, error) {
	if len(messages) == 0 {
		return nil, nil
	}
	client, cfg, err := s.workerClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), memoryWorkerTimeout)
	defer cancel()
	resp, err := client.Complete(ctx, llm.CompletionRequest{Messages: []llm.Message{
		{Role: llm.RoleSystem, Text: "You are Ghost-OS memory worker. Extract only stable graph facts. Return strict JSON array with objects: subject,subject_type,predicate,object,object_type,aliases,confidence,snippet. Allowed predicates: owner_of,member_of,uses,prefers,avoids,depends_on,related_to,blocked_by,works_on. Every fact must include a concise snippet copied from the conversation. Do not include markdown or explanation."},
		{Role: llm.RoleUser, Text: fmt.Sprintf("Worker model: %s\n\nConversation fragments:\n%s\n\nReturn only JSON.", effectiveWorkerModel(cfg), renderMemoryWorkerMessages(messages))},
	}})
	if err != nil {
		return nil, err
	}
	text := stripJSONCodeFence(strings.TrimSpace(resp.Message.Text))
	if text == "" {
		return nil, nil
	}
	var facts []memory.GraphFact
	if err := json.Unmarshal([]byte(text), &facts); err != nil {
		return nil, fmt.Errorf("decode graph facts: %w", err)
	}
	return facts, nil
}

func (s *memoryWorkerSummarizer) ExtractDecisionMemo(input memory.DecisionCaptureInput) (memory.DecisionMemo, error) {
	client, cfg, err := s.workerClient()
	if err != nil {
		return memory.DecisionMemo{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), memoryWorkerTimeout)
	defer cancel()
	resp, err := client.Complete(ctx, llm.CompletionRequest{Messages: []llm.Message{
		{Role: llm.RoleSystem, Text: "You are Ghost-OS decision memory worker. Return strict JSON only. Do not include markdown, commentary, or chain-of-thought. Fill only these optional fields when strongly supported: intent_key, intent_summary, problem_summary, context_summary, constraints, assumptions, strategy_summary, key_steps, avoid_patterns, validation_checks, needs_human_for, confidence, reuse_score. Leave all structural fields such as outcome, tools_used, questions_asked, human_blocked, environment, session identifiers, and timestamps empty."},
		{Role: llm.RoleUser, Text: renderDecisionWorkerPrompt(input, cfg)},
	}})
	if err != nil {
		return memory.DecisionMemo{}, err
	}
	text := stripJSONCodeFence(strings.TrimSpace(resp.Message.Text))
	if text == "" {
		return memory.DecisionMemo{}, fmt.Errorf("decision extractor returned empty content")
	}
	var memo memory.DecisionMemo
	if err := json.Unmarshal([]byte(text), &memo); err != nil {
		return memory.DecisionMemo{}, fmt.Errorf("decode decision memo: %w", err)
	}
	return sanitizeDecisionWorkerMemo(memo), nil
}

func (s *memoryWorkerSummarizer) workerClient() (*llm.Client, Config, error) {
	var cfg Config
	var err error
	if s.store != nil {
		cfg, err = loadConfigWithRuntime(s.store.RuntimeConfig())
	} else {
		cfg, err = LoadConfig()
	}
	if err != nil {
		return nil, Config{}, err
	}
	return llm.NewClientWithOptions(llm.ClientOptions{
		Provider:           cfg.Provider,
		BaseURL:            cfg.BaseURL,
		APIKey:             cfg.APIKey,
		Model:              effectiveWorkerModel(cfg),
		ChatPath:           cfg.ChatPath,
		Headers:            cfg.ProviderHeaders,
		AnthropicVersion:   cfg.AnthropicVersion,
		AnthropicMaxTokens: cfg.AnthropicMaxTokens,
	}), cfg, nil
}

func effectiveWorkerModel(cfg Config) string {
	workerModel := strings.TrimSpace(cfg.WorkerModel)
	if workerModel != "" {
		return workerModel
	}
	return cfg.Model
}

func renderMemoryWorkerMessages(messages []llm.Message) string {
	parts := make([]string, 0, len(messages))
	for _, msg := range messages {
		text := strings.TrimSpace(msg.Text)
		if text == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("- %s: %s", msg.Role, summarizeWorkerLine(text, 600)))
	}
	return strings.Join(parts, "\n")
}

func renderDecisionWorkerPrompt(input memory.DecisionCaptureInput, cfg Config) string {
	lines := []string{
		fmt.Sprintf("Worker model: %s", effectiveWorkerModel(cfg)),
		fmt.Sprintf("Outcome: %s", strings.TrimSpace(input.Outcome)),
		fmt.Sprintf("Session ended: %t", input.SessionEnded),
		fmt.Sprintf("User message: %s", summarizeWorkerLine(input.UserMessage, 220)),
		fmt.Sprintf("Environment: %s", renderDecisionEnvironment(input.Environment)),
	}
	if questions := renderDecisionAnsweredQuestions(input.AnsweredQuestions); questions != "" {
		lines = append(lines, "Answered questions:", questions)
	}
	if history := renderDecisionWorkerMessages(input.RecentHistory, 8); history != "" {
		lines = append(lines, "Recent history:", history)
	}
	if messages := renderDecisionWorkerMessages(input.NewMessages, 12); messages != "" {
		lines = append(lines, "New turn messages:", messages)
	}
	lines = append(lines, "Return only a single JSON object.")
	return strings.Join(lines, "\n\n")
}

func renderDecisionEnvironment(env memory.DecisionEnvFingerprint) string {
	parts := []string{
		"domain=" + summarizeWorkerLine(env.Domain, 40),
		"platform=" + summarizeWorkerLine(env.Platform, 40),
		"provider=" + summarizeWorkerLine(env.Provider, 40),
		"model=" + summarizeWorkerLine(env.Model, 60),
		"toolset=" + summarizeWorkerLine(env.ToolsetSignature, 120),
		"target=" + summarizeWorkerLine(env.TargetAppOrSite, 80),
	}
	return strings.Join(parts, ", ")
}

func renderDecisionAnsweredQuestions(questions []memory.DecisionAnsweredQuestion) string {
	if len(questions) == 0 {
		return ""
	}
	parts := make([]string, 0, len(questions))
	for _, item := range questions {
		prompt := summarizeWorkerLine(item.Prompt, 160)
		answer := summarizeWorkerLine(item.Answer, 160)
		if prompt == "" && answer == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("- prompt=%q answer=%q", prompt, answer))
	}
	return strings.Join(parts, "\n")
}

func renderDecisionWorkerMessages(messages []llm.Message, limit int) string {
	trimmed := decisionWorkerMessagesWindow(messages, limit)
	parts := make([]string, 0, len(trimmed))
	for _, msg := range trimmed {
		if line := renderDecisionWorkerMessage(msg); line != "" {
			parts = append(parts, line)
		}
	}
	return strings.Join(parts, "\n")
}

func decisionWorkerMessagesWindow(messages []llm.Message, limit int) []llm.Message {
	if len(messages) == 0 || limit <= 0 || len(messages) <= limit {
		return llm.CloneMessages(messages)
	}
	return llm.CloneMessages(messages[len(messages)-limit:])
}

func renderDecisionWorkerMessage(msg llm.Message) string {
	parts := make([]string, 0, 2)
	text := strings.TrimSpace(msg.Text)
	if text != "" {
		if msg.Role == llm.RoleTool {
			if envelope, ok := agent.ParseToolResultEnvelope(text); ok {
				text = firstNonEmpty(envelope.Error, envelope.Output)
			}
		}
		parts = append(parts, summarizeWorkerLine(text, 220))
	}
	if len(msg.ToolCalls) > 0 {
		calls := make([]string, 0, len(msg.ToolCalls))
		for _, call := range msg.ToolCalls {
			calls = append(calls, fmt.Sprintf("%s(%s)", call.Name, summarizeWorkerLine(string(call.Arguments), 120)))
		}
		parts = append(parts, strings.Join(calls, "; "))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("- %s: %s", msg.Role, strings.Join(parts, " | "))
}

func sanitizeDecisionWorkerMemo(memo memory.DecisionMemo) memory.DecisionMemo {
	return memory.DecisionMemo{
		IntentKey:        strings.TrimSpace(memo.IntentKey),
		IntentSummary:    summarizeWorkerLine(memo.IntentSummary, 220),
		ProblemSummary:   summarizeWorkerLine(memo.ProblemSummary, 220),
		ContextSummary:   summarizeWorkerLine(memo.ContextSummary, 220),
		Constraints:      summarizeWorkerLines(memo.Constraints, 140),
		Assumptions:      summarizeWorkerLines(memo.Assumptions, 140),
		StrategySummary:  summarizeWorkerLine(memo.StrategySummary, 220),
		KeySteps:         sanitizeDecisionWorkerSteps(memo.KeySteps),
		AvoidPatterns:    summarizeWorkerLines(memo.AvoidPatterns, 140),
		ValidationChecks: summarizeWorkerLines(memo.ValidationChecks, 140),
		NeedsHumanFor:    summarizeWorkerLines(memo.NeedsHumanFor, 140),
		Confidence:       memo.Confidence,
		ReuseScore:       memo.ReuseScore,
	}
}

func sanitizeDecisionWorkerSteps(steps []memory.DecisionStep) []memory.DecisionStep {
	if len(steps) == 0 {
		return nil
	}
	out := make([]memory.DecisionStep, 0, len(steps))
	for _, step := range steps {
		title := summarizeWorkerLine(step.Title, 80)
		summary := summarizeWorkerLine(step.Summary, 160)
		outcome := summarizeWorkerLine(step.Outcome, 40)
		if title == "" && summary == "" && outcome == "" {
			continue
		}
		out = append(out, memory.DecisionStep{Title: title, Summary: summary, Outcome: outcome})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func summarizeWorkerLines(values []string, maxLen int) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := summarizeWorkerLine(value, maxLen)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func summarizeWorkerLine(text string, maxLen int) string {
	trimmed := strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if len(trimmed) <= maxLen {
		return trimmed
	}
	if maxLen <= 3 {
		return trimmed[:maxLen]
	}
	return trimmed[:maxLen-3] + "..."
}

func stripJSONCodeFence(text string) string {
	trimmed := strings.TrimSpace(text)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	return strings.TrimSpace(trimmed)
}
