package app

import (
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

type ProviderConfig struct {
	Type               llm.Provider
	APIKey             string
	BaseURL            string
	Model              string
	Headers            map[string]string
	AnthropicVersion   string
	AnthropicMaxTokens int
}

type MemoryRuntimeConfig struct {
	WarmPath                                  string
	ColdPath                                  string
	LedgerPath                                string
	LedgerReadEnabled                         bool
	AutoRecallEnabled                         bool
	AutoRecallLimit                           int
	WarmTTL                                   time.Duration
	TemporalDecayEnabled                      bool
	TemporalDecayHalfLife                     time.Duration
	AnchorEnabled                             bool
	AnchorMinWeight                           float64
	EvolutionInterval                         time.Duration
	EvolutionEnabled                          bool
	EvolutionUseWorker                        bool
	EvolutionBatchSize                        int
	GraphEnabled                              bool
	GraphPath                                 string
	GraphExtractOnArchive                     bool
	GraphExtractOnEvolve                      bool
	GraphMaxHops                              int
	GraphMaxHits                              int
	GraphMinConfidence                        float64
	GraphNamespace                            string
	GraphDebugEnabled                         bool
	DecisionEnabled                           bool
	DecisionCaptureOnTurn                     bool
	DecisionPath                              string
	DecisionMaxHits                           int
	DecisionMinConfidence                     float64
	DecisionMinReuseScore                     float64
	DecisionRecipeEnabled                     bool
	DecisionRecipeInterval                    time.Duration
	DecisionRecipeMinSupport                  int
	DecisionDebugEnabled                      bool
	DecisionSelectorHintEnabled               bool
	DecisionRecipeReuseEnabled                bool
	DecisionRecipeReuseEnabledSet             bool
	DecisionRecipeExecutionTrackingEnabled    bool
	DecisionRecipeExecutionTrackingEnabledSet bool
	DecisionRecipeBackfillEnabled             bool
	DecisionRecipeBackfillEnabledSet          bool
	DecisionRecipeDefaultEnabled              bool
	DecisionRecipeDefaultEnabledSet           bool
	DecisionRecipeDefaultGrayPercent          int
	DecisionRecipeMinSelectionConfidence      float64
	DecisionRecipeMinSuccessRate              float64
	DecisionRecipeBackfillBatchSize           int
	DecisionRecipeBackfillInterval            time.Duration
}

type RSSConfig struct {
	FeedsPath           string
	InboxPath           string
	BriefingsPath       string
	ReportsPath         string
	PollEnabled         bool
	PollInterval        time.Duration
	PollMaxItemsPerFeed int
	AIBatchSize         int
	BriefingEnabled     bool
	BriefingInterval    time.Duration
}

type WorkerConfig struct {
	Model          string
	MaxConcurrency int
	MaxFiles       int
	MaxFileChunks  int
}

type ToolSelectorConfig struct {
	Enabled    bool
	Mode       string
	Model      string
	TimeoutMS  int
	Confidence float64
	Shadow     bool
	RecentMsgs int
	Allowlist  []string
	Blocklist  []string
}

// Config 描述 bridge 在运行时依赖的最小配置集合。
type Config struct {
	Provider              ProviderConfig
	Memory                MemoryRuntimeConfig
	RSS                   RSSConfig
	Worker                WorkerConfig
	ToolSelector          ToolSelectorConfig
	NativePersistent      bool
	ChatPath              string
	PromptsPath           string
	SessionsPath          string
	WebSearchTavilyAPIKey string
	MaxTurns              int
}

type runtimeConfig struct {
	ProviderName     string
	Provider         llm.Provider
	APIKey           string
	BaseURL          string
	Model            string
	ChatPath         string
	NativePersistent bool
}

const (
	defaultProvider                                   = llm.ProviderOpenAI
	defaultBaseURL                                    = "https://api.openai.com/v1"
	defaultAnthropicBaseURL                           = "https://api.anthropic.com"
	defaultModel                                      = "gpt-4o"
	defaultPromptsPath                                = "prompts.yaml"
	defaultSessionsPath                               = "~/.ghost-os/sessions"
	defaultRSSFeedsPath                               = "~/.ghost-os/rss/feeds.json"
	defaultRSSInboxPath                               = "~/.ghost-os/rss/inbox.json"
	defaultRSSBriefingsPath                           = "~/.ghost-os/rss/briefings.json"
	defaultRSSReportsPath                             = "~/.ghost-os/rss/reports/index.json"
	defaultRSSPollInterval                            = 15 * time.Minute
	defaultRSSPollMaxItemsPerFeed                     = 10
	defaultRSSAIBatchSize                             = 5
	defaultRSSBriefingInterval                        = 30 * time.Minute
	defaultTasksPath                                  = "~/.ghost-os/tasks"
	defaultMemoryWarmPath                             = "~/.ghost-os/memory/warm.json"
	defaultMemoryColdPath                             = "~/.ghost-os/memory/cold"
	defaultMemoryGraphPath                            = "~/.ghost-os/memory/graph"
	defaultMemoryDecisionPath                         = "~/.ghost-os/memory/decision"
	defaultMemoryAutoRecallLimit                      = 5
	defaultMemoryWarmTTL                              = 24 * time.Hour
	defaultMemoryTemporalHalfLife                     = 72 * time.Hour
	defaultMemoryAnchorMinWeight                      = 0.65
	defaultMemoryEvolutionInterval                    = time.Hour
	defaultMemoryEvolutionBatchSize                   = 20
	defaultMemoryGraphMaxHops                         = 1
	defaultMemoryGraphMaxHits                         = 6
	defaultMemoryGraphNamespace                       = "default"
	defaultMemoryGraphMinConfidence                   = 0.72
	defaultMemoryDecisionMaxHits                      = 4
	defaultMemoryDecisionCaptureOnTurn                = true
	defaultMemoryDecisionMinConfidence                = 0.75
	defaultMemoryDecisionMinReuseScore                = 0.70
	defaultMemoryDecisionRecipeInterval               = 6 * time.Hour
	defaultMemoryDecisionRecipeMinSupport             = 3
	defaultMemoryDecisionSelectorHintEnabled          = true
	defaultMemoryDecisionRecipeDefaultGrayPercent     = 10
	defaultMemoryDecisionRecipeMinSelectionConfidence = 0.72
	defaultMemoryDecisionRecipeMinSuccessRate         = 0.66
	defaultMemoryDecisionRecipeBackfillBatchSize      = 50
	defaultMemoryDecisionRecipeBackfillInterval       = 30 * time.Minute
	defaultAnthropicVersion                           = "2023-06-01"
	defaultAnthropicMaxTokens                         = 1024
	defaultMaxTurns                                   = 20
	defaultWorkerMaxConcurrency                       = 4
	defaultWorkerMaxFiles                             = 20
	defaultWorkerMaxFileChunks                        = 4
	defaultToolSelectorTimeoutMS                      = 1500
	defaultToolSelectorConfidence                     = 0.75
	defaultToolSelectorRecentMsgs                     = 6
)

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
