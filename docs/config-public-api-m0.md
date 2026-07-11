# Config Public API Baseline (M0)

Date: 2026-06-06

Baseline checks:

- `timeout 60s go test ./config -timeout 60s`: passed before M1 extraction.
- Top-level `core/bridge/config/*.go`: 96 files at baseline.
- Split acceptance target: top-level `config/*.go <= 15`; each `config/internal/*` directory `<= 25` files.

External import contract:

- External packages import only `ghost-os/bridge/config`.
- `ghost-os/bridge/config/internal/*` is private to the config package tree.

Store methods:

- `Config() (Config, error)`
- `Snapshot() Snapshot`
- `PublicSnapshot() (Snapshot, error)`
- `ListProviders() ([]ProviderRecord, error)`
- `AddProvider(ProviderRecord) error`
- `UpdateProvider(name string, cfg ProviderRecord) error`
- `DeleteProvider(name string) error`
- `SetActiveProvider(name string) error`
- `ListTools() ([]ToolRecord, error)`
- `UpdateTool(ToolUpdateRequest) error`
- `SetSkillEnabled(skillID string, enabled bool) error`
- `SystemPrompts() (SystemPromptFiles, error)`
- `UpdateSystemPrompts(SystemPromptUpdateRequest) (SystemPromptFiles, error)`
- `Presets() ([]Preset, error)`
- `CreatePreset(PresetCreateRequest) (Preset, error)`
- `UpdatePreset(string, PresetUpdateRequest) (Preset, error)`
- `DeletePreset(string) (Preset, error)`
- `ApplyPreset(string) (Preset, error)`
- `Update(UpdateRequest) error`
- `SetProjectRoot(path string) error`

Public constructors and loaders:

- `NewStoreFromEnv() (Store, error)`
- `Load() (Config, error)`
- `LoadServerConfig() (ServerConfig, error)`
- `LoadExecutionConfig() (ExecutionConfig, error)`
- `LoadTaskConfig() (TaskConfig, error)`
- `LoadSystemPromptFiles(promptsDir string) (SystemPromptFiles, error)`
- `UpdateSystemPromptFiles(promptsDir string, req SystemPromptUpdateRequest) (SystemPromptFiles, error)`
- `LoadPresets(promptsDir string) ([]Preset, error)`
- `CreatePreset(promptsDir string, req PresetCreateRequest) (Preset, error)`
- `UpdatePreset(promptsDir string, presetID string, req PresetUpdateRequest) (Preset, error)`
- `DeletePreset(promptsDir string, presetID string) (Preset, error)`
- `ApplyPresetToSystemPromptFiles(files SystemPromptFiles, preset Preset) (SystemPromptFiles, error)`
- `FindPresetByID(presets []Preset, presetID string) (Preset, bool)`
- `LoadToolPromptOverrides(promptsDir string) (map[string]string, error)`
- `ToolBasePrompt(name string) (string, bool)`
- `ToolBasePrompts() map[string]string`
- `NormalizeProjectRoot(raw string) (string, error)`
- `WithProjectRootOverride(store Store, projectRoot string) Store`

Primary DTO fields:

