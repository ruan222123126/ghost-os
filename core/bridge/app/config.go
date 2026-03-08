package app

import (
	"time"

	"ghost-os/bridge/llm"
)

// Config 描述 bridge 在运行时依赖的最小配置集合。
type Config struct {
	Provider                                        llm.Provider
	APIKey                                          string
	BaseURL                                         string
	Model                                           string
	NativePersistent                                bool
	WorkerModel                                     string
	ChatPath                                        string
	PromptsPath                                     string
	SessionsPath                                    string
	RSSFeedsPath                                    string
	RSSInboxPath                                    string
	RSSPollEnabled                                  bool
	RSSPollInterval                                 time.Duration
	RSSPollMaxItemsPerFeed                          int
	RSSAIBatchSize                                  int
	MemoryWarmPath                                  string
	MemoryColdPath                                  string
	MemoryLedgerPath                                string
	MemoryLedgerDualWrite                           bool
	MemoryLedgerReadEnabled                         bool
	MemoryLedgerShadowCompare                       bool
	MemoryAutoRecallEnabled                         bool
	MemoryAutoRecallLimit                           int
	MemoryWarmTTL                                   time.Duration
	MemoryTemporalDecayEnabled                      bool
	MemoryTemporalDecayHalfLife                     time.Duration
	MemoryAnchorEnabled                             bool
	MemoryAnchorMinWeight                           float64
	MemoryEvolutionInterval                         time.Duration
	MemoryEvolutionEnabled                          bool
	MemoryEvolutionUseWorker                        bool
	MemoryEvolutionBatchSize                        int
	MemoryGraphEnabled                              bool
	MemoryGraphPath                                 string
	MemoryGraphExtractOnArchive                     bool
	MemoryGraphExtractOnEvolve                      bool
	MemoryGraphMaxHops                              int
	MemoryGraphMaxHits                              int
	MemoryGraphMinConfidence                        float64
	MemoryGraphNamespace                            string
	MemoryGraphDebugEnabled                         bool
	MemoryDecisionEnabled                           bool
	MemoryDecisionCaptureOnTurn                     bool
	MemoryDecisionPath                              string
	MemoryDecisionMaxHits                           int
	MemoryDecisionMinConfidence                     float64
	MemoryDecisionMinReuseScore                     float64
	MemoryDecisionRecipeEnabled                     bool
	MemoryDecisionRecipeInterval                    time.Duration
	MemoryDecisionRecipeMinSupport                  int
	MemoryDecisionDebugEnabled                      bool
	MemoryDecisionSelectorHintEnabled               bool
	MemoryDecisionRecipeReuseEnabled                bool
	MemoryDecisionRecipeReuseEnabledSet             bool
	MemoryDecisionRecipeExecutionTrackingEnabled    bool
	MemoryDecisionRecipeExecutionTrackingEnabledSet bool
	MemoryDecisionRecipeBackfillEnabled             bool
	MemoryDecisionRecipeBackfillEnabledSet          bool
	MemoryDecisionRecipeDefaultEnabled              bool
	MemoryDecisionRecipeDefaultEnabledSet           bool
	MemoryDecisionRecipeDefaultGrayPercent          int
	MemoryDecisionRecipeMinSelectionConfidence      float64
	MemoryDecisionRecipeMinSuccessRate              float64
	MemoryDecisionRecipeBackfillBatchSize           int
	MemoryDecisionRecipeBackfillInterval            time.Duration
	ProviderHeaders                                 map[string]string
	AnthropicVersion                                string
	AnthropicMaxTokens                              int
	MaxTurns                                        int
	WorkerMaxConcurrency                            int
	WorkerMaxFiles                                  int
	WorkerMaxFileChunks                             int
	ToolSelectorEnabled                             bool
	ToolSelectorMode                                string
	ToolSelectorModel                               string
	ToolSelectorTimeoutMS                           int
	ToolSelectorConfidence                          float64
	ToolSelectorShadow                              bool
	ToolSelectorRecentMsgs                          int
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
	defaultRSSPollInterval                            = 15 * time.Minute
	defaultRSSPollMaxItemsPerFeed                     = 10
	defaultRSSAIBatchSize                             = 5
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
