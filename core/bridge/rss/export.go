package rss

import (
	"strings"
	"time"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/tools"
)

type Config = bridgeconfig.Config
type RSSInboxFetcher = rssInboxFetcher
type RSSInboxClassifier = rssInboxClassifier
type RSSBriefingBuilder = rssBriefingBuilder
type RSSReportBuilder = rssReportBuilder
type RSSInboxClassification = rssInboxClassification
type RSSInboxCandidate = rssInboxCandidate
type RSSBriefingDraft = rssBriefingDraft
type RSSBriefingDraftHighlight = rssBriefingDraftHighlight

const (
	defaultRSSFeedsPath               = bridgeconfig.DefaultRSSFeedsPath
	defaultRSSInboxPath               = bridgeconfig.DefaultRSSInboxPath
	defaultRSSBriefingsPath           = bridgeconfig.DefaultRSSBriefingsPath
	defaultRSSReportsPath             = bridgeconfig.DefaultRSSReportsPath
	defaultRSSPollInterval            = bridgeconfig.DefaultRSSPollInterval
	defaultRSSPollMaxItemsPerFeed     = bridgeconfig.DefaultRSSPollMaxItemsPerFeed
	defaultRSSAIBatchSize             = bridgeconfig.DefaultRSSAIBatchSize
	DefaultRSSPollTaskID              = defaultRSSPollTaskID
	DefaultRSSBriefingTaskID          = defaultRSSBriefingTaskID
	DefaultRSSAggregateWindowHours    = defaultRSSAggregateWindowHours
	DefaultRSSAggregateItemLimit      = defaultRSSAggregateItemLimit
	DefaultRSSBriefingGroupLimit      = defaultRSSBriefingGroupLimit
	DefaultRSSBriefingHighlightsLimit = defaultRSSBriefingHighlightsLimit
)

type agentRuntimeDependencies struct {
	cfg          Config
	client       agent.Completer
	registry     *tools.Registry
	systemPrompt string
	cleanup      func()
}

func (d agentRuntimeDependencies) Close() {
	if d.cleanup != nil {
		d.cleanup()
	}
}

type AgentRuntimeFactory interface {
	Build(store *ConfigStore) (agentRuntimeDependencies, error)
}

type runtimeFactoryAdapter struct {
	inner bridgeruntime.AgentRuntimeFactory
}

func (f runtimeFactoryAdapter) Build(store *ConfigStore) (agentRuntimeDependencies, error) {
	deps, err := f.inner.Build(store.unwrap())
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	return agentRuntimeDependencies{
		cfg:          deps.Config(),
		client:       deps.Client(),
		registry:     deps.Registry(),
		systemPrompt: deps.SystemPrompt(),
		cleanup:      deps.Close,
	}, nil
}

func newAgentRuntimeFactory() AgentRuntimeFactory {
	return runtimeFactoryAdapter{inner: bridgeruntime.NewAgentRuntimeFactory()}
}

type ConfigStore struct {
	inner bridgeconfig.Store
}

func wrapConfigStore(store bridgeconfig.Store) *ConfigStore {
	if store == nil {
		return nil
	}
	return &ConfigStore{inner: store}
}

func (s *ConfigStore) unwrap() bridgeconfig.Store {
	if s == nil {
		return nil
	}
	return s.inner
}

func (s *ConfigStore) Config() (Config, error) {
	if s == nil || s.inner == nil {
		return LoadConfig()
	}
	return s.inner.Config()
}

func LoadConfig() (Config, error) {
	return bridgeconfig.Load()
}

func providerClientOptions(cfg Config, model string) llm.ClientOptions {
	resolvedModel := strings.TrimSpace(model)
	if resolvedModel == "" {
		resolvedModel = strings.TrimSpace(cfg.Provider.Model)
	}
	return llm.ClientOptions{
		Provider:           cfg.Provider.Type,
		BaseURL:            cfg.Provider.BaseURL,
		APIKey:             cfg.Provider.APIKey,
		Model:              resolvedModel,
		ChatPath:           cfg.ChatPath,
		Headers:            cfg.Provider.Headers,
		AnthropicVersion:   cfg.Provider.AnthropicVersion,
		AnthropicMaxTokens: cfg.Provider.AnthropicMaxTokens,
	}
}

func effectiveWorkerModel(cfg Config) string {
	if model := strings.TrimSpace(cfg.Worker.Model); model != "" {
		return model
	}
	return strings.TrimSpace(cfg.Provider.Model)
}

func resolveUserPath(pathValue string) (string, error) {
	return bridgeconfig.ResolveUserPath(pathValue)
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}

func stripJSONCodeFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 {
		return trimmed
	}
	lines = lines[1:]
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
		lines = lines[:len(lines)-1]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func NewRSSInboxServiceFromConfig(store bridgeconfig.Store) (*RSSInboxService, error) {
	return newRSSInboxServiceFromConfig(wrapConfigStore(store))
}

func NewLLMRSSBriefingBuilder(client llm.Completer, cfg Config) RSSBriefingBuilder {
	return &llmRSSBriefingBuilder{client: client, cfg: cfg, timeout: defaultRSSBriefingTimeout}
}

func (s *RSSInboxService) FeedStore() *rsssubscriptions.FeedStore {
	if s == nil {
		return nil
	}
	return s.feedStore
}

func (s *RSSInboxService) SetFetcher(fetcher RSSInboxFetcher) {
	if s != nil && fetcher != nil {
		s.fetcher = fetcher
	}
}

func (s *RSSInboxService) SetClassifier(classifier RSSInboxClassifier) {
	if s != nil && classifier != nil {
		s.classifier = classifier
	}
}

func (s *RSSInboxService) SetBriefingBuilder(builder RSSBriefingBuilder) {
	if s != nil && builder != nil {
		s.briefingBuilder = builder
	}
}

func (s *RSSInboxService) SetReportBuilder(builder RSSReportBuilder) {
	if s != nil && builder != nil {
		s.reportBuilder = builder
	}
}

func (s *RSSInboxService) SetNow(now func() time.Time) {
	if s != nil && now != nil {
		s.now = now
	}
}

func buildSystemPromptForCatalog(cfg Config, catalog tools.ToolCatalog) (string, error) {
	return bridgeruntime.BuildSystemPromptForCatalog(cfg, catalog)
}
