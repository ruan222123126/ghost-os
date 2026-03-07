package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	return memory.MemoryConfig{
		WarmCapacity:          agentWarmMemoryCapacity,
		WarmPath:              memoryWarmPathFromEnv(),
		ColdBaseDir:           memoryColdPathFromEnv(),
		AutoRecallEnabled:     memoryAutoRecallEnabledFromEnv(),
		AutoRecallLimit:       memoryAutoRecallLimitFromEnv(),
		WarmTTL:               memoryWarmTTLFromEnv(),
		TemporalDecayEnabled:  memoryTemporalDecayEnabledFromEnv(),
		TemporalDecayHalfLife: configuredTemporalHalfLife(memoryTemporalDecayEnabledFromEnv(), memoryTemporalDecayHalfLifeFromEnv()),
		AnchorEnabled:         memoryAnchorEnabledFromEnv(),
		AnchorMinWeight:       configuredAnchorMinWeight(memoryAnchorEnabledFromEnv(), memoryAnchorMinWeightFromEnv()),
		EvolutionInterval:     memoryEvolutionIntervalFromEnv(),
		EvolutionEnabled:      memoryEvolutionEnabledFromEnv(),
		EvolutionUseWorker:    memoryEvolutionUseWorkerFromEnv(),
		EvolutionBatchSize:    memoryEvolutionBatchSizeFromEnv(),
		SessionStore:          sessionStore,
		Summarizer:            summarizer,
	}
}

func memoryManagerConfigFromAppConfig(cfg Config, sessionStore *session.Store, summarizer memory.Summarizer) memory.MemoryConfig {
	if !cfg.MemoryEvolutionUseWorker {
		summarizer = nil
	}
	return memory.MemoryConfig{
		WarmCapacity:          agentWarmMemoryCapacity,
		WarmPath:              cfg.MemoryWarmPath,
		ColdBaseDir:           cfg.MemoryColdPath,
		AutoRecallEnabled:     cfg.MemoryAutoRecallEnabled,
		AutoRecallLimit:       cfg.MemoryAutoRecallLimit,
		WarmTTL:               cfg.MemoryWarmTTL,
		TemporalDecayEnabled:  cfg.MemoryTemporalDecayEnabled,
		TemporalDecayHalfLife: configuredTemporalHalfLife(cfg.MemoryTemporalDecayEnabled, cfg.MemoryTemporalDecayHalfLife),
		AnchorEnabled:         cfg.MemoryAnchorEnabled,
		AnchorMinWeight:       configuredAnchorMinWeight(cfg.MemoryAnchorEnabled, cfg.MemoryAnchorMinWeight),
		EvolutionInterval:     cfg.MemoryEvolutionInterval,
		EvolutionEnabled:      cfg.MemoryEvolutionEnabled,
		EvolutionUseWorker:    cfg.MemoryEvolutionUseWorker,
		EvolutionBatchSize:    cfg.MemoryEvolutionBatchSize,
		SessionStore:          sessionStore,
		Summarizer:            summarizer,
	}
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
