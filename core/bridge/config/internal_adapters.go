package config

import (
	"time"

	"ghost-os/bridge/config/internal/providers"
	configruntime "ghost-os/bridge/config/internal/runtime"
	"ghost-os/bridge/config/internal/storage"
	"ghost-os/bridge/llm"
)

type providerConfig = providers.Record
type providerFileConfig = storage.ProviderFileConfig
type bridgeFileConfig = storage.FileConfig
type envSnapshot = storage.EnvSnapshot
type runtimeConfig = configruntime.Snapshot

func currentEnv() envSnapshot {
	return storage.CurrentEnv()
}

func envFromEntries(entries []string) envSnapshot {
	return storage.EnvFromEntries(entries)
}

func getenvDefault(name, fallback string) string {
	return currentEnv().DefaultValue(name, fallback)
}

func loadBridgeFileConfig() (bridgeFileConfig, string, error) {
	return storage.Load()
}

func writeBridgeFileConfig(path string, cfg bridgeFileConfig) error {
	return storage.Write(path, cfg)
}

func normalizeBridgeFileConfigForWrite(cfg bridgeFileConfig) (bridgeFileConfig, error) {
	return storage.NormalizeForWrite(cfg)
}

func normalizedActiveProviderName(providers []providerConfig, preferred ...*string) *string {
	return storage.NormalizedActiveProviderName(providers, preferred...)
}

func normalizeStringList(raw []string) []string {
	return storage.NormalizeStringList(raw)
}

func resolveUserPath(pathValue string) (string, error) {
	return storage.ResolveUserPath(pathValue)
}

func normalizeConfiguredPathList(values []string) []string {
	return storage.NormalizeConfiguredPathList(values)
}

func cloneOptionalStringPointer(raw *string) *string {
	return storage.CloneOptionalStringPointer(raw)
}

func cloneBoolPointer(raw *bool) *bool {
	return storage.CloneBoolPointer(raw)
}

func cloneIntPointer(raw *int) *int {
	return storage.CloneIntPointer(raw)
}

func cloneStringPointer(raw *string) *string {
	return storage.CloneStringPointer(raw)
}

func optionalStringPointer(raw string) *string {
	return storage.OptionalStringPointer(raw)
}

func stringPointer(raw string) *string {
	return storage.StringPointer(raw)
}

func stringValue(raw *string) string {
	return storage.StringValue(raw)
}

func parseProviderHeaders(raw string) (map[string]string, error) {
	return storage.ParseProviderHeaders(raw)
}

func parseNamedHeaders(raw string, envName string) (map[string]string, error) {
	return storage.ParseNamedHeaders(raw, envName)
}

func resolveNativePersistent(raw *bool, env envSnapshot) (bool, error) {
	return storage.ResolveNativePersistent(raw, env)
}

func parseBoolValue(raw, fieldName string, fallback bool) (bool, error) {
	return storage.ParseBoolValue(raw, fieldName, fallback)
}

func parsePositiveIntValue(raw, fieldName string, fallback int) (int, error) {
	return storage.ParsePositiveIntValue(raw, fieldName, fallback)
}

func parseNonNegativeIntValue(raw, fieldName string, fallback int) (int, error) {
	return storage.ParseNonNegativeIntValue(raw, fieldName, fallback)
}

func parseFloatValue(raw, fieldName string, fallback float64) (float64, error) {
	return storage.ParseFloatValue(raw, fieldName, fallback)
}

func parseDurationValue(raw, fieldName string, fallback time.Duration) (time.Duration, error) {
	return storage.ParseDurationValue(raw, fieldName, fallback)
}

func valueOrEnv(raw *string, envName, fallback string) string {
	return storage.ValueOrEnv(raw, envName, fallback)
}

func valueOrEnvWithEnv(raw *string, env envSnapshot, envName, fallback string) string {
	return storage.ValueOrEnvWithEnv(raw, env, envName, fallback)
}

