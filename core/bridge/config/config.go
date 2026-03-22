package config

import (
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

type ProviderConfig struct {
	Type                       llm.Provider
	APIKey                     string
	BaseURL                    string
	Model                      string
	Headers                    map[string]string
	AnthropicVersion           string
	AnthropicMaxTokens         int
	ContextWindowTokens        int
	ResponseReserveTokens      int
	ModelContextWindowTokens   map[string]int
	ModelResponseReserveTokens map[string]int
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

type ProviderRecord struct {
	Name                       string
	Type                       llm.Provider
	BaseURL                    string
	APIKey                     *string
	Models                     []string
	ContextWindowTokens        int
	ResponseReserveTokens      int
	ModelContextWindowTokens   map[string]int
	ModelResponseReserveTokens map[string]int
}

type GraphQLDomainConfig struct {
	Name          string
	Description   string
	RootQueries   []string
	Types         []string
	MaxDepth      int
	MaxFields     int
	MaxRootFields int
}

type GraphQLSourceConfig struct {
	Name             string
	Description      string
	Endpoint         string
	APIKey           string
	SchemaPath       string
	TimeoutMS        int
	MaxResponseBytes int
	Headers          map[string]string
	MaxDepth         int
	MaxFields        int
	MaxRootFields    int
	MaxFragments     int
	Domains          []GraphQLDomainConfig
}

type GraphQLMutationPolicyConfig struct {
	Name                    string
	Description             string
	Source                  string
	Domain                  string
	RootMutation            string
	ApprovalRequired        bool
	IdempotencyMode         string
	IdempotencyHeader       string
	IdempotencyVariablePath string
	MaxDepth                int
	MaxFields               int
	MaxRootFields           int
	MaxFragments            int
}

type GraphQLConfig struct {
	ToolRuntimeEnabled bool
	DefaultSource      string
	Sources            []GraphQLSourceConfig
	MutationPolicies   []GraphQLMutationPolicyConfig
}

type ToolSelectorConfig struct {
	Enabled    bool
	Mode       string
	Model      string
	TimeoutMS  int
	Confidence float64
	Shadow     bool
	RecentMsgs int
	// AllowlistOnly enables allowlist-mode: the agent is only exposed to tools in Allowlist (plus ask_human if registered).
	AllowlistOnly bool
	Allowlist     []string
	Blocklist     []string
}

type ToolSearchConfig struct {
	Enabled   bool
	IdleTurns int
}

type MemoryAugmentationConfig struct {
	Enabled             bool
	LearningEnabled     bool
	RecallEnabled       bool
	MaxRecallItems      int
	MinConfidence       float64
	SessionScopeEnabled bool
	UserScopeEnabled    bool
	LLMModel            string
	UserScopeID         string
}

type ServerConfig struct {
	BindAddr     string
	APIToken     string
	CORSOrigins  []string
	SessionsPath string
}

type ExecutionConfig struct {
	Persistent             bool
	NativeBinaryPath       string
	NativeBinaryRoots      []string
	NativeBinaryCandidates []string
	AllowedReadPaths       []string
	AllowedWritePaths      []string
	ProjectRoot            string
}

type TaskConfig struct {
	TasksPath string
}

// Config 描述 bridge 在运行时依赖的最小配置集合。
type Config struct {
	Provider                      ProviderConfig
	RSS                           RSSConfig
	Worker                        WorkerConfig
	GraphQL                       GraphQLConfig
	ToolSelector                  ToolSelectorConfig
	ToolSearch                    ToolSearchConfig
	MemoryAugmentation            MemoryAugmentationConfig
	NativePersistent              bool
	NativeBinaryPath              string
	NativeBinaryRoots             []string
	NativeBinaryCandidates        []string
	NativeAllowedReadPaths        []string
	NativeAllowedWritePaths       []string
	ProjectRoot                   string
	ChatPath                      string
	PromptsPath                   string
	PromptsDir                    string
	PromptsCoreFiles              []string
	PromptsRuntimeConstraintFiles []string
	PromptsResponseRuleFiles      []string
	SessionsPath                  string
	WebSearchTavilyAPIKey         string
	WebSearchExaAPIKey            string
	ProMaxIterations              int
	MaxTurns                      int
}

// runtimeConfig is the resolved runtime layer: persisted file DTO merged with
// env fallback, normalized once for execution and public exposure.
type runtimeConfig struct {
	ProviderName               string
	Provider                   llm.Provider
	APIKey                     string
	BaseURL                    string
	Model                      string
	ChatPath                   string
	NativePersistent           bool
	ProjectRoot                string
	ModelSelectionEnabled      bool
	ContextWindowTokens        int
	ResponseReserveTokens      int
	ModelContextWindowTokens   map[string]int
	ModelResponseReserveTokens map[string]int
	WebSearchTavilyAPIKey      string
	WebSearchExaAPIKey         string
	GraphQL                    GraphQLConfig
}

const (
	defaultProvider                = llm.ProviderOpenAI
	defaultBaseURL                 = "https://api.openai.com/v1"
	defaultAnthropicBaseURL        = "https://api.anthropic.com"
	defaultModel                   = "gpt-4o"
	defaultPromptsPath             = "prompts.yaml"
	defaultPromptsDir              = "~/.ghost-os/prompts"
	defaultSessionsPath            = "~/.ghost-os/sessions"
	defaultRSSFeedsPath            = "~/.ghost-os/rss/feeds.json"
	defaultRSSInboxPath            = "~/.ghost-os/rss/inbox.json"
	defaultRSSBriefingsPath        = "~/.ghost-os/rss/briefings.json"
	defaultRSSReportsPath          = "~/.ghost-os/rss/reports/index.json"
	defaultRSSPollInterval         = 15 * time.Minute
	defaultRSSPollMaxItemsPerFeed  = 10
	defaultRSSAIBatchSize          = 5
	defaultRSSBriefingInterval     = 30 * time.Minute
	defaultTasksPath               = "~/.ghost-os/tasks"
	defaultAnthropicVersion        = "2023-06-01"
	defaultAnthropicMaxTokens      = 1024
	defaultProMaxIterations        = 20
	defaultMaxTurns                = 20
	defaultWorkerMaxConcurrency    = 4
	defaultWorkerMaxFiles          = 20
	defaultWorkerMaxFileChunks     = 4
	defaultGraphQLTimeoutMS        = 10_000
	defaultGraphQLMaxResponseBytes = 1 << 20
	defaultGraphQLMaxDepth         = 8
	defaultGraphQLMaxFields        = 64
	defaultGraphQLMaxRootFields    = 3
	defaultGraphQLMaxFragments     = 8
	defaultToolSelectorTimeoutMS   = 1500
	defaultToolSelectorConfidence  = 0.75
	defaultToolSelectorRecentMsgs  = 6
	defaultToolSearchIdleTurns     = 3
	defaultMemoryRecallItems       = 8
	defaultMemoryMinConfidence     = 0.7
	defaultMemoryUserScopeID       = "local-user"
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
