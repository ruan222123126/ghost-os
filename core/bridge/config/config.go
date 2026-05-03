package config

import "ghost-os/bridge/llm"

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
	TasksPath             string
	WorkflowToolAllowlist []string
}

// Config 描述 bridge 在运行时依赖的最小配置集合。
type Config struct {
	Provider                      ProviderConfig
	RSS                           RSSConfig
	Worker                        WorkerConfig
	ToolSelector                  ToolSelectorConfig
	ToolSearch                    ToolSearchConfig
	SkillBlocklist                []string
	ScriptExecSandboxMemoryMB     int
	NativePersistent              bool
	NativeBinaryPath              string
	NativeBinaryRoots             []string
	NativeBinaryCandidates        []string
	NativeAllowedReadPaths        []string
	NativeAllowedWritePaths       []string
	ProjectRoot                   string
	Task                          TaskConfig
	ChatPath                      string
	ResponseOptions               llm.ResponseOptions
	CodexStatelessRetryEnabled    bool
	PromptsPath                   string
	PromptsDir                    string
	PromptsCoreFiles              []string
	PromptsRuntimeConstraintFiles []string
	PromptsResponseRuleFiles      []string
	SessionsPath                  string
	WebSearchTavilyURL            string
	WebSearchExaURL               string
	WebSearchTavilyAPIKey         string
	WebSearchExaAPIKey            string
	LLMCompletionRetryCount       int
	LLMCompletionRetryIntervalMS  int
	SessionHumanLogFullEnabled    bool
	SessionSystemPromptVisible    bool
	AssistantMarkdownEnabled      bool
	ToolCallCompactOutputEnabled  bool
	MemoryModeEnabled             bool
	MicrocompactEnabled           bool
	ProMaxIterations              int
	MaxTurns                      int
}