func boolOrEnvWithEnv(raw *bool, env envSnapshot, envName string, fallback bool) (bool, error) {
	return storage.BoolOrEnvWithEnv(raw, env, envName, fallback)
}

func intOrEnvWithEnv(raw *int, fieldName string, env envSnapshot, envName string, fallback int) (int, error) {
	return storage.IntOrEnvWithEnv(raw, fieldName, env, envName, fallback)
}

func floatOrEnvWithEnv(raw *float64, fieldName string, env envSnapshot, envName string, fallback float64) (float64, error) {
	return storage.FloatOrEnvWithEnv(raw, fieldName, env, envName, fallback)
}

func durationOrEnvWithEnv(raw *string, fieldName string, env envSnapshot, envName string, fallback time.Duration) (time.Duration, error) {
	return storage.DurationOrEnvWithEnv(raw, fieldName, env, envName, fallback)
}

func headersOrEnvWithEnv(raw map[string]string, env envSnapshot) (map[string]string, error) {
	return storage.HeadersOrEnvWithEnv(raw, env)
}

func corsOriginsOrEnv(raw []string) []string {
	return storage.CORSOriginsOrEnv(raw)
}

func corsOriginsOrEnvWithEnv(raw []string, env envSnapshot) []string {
	return storage.CORSOriginsOrEnvWithEnv(raw, env)
}

func normalizeProviderHeaders(raw map[string]string) (map[string]string, error) {
	return storage.NormalizeProviderHeaders(raw)
}

func normalizeResponseMetadata(raw map[string]string) (map[string]string, error) {
	return storage.NormalizeResponseMetadata(raw)
}

func normalizeNamedHeaders(raw map[string]string, fieldName string) (map[string]string, error) {
	return storage.NormalizeNamedHeaders(raw, fieldName)
}

func normalizeOrigins(origins []string) []string {
	return storage.NormalizeOrigins(origins)
}

func parseOriginsCSV(raw string) []string {
	return storage.ParseOriginsCSV(raw)
}

func parseStringCSV(raw string) []string {
	return storage.ParseStringCSV(raw)
}

func resolveSessionsPath(fileCfg bridgeFileConfig, env envSnapshot) string {
	return storage.ResolveSessionsPath(fileCfg, env, defaultSessionsPath)
}

func resolveWebSearchTavilyAPIKey(fileCfg bridgeFileConfig, env envSnapshot) string {
	return storage.ResolveWebSearchTavilyAPIKey(fileCfg, env)
}

func resolveTasksPath(fileCfg bridgeFileConfig, env envSnapshot) string {
	return storage.ResolveTasksPath(fileCfg, env, defaultTasksPath)
}

func resolveScriptExecSandboxMemoryMB(fileCfg bridgeFileConfig, env envSnapshot) (int, error) {
	return storage.ResolveScriptExecSandboxMemoryMB(
		fileCfg,
		env,
		defaultScriptExecSandboxMemoryMB,
		maxScriptExecSandboxMemoryMB,
	)
}

func resolveNativeBinaryPath(fileCfg bridgeFileConfig, env envSnapshot) string {
	return storage.ResolveNativeBinaryPath(fileCfg, env)
}

func resolveNativeBinaryRoots(fileCfg bridgeFileConfig, env envSnapshot) []string {
	return storage.ResolveNativeBinaryRoots(fileCfg, env)
}

func resolveNativeBinaryCandidates(fileCfg bridgeFileConfig, env envSnapshot) []string {
	return storage.ResolveNativeBinaryCandidates(fileCfg, env)
}

func resolveCodexCLIPath(fileCfg bridgeFileConfig, env envSnapshot) string {
	return storage.ResolveCodexCLIPath(fileCfg, env)
}

func resolveNodeBinPath(fileCfg bridgeFileConfig, env envSnapshot) string {
	return storage.ResolveNodeBinPath(fileCfg, env)
}