- `Config`: `Provider`, `Worker`, `ToolSelector`, `ToolSearch`, `SkillBlocklist`, `ScriptExecSandboxMemoryMB`, `NativePersistent`, `NativeBinaryPath`, `NativeBinaryRoots`, `NativeBinaryCandidates`, `CodexCLIPath`, `NodeBinPath`, `NativeAllowedReadPaths`, `NativeAllowedWritePaths`, `ProjectRoot`, `Task`, `ChatPath`, `ResponseOptions`, `CodexStatelessRetryEnabled`, `PromptsPath`, `PromptsDir`, `PromptsCoreFiles`, `PromptsRuntimeConstraintFiles`, `PromptsResponseRuleFiles`, `SessionsPath`, `WebSearchTavilyURL`, `WebSearchExaURL`, `WebSearchTavilyAPIKey`, `WebSearchExaAPIKey`, `LLMCompletionRetryCount`, `LLMCompletionRetryIntervalMS`, `TaskExecutionTimeoutMS`, `SessionHumanLogFullEnabled`, `SessionSystemPromptVisible`, `AssistantMarkdownEnabled`, `ToolCallCompactOutputEnabled`, `MemoryModeEnabled`, `MicrocompactEnabled`, `SessionTitleMode`, `RelayDefaultStopPolicy`, `RelayDefaultMaxRounds`, `RelayDefaultExecutionTimeoutMS`, `MaxTurns`.
- `ServerConfig`: `BindAddr`, `APIToken`, `CORSOrigins`, `SessionsPath`.
- `ExecutionConfig`: `Persistent`, `NativeBinaryPath`, `NativeBinaryRoots`, `NativeBinaryCandidates`, `CodexCLIPath`, `NodeBinPath`, `AllowedReadPaths`, `AllowedWritePaths`, `ProjectRoot`.
- `TaskConfig`: `TasksPath`, `ExecutionTimeoutMS`, `WorkflowToolAllowlist`.
- `Snapshot`: `Provider`, `ProviderType`, `BaseURL`, `Model`, `ChatPath`, `ProjectRoot`, `MaxTurns`, `TaskExecutionTimeoutMS`, `RelayDefaultStopPolicy`, `RelayDefaultMaxRounds`, `RelayDefaultExecutionTimeoutMS`, `LLMCompletionRetryCount`, `LLMCompletionRetryIntervalMS`, `APIKeySet`, `ModelSelectionEnabled`, `SessionHumanLogFullEnabled`, `SessionSystemPromptVisible`, `AssistantMarkdownEnabled`, `ToolCallCompactOutputEnabled`, `MemoryModeEnabled`, `MicrocompactEnabled`, `SessionTitleMode`, `WebSearchTavilyURL`, `WebSearchExaURL`, `WebSearchTavilyAPIKeySet`, `WebSearchExaAPIKeySet`.
- `UpdateRequest`: `Provider`, `APIKey`, `BaseURL`, `Model`, `ChatPath`, `ProjectRoot`, `MaxTurns`, `TaskExecutionTimeoutMS`, `RelayDefaultStopPolicy`, `RelayDefaultMaxRounds`, `RelayDefaultExecutionTimeoutMS`, `LLMCompletionRetryCount`, `LLMCompletionRetryIntervalMS`, `SessionHumanLogFullEnabled`, `SessionSystemPromptVisible`, `AssistantMarkdownEnabled`, `ToolCallCompactOutputEnabled`, `MemoryModeEnabled`, `MicrocompactEnabled`, `SessionTitleMode`, `WebSearchTavilyURL`, `WebSearchExaURL`, `WebSearchTavilyAPIKey`, `WebSearchExaAPIKey`, `TraceID`.
- `ProviderConfig`: `Type`, `APIKey`, `BaseURL`, `Model`, `Headers`, `AnthropicVersion`, `AnthropicMaxTokens`, `ContextWindowTokens`, `ResponseReserveTokens`, `ModelContextWindowTokens`, `ModelResponseReserveTokens`.
- `ProviderRecord`: `Name`, `Type`, `BaseURL`, `APIKey`, `Models`, `ContextWindowTokens`, `ResponseReserveTokens`, `ModelContextWindowTokens`, `ModelResponseReserveTokens`.
- `ToolRecord`: `Name`, `Enabled`, `PromptOverride`, `SandboxMemoryMB`.
- `ToolUpdateRequest`: `Name`, `Enabled`, `PromptOverride`, `SandboxMemoryMB`.
- `ToolSelectorConfig`: `Enabled`, `Mode`, `Model`, `TimeoutMS`, `Confidence`, `Shadow`, `RecentMsgs`, `AllowlistOnly`, `Allowlist`, `Blocklist`, `PromptOverrides`.
- `ToolSearchConfig`: `Enabled`, `IdleTurns`.
- `PresetPromptRefs`: `Rule`, `CoreJob`, `Memory`, `Context`.
- `Preset`: `ID`, `Name`, `ToolAllowlist`, `PromptRefs`.
- `PresetCreateRequest`: `Name`, `ToolAllowlist`, `PromptRefs`, `TraceID`.
- `PresetUpdateRequest`: `Name`, `ToolAllowlist`, `PromptRefs`, `TraceID`.
- `SystemPromptLibraryItem`: `ID`, `Name`, `InsertPoint`, `Content`, `Active`.
- `SystemPromptFiles`: `CorePrompt`, `PromptLibrary`.
- `SystemPromptUpdateRequest`: `CorePrompt`, `PromptLibrary`, `TraceID`.
- `WorkerConfig`: `Model`, `MaxConcurrency`, `MaxFiles`, `MaxFileChunks`.

Public defaults:

- `DefaultProvider`, `DefaultBaseURL`, `DefaultAnthropicBaseURL`, `DefaultModel`, `DefaultPromptsPath`, `DefaultPromptsDir`, `DefaultSessionsPath`, `DefaultTasksPath`, `DefaultTaskExecutionTimeoutMS`, `DefaultAnthropicVersion`, `DefaultAnthropicMaxTokens`, `DefaultRelayStopPolicy`, `DefaultRelayMaxRounds`, `DefaultRelayExecutionTimeoutMS`, `DefaultMaxTurns`, `DefaultLLMCompletionRetryCount`, `DefaultLLMCompletionRetryIntervalMS`, `DefaultModelSelectionEnabled`, `DefaultWorkerMaxConcurrency`, `DefaultWorkerMaxFiles`, `DefaultWorkerMaxFileChunks`, `DefaultScriptExecSandboxMemoryMB`, `DefaultSessionSystemPromptVisible`, `DefaultToolSelectorTimeoutMS`, `DefaultToolSelectorConfidence`, `DefaultToolSelectorRecentMsgs`, `DefaultToolSearchIdleTurns`, `DefaultAssistantMarkdownEnabled`, `DefaultToolCallCompactOutputEnabled`, `DefaultMemoryModeEnabled`, `DefaultMicrocompactEnabled`, `DefaultSessionTitleMode`.

Public enum constants:

- `RelayStopPolicyAIDecides`, `RelayStopPolicyMaxRounds`.
- `SessionTitleModeSessionID`, `SessionTitleModeFirstMessage`, `SessionTitleModeAIGenerated`.
- `SystemPromptInsertPointRule`, `SystemPromptInsertPointCoreJob`, `SystemPromptInsertPointMemory`, `SystemPromptInsertPointContext`.

Public errors:

- `ErrProviderNameRequired`, `ErrProviderTypeInvalid`, `ErrProviderBaseURLRequired`, `ErrProviderNotFound`, `ErrProviderExists`, `ErrModelSelectionDisabled`, `ErrToolNameRequired`, `ErrToolNotFound`, `ErrToolUpdateEmpty`, `ErrToolConfigInvalid`, `ErrSystemPromptUpdateEmpty`, `ErrSystemPromptUpdateConflict`, `ErrSystemPromptLibraryInvalid`, `ErrPresetInvalid`, `ErrPresetIDRequired`, `ErrPresetNotFound`, `ErrPresetUpdateEmpty`.
