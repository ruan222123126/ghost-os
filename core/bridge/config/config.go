package config

import "ghost-os/bridge/llm"

type ServerConfig struct {
	BindAddr     string
	APIToken     string
	CORSOrigins  []string
	SessionsPath string
	MobileWebRTC MobileWebRTCConfig
}

type MobileICEServerConfig struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

type MobileWebRTCConfig struct {
	Enabled             bool
	SignalingURL        string
	SignalingToken      string
	PCID                string
	ICEServers          []MobileICEServerConfig
	CredentialStorePath string
}

type ExecutionConfig struct {
	Persistent             bool
	NativeBinaryPath       string
	NativeBinaryRoots      []string
	NativeBinaryCandidates []string
	CodexCLIPath           string
	NodeBinPath            string
	AllowedReadPaths       []string
	AllowedWritePaths      []string
	ProjectRoot            string
}

type TaskConfig struct {
	TasksPath             string
	ExecutionTimeoutMS    int
	WorkflowToolAllowlist []string
}

type WorkerConfig struct {
	Model          string
	MaxConcurrency int
	MaxFiles       int
	MaxFileChunks  int
}

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

type ProviderRecord struct {
	Name                       string
	Type                       llm.Provider
	BaseURL                    string
	APIKey                     *string
	ProviderID                 string
	UpdatedAt                  string
	DeletedAt                  string
	Models                     []string
	ContextWindowTokens        int
	ResponseReserveTokens      int
	ModelContextWindowTokens   map[string]int
	ModelResponseReserveTokens map[string]int
}

type ToolRecord struct {
	Name            string
	Enabled         bool
	PromptOverride  string
	SandboxMemoryMB *int
}

type ToolUpdateRequest struct {
	Name            string
	Enabled         *bool
	PromptOverride  *string
	SandboxMemoryMB *int
}

type ToolSelectorConfig struct {
	Enabled    bool
	Mode       string
	Model      string
	TimeoutMS  int
	Confidence float64
	Shadow     bool
	RecentMsgs int
	// AllowlistOnly switches tool_allowlist from resident-only mode to strict static visibility mode.
	// When false, Allowlist defines resident tools and selector-visible static tools still include other non-blocked tools.
	AllowlistOnly   bool
	Allowlist       []string
	Blocklist       []string
	PromptOverrides map[string]string
}

type ToolSearchConfig struct {
	Enabled   bool
	IdleTurns int
}

// Config 描述 bridge 在运行时依赖的最小配置集合。
type Config struct {
	Provider                       ProviderConfig
	Worker                         WorkerConfig
	ToolSelector                   ToolSelectorConfig
	ToolSearch                     ToolSearchConfig
	SkillBlocklist                 []string
	ScriptExecSandboxMemoryMB      int
	NativePersistent               bool
	NativeBinaryPath               string
	NativeBinaryRoots              []string
	NativeBinaryCandidates         []string
	CodexCLIPath                   string
	NodeBinPath                    string
	NativeAllowedReadPaths         []string
	NativeAllowedWritePaths        []string
	ProjectRoot                    string
	Task                           TaskConfig
	ChatPath                       string
	ResponseOptions                llm.ResponseOptions
	CodexStatelessRetryEnabled     bool
	PromptsPath                    string
	PromptsDir                     string
	PromptsCoreFiles               []string
	PromptsRuntimeConstraintFiles  []string
	PromptsResponseRuleFiles       []string
	SessionsPath                   string
	WebSearchTavilyURL             string
	WebSearchExaURL                string
	WebSearchTavilyAPIKey          string
	WebSearchExaAPIKey             string
	LLMCompletionRetryCount        int
	LLMCompletionRetryIntervalMS   int
	TaskExecutionTimeoutMS         int
	SessionHumanLogFullEnabled     bool
	SessionSystemPromptVisible     bool
	AssistantMarkdownEnabled       bool
	ToolCallCompactOutputEnabled   bool
	MemoryModeEnabled              bool
	MicrocompactEnabled            bool
	SessionTitleMode               string
	RelayDefaultStopPolicy         string
	RelayDefaultMaxRounds          int
	RelayDefaultExecutionTimeoutMS int
	MaxTurns                       int
}