func resolveNativeAllowedReadPaths(fileCfg bridgeFileConfig, env envSnapshot) []string {
	return storage.ResolveNativeAllowedReadPaths(fileCfg, env)
}

func resolveNativeAllowedWritePaths(fileCfg bridgeFileConfig, env envSnapshot) []string {
	return storage.ResolveNativeAllowedWritePaths(fileCfg, env)
}

func resolveProjectRoot(fileCfg bridgeFileConfig, env envSnapshot) string {
	return storage.ResolveProjectRoot(fileCfg, env)
}

func runtimeConfigFromEnv() (runtimeConfig, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return runtimeConfig{}, err
	}
	return resolveRuntimeConfig(fileCfg, currentEnv())
}

func resolveRuntimeConfig(fileCfg bridgeFileConfig, env envSnapshot) (runtimeConfig, error) {
	return configruntime.Resolve(fileCfg, env)
}

func runtimeFallbackFromEnv(env envSnapshot) (runtimeConfig, error) {
	return configruntime.FallbackFromEnv(env)
}

func resolveRuntimeConfigWithFallback(fileCfg bridgeFileConfig, fallback runtimeConfig) (runtimeConfig, error) {
	return configruntime.ResolveWithFallback(fileCfg, fallback)
}

func normalizeRuntimeConfig(raw runtimeConfig) runtimeConfig {
	return configruntime.Normalize(raw)
}

func cloneRuntimeConfig(raw runtimeConfig) runtimeConfig {
	return configruntime.Clone(raw)
}

func validateRuntimeForExecution(raw runtimeConfig) error {
	return configruntime.ValidateForExecution(raw)
}

func activeProviderLabel(raw runtimeConfig) string {
	return configruntime.ActiveProviderLabel(raw)
}

func inferProviderType(name, baseURL, model string) llm.Provider {
	return configruntime.InferProviderType(name, baseURL, model)
}

func defaultBaseURLForProvider(provider llm.Provider) string {
	return configruntime.DefaultBaseURLForProvider(provider)
}

func normalizeSessionTitleMode(raw string) (string, error) {
	return configruntime.NormalizeSessionTitleMode(raw)
}

func snapshotFromRuntimeConfig(runtime runtimeConfig) Snapshot {
	runtime = normalizeRuntimeConfig(runtime)
	return Snapshot{
		Provider:                       activeProviderLabel(runtime),
		ProviderType:                   string(runtime.Provider),
		BaseURL:                        runtime.BaseURL,
		Model:                          runtime.Model,
		ChatPath:                       runtime.ChatPath,
		ProjectRoot:                    runtime.ProjectRoot,
		MaxTurns:                       runtime.MaxTurns,
		TaskExecutionTimeoutMS:         runtime.TaskExecutionTimeoutMS,
		RelayDefaultStopPolicy:         runtime.RelayDefaultStopPolicy,
		RelayDefaultMaxRounds:          runtime.RelayDefaultMaxRounds,
		RelayDefaultExecutionTimeoutMS: runtime.RelayDefaultExecutionTimeoutMS,
		LLMCompletionRetryCount:        runtime.LLMCompletionRetryCount,
		LLMCompletionRetryIntervalMS:   runtime.LLMCompletionRetryIntervalMS,
		APIKeySet:                      runtime.APIKey != "",
		ModelSelectionEnabled:          runtime.ModelSelectionEnabled,
		SessionHumanLogFullEnabled:     runtime.SessionHumanLogFullEnabled,
		SessionSystemPromptVisible:     runtime.SessionSystemPromptVisible,
		AssistantMarkdownEnabled:       runtime.AssistantMarkdownEnabled,
		ToolCallCompactOutputEnabled:   runtime.ToolCallCompactOutputEnabled,
		MemoryModeEnabled:              runtime.MemoryModeEnabled,
		MicrocompactEnabled:            runtime.MicrocompactEnabled,
		SessionTitleMode:               runtime.SessionTitleMode,
		WebSearchTavilyURL:             runtime.WebSearchTavilyURL,
		WebSearchExaURL:                runtime.WebSearchExaURL,
		WebSearchTavilyAPIKeySet:       runtime.WebSearchTavilyAPIKey != "",
		WebSearchExaAPIKeySet:          runtime.WebSearchExaAPIKey != "",
	}
}

func normalizeProviderConfigs(raw map[string]providerFileConfig, model string) []providerConfig {
	return storage.NormalizeProviderConfigs(raw, model)
}

func providerConfigsToFileMap(providers []providerConfig) map[string]providerFileConfig {
	return storage.ProviderConfigsToFileMap(providers)
}

func normalizeProviderModels(models []string) []string {
	return providers.NormalizeModels(models)
}

func providerIndexByName(records []providerConfig, name string) int {
	return providers.IndexByName(records, name)
}

func normalizePositiveInt(value int) int {
	return providers.NormalizePositiveInt(value)
}

func normalizeModelTokenOverrides(raw map[string]int) map[string]int {
	return providers.NormalizeModelTokenOverrides(raw)
}

func cloneModelTokenOverrides(raw map[string]int) map[string]int {
	return providers.CloneModelTokenOverrides(raw)
}

func providerRecordsFromConfigs(raw []providerConfig) []ProviderRecord {
	if len(raw) == 0 {
		return nil
	}

	out := make([]ProviderRecord, 0, len(raw))
	for _, provider := range raw {
		out = append(out, ProviderRecord{
			Name:                       provider.Name,
			Type:                       provider.Type,
			BaseURL:                    provider.BaseURL,
			APIKey:                     cloneOptionalStringPointer(provider.APIKey),
			Models:                     append([]string(nil), provider.Models...),
			ContextWindowTokens:        provider.ContextWindowTokens,
			ResponseReserveTokens:      provider.ResponseReserveTokens,
			ModelContextWindowTokens:   cloneModelTokenOverrides(provider.ModelContextWindowTokens),
			ModelResponseReserveTokens: cloneModelTokenOverrides(provider.ModelResponseReserveTokens),
		})
	}
	return out
}

func providerConfigFromRecord(raw ProviderRecord) providerConfig {
	return providerConfig{
		Name:                       raw.Name,
		Type:                       raw.Type,
		BaseURL:                    raw.BaseURL,
		APIKey:                     cloneOptionalStringPointer(raw.APIKey),
		Models:                     append([]string(nil), raw.Models...),
		ContextWindowTokens:        raw.ContextWindowTokens,
		ResponseReserveTokens:      raw.ResponseReserveTokens,
		ModelContextWindowTokens:   cloneModelTokenOverrides(raw.ModelContextWindowTokens),
		ModelResponseReserveTokens: cloneModelTokenOverrides(raw.ModelResponseReserveTokens),
	}
}

func providerStateFromFileConfig(fileCfg bridgeFileConfig) providers.State {
	return providers.NewState(
		normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model)),
		stringValue(fileCfg.ActiveProvider),
		stringValue(fileCfg.Model),
	)
}

func applyProviderPatchToFileConfig(fileCfg *bridgeFileConfig, patch providers.Patch) {
	if fileCfg == nil {
		return
	}
	fileCfg.Providers = providerConfigsToFileMap(patch.Records)
	fileCfg.ActiveProvider = cloneOptionalStringPointer(patch.ActiveProvider)
	if patch.Model != nil {
		fileCfg.Model = cloneOptionalStringPointer(patch.Model)
	}
}

func providerRuntimeSnapshot(runtime runtimeConfig) providers.RuntimeSnapshot {
	return providers.RuntimeSnapshot{
		ProviderName: runtime.ProviderName,
		Provider:     runtime.Provider,
		APIKey:       runtime.APIKey,
		BaseURL:      runtime.BaseURL,
		Model:        runtime.Model,
	}
}
