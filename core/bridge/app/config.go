package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

// Config 描述 bridge 在运行时依赖的最小配置集合。
type Config struct {
	Provider                          llm.Provider
	APIKey                            string
	BaseURL                           string
	Model                             string
	NativePersistent                  bool
	WorkerModel                       string
	ChatPath                          string
	PromptsPath                       string
	SessionsPath                      string
	MemoryWarmPath                    string
	MemoryColdPath                    string
	MemoryAutoRecallEnabled           bool
	MemoryAutoRecallLimit             int
	MemoryWarmTTL                     time.Duration
	MemoryTemporalDecayEnabled        bool
	MemoryTemporalDecayHalfLife       time.Duration
	MemoryAnchorEnabled               bool
	MemoryAnchorMinWeight             float64
	MemoryEvolutionInterval           time.Duration
	MemoryEvolutionEnabled            bool
	MemoryEvolutionUseWorker          bool
	MemoryEvolutionBatchSize          int
	MemoryGraphEnabled                bool
	MemoryGraphPath                   string
	MemoryGraphExtractOnArchive       bool
	MemoryGraphExtractOnEvolve        bool
	MemoryGraphMaxHops                int
	MemoryGraphMaxHits                int
	MemoryGraphMinConfidence          float64
	MemoryGraphNamespace              string
	MemoryGraphDebugEnabled           bool
	MemoryDecisionEnabled             bool
	MemoryDecisionCaptureOnTurn       bool
	MemoryDecisionPath                string
	MemoryDecisionMaxHits             int
	MemoryDecisionMinConfidence       float64
	MemoryDecisionMinReuseScore       float64
	MemoryDecisionRecipeEnabled       bool
	MemoryDecisionRecipeInterval      time.Duration
	MemoryDecisionRecipeMinSupport    int
	MemoryDecisionDebugEnabled        bool
	MemoryDecisionSelectorHintEnabled bool
	ProviderHeaders                   map[string]string
	AnthropicVersion                  string
	AnthropicMaxTokens                int
	MaxTurns                          int
	WorkerMaxConcurrency              int
	WorkerMaxFiles                    int
	WorkerMaxFileChunks               int
	ToolSelectorEnabled               bool
	ToolSelectorMode                  string
	ToolSelectorModel                 string
	ToolSelectorTimeoutMS             int
	ToolSelectorConfidence            float64
	ToolSelectorShadow                bool
	ToolSelectorRecentMsgs            int
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
	defaultProvider                          = llm.ProviderOpenAI
	defaultBaseURL                           = "https://api.openai.com/v1"
	defaultAnthropicBaseURL                  = "https://api.anthropic.com"
	defaultModel                             = "gpt-4o"
	defaultPromptsPath                       = "prompts.yaml"
	defaultSessionsPath                      = "~/.ghost-os/sessions"
	defaultMemoryWarmPath                    = "~/.ghost-os/memory/warm.json"
	defaultMemoryColdPath                    = "~/.ghost-os/memory/cold"
	defaultMemoryGraphPath                   = "~/.ghost-os/memory/graph"
	defaultMemoryDecisionPath                = "~/.ghost-os/memory/decision"
	defaultMemoryAutoRecallLimit             = 5
	defaultMemoryWarmTTL                     = 24 * time.Hour
	defaultMemoryTemporalHalfLife            = 72 * time.Hour
	defaultMemoryAnchorMinWeight             = 0.65
	defaultMemoryEvolutionInterval           = time.Hour
	defaultMemoryEvolutionBatchSize          = 20
	defaultMemoryGraphMaxHops                = 1
	defaultMemoryGraphMaxHits                = 6
	defaultMemoryGraphNamespace              = "default"
	defaultMemoryGraphMinConfidence          = 0.72
	defaultMemoryDecisionMaxHits             = 4
	defaultMemoryDecisionCaptureOnTurn       = true
	defaultMemoryDecisionMinConfidence       = 0.75
	defaultMemoryDecisionMinReuseScore       = 0.70
	defaultMemoryDecisionRecipeInterval      = 6 * time.Hour
	defaultMemoryDecisionRecipeMinSupport    = 3
	defaultMemoryDecisionSelectorHintEnabled = true
	defaultAnthropicVersion                  = "2023-06-01"
	defaultAnthropicMaxTokens                = 1024
	defaultMaxTurns                          = 20
	defaultWorkerMaxConcurrency              = 4
	defaultWorkerMaxFiles                    = 20
	defaultWorkerMaxFileChunks               = 4
	defaultToolSelectorTimeoutMS             = 1500
	defaultToolSelectorConfidence            = 0.75
	defaultToolSelectorRecentMsgs            = 6
)

// LoadConfig 从环境变量加载配置并做基础校验与归一化。
func LoadConfig() (Config, error) {
	runtime, err := runtimeConfigFromEnv()
	if err != nil {
		return Config{}, err
	}
	return loadConfigWithRuntime(runtime)
}

// loadConfigWithRuntime 在 runtimeConfig 基础上补齐环境默认值与执行期约束。
func loadConfigWithRuntime(runtime runtimeConfig) (Config, error) {
	runtime = normalizeRuntimeConfig(runtime)
	if err := validateRuntimeForExecution(runtime); err != nil {
		return Config{}, err
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return Config{}, err
	}

	headers, err := headersOrEnv(fileCfg.ProviderHeaders)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Provider:                          runtime.Provider,
		APIKey:                            runtime.APIKey,
		BaseURL:                           runtime.BaseURL,
		Model:                             runtime.Model,
		NativePersistent:                  runtime.NativePersistent,
		WorkerModel:                       valueOrEnv(fileCfg.WorkerModel, "GHOST_WORKER_MODEL", ""),
		ChatPath:                          runtime.ChatPath,
		PromptsPath:                       valueOrEnv(fileCfg.PromptsPath, "GHOST_PROMPTS_PATH", defaultPromptsPath),
		SessionsPath:                      sessionsPathFromEnv(),
		MemoryWarmPath:                    memoryWarmPathFromEnv(),
		MemoryColdPath:                    memoryColdPathFromEnv(),
		MemoryAutoRecallEnabled:           memoryAutoRecallEnabledFromEnv(),
		MemoryAutoRecallLimit:             memoryAutoRecallLimitFromEnv(),
		MemoryWarmTTL:                     memoryWarmTTLFromEnv(),
		MemoryTemporalDecayEnabled:        memoryTemporalDecayEnabledFromEnv(),
		MemoryTemporalDecayHalfLife:       memoryTemporalDecayHalfLifeFromEnv(),
		MemoryAnchorEnabled:               memoryAnchorEnabledFromEnv(),
		MemoryAnchorMinWeight:             memoryAnchorMinWeightFromEnv(),
		MemoryEvolutionInterval:           memoryEvolutionIntervalFromEnv(),
		MemoryEvolutionEnabled:            memoryEvolutionEnabledFromEnv(),
		MemoryEvolutionUseWorker:          memoryEvolutionUseWorkerFromEnv(),
		MemoryEvolutionBatchSize:          memoryEvolutionBatchSizeFromEnv(),
		MemoryGraphEnabled:                memoryGraphEnabledFromEnv(),
		MemoryGraphPath:                   memoryGraphPathFromEnv(),
		MemoryGraphExtractOnArchive:       memoryGraphExtractOnArchiveFromEnv(),
		MemoryGraphExtractOnEvolve:        memoryGraphExtractOnEvolveFromEnv(),
		MemoryGraphMaxHops:                memoryGraphMaxHopsFromEnv(),
		MemoryGraphMaxHits:                memoryGraphMaxHitsFromEnv(),
		MemoryGraphMinConfidence:          memoryGraphMinConfidenceFromEnv(),
		MemoryGraphNamespace:              memoryGraphNamespaceFromEnv(),
		MemoryGraphDebugEnabled:           memoryGraphDebugEnabledFromEnv(),
		MemoryDecisionEnabled:             memoryDecisionEnabledFromEnv(),
		MemoryDecisionCaptureOnTurn:       memoryDecisionCaptureOnTurnFromEnv(),
		MemoryDecisionPath:                memoryDecisionPathFromEnv(),
		MemoryDecisionMaxHits:             memoryDecisionMaxHitsFromEnv(),
		MemoryDecisionMinConfidence:       memoryDecisionMinConfidenceFromEnv(),
		MemoryDecisionMinReuseScore:       memoryDecisionMinReuseScoreFromEnv(),
		MemoryDecisionRecipeEnabled:       memoryDecisionRecipeEnabledFromEnv(),
		MemoryDecisionRecipeInterval:      memoryDecisionRecipeIntervalFromEnv(),
		MemoryDecisionRecipeMinSupport:    memoryDecisionRecipeMinSupportFromEnv(),
		MemoryDecisionDebugEnabled:        memoryDecisionDebugEnabledFromEnv(),
		MemoryDecisionSelectorHintEnabled: memoryDecisionSelectorHintEnabledFromEnv(),
		ProviderHeaders:                   headers,
		AnthropicVersion:                  valueOrEnv(fileCfg.AnthropicVersion, "GHOST_ANTHROPIC_VERSION", defaultAnthropicVersion),
		AnthropicMaxTokens:                intOrEnv(fileCfg.AnthropicMaxTokens, "GHOST_ANTHROPIC_MAX_TOKENS", defaultAnthropicMaxTokens),
		MaxTurns:                          intOrEnv(fileCfg.MaxTurns, "GHOST_MAX_TURNS", defaultMaxTurns),
		WorkerMaxConcurrency:              intOrEnv(fileCfg.WorkerMaxConcurrency, "GHOST_WORKER_MAX_CONCURRENCY", defaultWorkerMaxConcurrency),
		WorkerMaxFiles:                    intOrEnv(fileCfg.WorkerMaxFiles, "GHOST_WORKER_MAX_FILES", defaultWorkerMaxFiles),
		WorkerMaxFileChunks:               intOrEnv(fileCfg.WorkerMaxFileChunks, "GHOST_WORKER_MAX_FILE_CHUNKS", defaultWorkerMaxFileChunks),
		ToolSelectorEnabled:               boolOrEnv(fileCfg.ToolSelectorEnabled, "GHOST_TOOL_SELECTOR_ENABLED", false),
		ToolSelectorMode:                  strings.ToLower(valueOrEnv(fileCfg.ToolSelectorMode, "GHOST_TOOL_SELECTOR_MODE", "llm")),
		ToolSelectorModel:                 valueOrEnv(fileCfg.ToolSelectorModel, "GHOST_TOOL_SELECTOR_MODEL", ""),
		ToolSelectorTimeoutMS:             intOrEnv(fileCfg.ToolSelectorTimeoutMS, "GHOST_TOOL_SELECTOR_TIMEOUT_MS", defaultToolSelectorTimeoutMS),
		ToolSelectorConfidence:            floatOrEnv(fileCfg.ToolSelectorConfidence, "GHOST_TOOL_SELECTOR_CONFIDENCE", defaultToolSelectorConfidence),
		ToolSelectorShadow:                boolOrEnv(fileCfg.ToolSelectorShadow, "GHOST_TOOL_SELECTOR_SHADOW", false),
		ToolSelectorRecentMsgs:            intOrEnv(fileCfg.ToolSelectorRecentMsgs, "GHOST_TOOL_SELECTOR_RECENT_MESSAGES", defaultToolSelectorRecentMsgs),
	}

	return cfg, nil
}

// runtimeConfigFromEnv 优先读取 ~/.ghost-os/config.toml 中的运行态字段，缺省时回退环境变量。
func runtimeConfigFromEnv() (runtimeConfig, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return runtimeConfig{}, err
	}
	return runtimeConfigFromFileConfig(fileCfg)
}

func runtimeConfigFromFileConfig(fileCfg bridgeFileConfig) (runtimeConfig, error) {
	fileCfg = normalizeBridgeFileConfigForWrite(fileCfg)
	providers := normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))
	if len(providers) > 0 {
		activeName := strings.TrimSpace(valueOrEnv(fileCfg.ActiveProvider, "GHOST_PROVIDER", ""))
		activeIndex := providerIndexByName(providers, activeName)
		if activeIndex < 0 {
			activeIndex = 0
		}
		active := providers[activeIndex]
		model := valueOrEnv(fileCfg.Model, "GHOST_MODEL", "")
		return normalizeRuntimeConfig(runtimeConfig{
			ProviderName:     active.Name,
			Provider:         active.Type.Normalized(),
			APIKey:           valueOrEnv(active.APIKey, "GHOST_API_KEY", ""),
			BaseURL:          providerBaseURL(active, active.Type.Normalized()),
			Model:            model,
			ChatPath:         valueOrEnv(fileCfg.ChatPath, "GHOST_CHAT_PATH", ""),
			NativePersistent: resolveNativePersistent(fileCfg.NativePersistent),
		}), nil
	}

	providerName := strings.TrimSpace(getenvDefault("GHOST_PROVIDER", string(defaultProvider)))
	providerType := inferProviderType(providerName, getenvDefault("GHOST_BASE_URL", ""), valueOrEnv(fileCfg.Model, "GHOST_MODEL", ""))
	return normalizeRuntimeConfig(runtimeConfig{
		ProviderName:     providerName,
		Provider:         providerType,
		APIKey:           getenvDefault("GHOST_API_KEY", ""),
		BaseURL:          getenvDefault("GHOST_BASE_URL", ""),
		Model:            valueOrEnv(fileCfg.Model, "GHOST_MODEL", ""),
		ChatPath:         valueOrEnv(fileCfg.ChatPath, "GHOST_CHAT_PATH", ""),
		NativePersistent: resolveNativePersistent(fileCfg.NativePersistent),
	}), nil
}

func providerBaseURL(provider providerConfig, providerType llm.Provider) string {
	configured := strings.TrimSpace(provider.BaseURL)
	if configured != "" {
		return configured
	}
	return getenvDefault("GHOST_BASE_URL", defaultBaseURLForProvider(providerType))
}

// normalizeProvider 统一 provider 大小写与空白字符。
func normalizeProvider(raw string) llm.Provider {
	return llm.Provider(strings.ToLower(strings.TrimSpace(raw)))
}

func inferProviderType(name, baseURL, model string) llm.Provider {
	if normalized := normalizeProvider(name).Normalized(); normalized != "" {
		return normalized
	}

	lowerBaseURL := strings.ToLower(strings.TrimSpace(baseURL))
	lowerModel := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(lowerModel, "claude"), strings.Contains(lowerBaseURL, "anthropic.com"):
		return llm.ProviderAnthropic
	case strings.Contains(lowerBaseURL, "openai.com"):
		return llm.ProviderOpenAI
	default:
		return llm.ProviderCustom
	}
}

func defaultBaseURLForProvider(provider llm.Provider) string {
	if provider.Normalized() == llm.ProviderAnthropic {
		return defaultAnthropicBaseURL
	}
	return defaultBaseURL
}

func activeProviderLabel(runtime runtimeConfig) string {
	if value := strings.TrimSpace(runtime.ProviderName); value != "" {
		return value
	}
	return string(runtime.Provider)
}

// normalizeRuntimeConfig 回填默认值并清理字符串字段。
func normalizeRuntimeConfig(runtime runtimeConfig) runtimeConfig {
	out := runtime
	out.ProviderName = strings.TrimSpace(out.ProviderName)
	if out.Provider == "" {
		out.Provider = inferProviderType(out.ProviderName, out.BaseURL, out.Model)
	}
	if out.Provider == "" {
		out.Provider = defaultProvider
	}
	if out.ProviderName == "" {
		out.ProviderName = string(out.Provider)
	}
	out.APIKey = strings.TrimSpace(out.APIKey)
	if strings.TrimSpace(out.BaseURL) == "" {
		out.BaseURL = defaultBaseURLForProvider(out.Provider)
	} else {
		out.BaseURL = strings.TrimSpace(out.BaseURL)
	}
	if strings.TrimSpace(out.Model) == "" {
		out.Model = defaultModel
	} else {
		out.Model = strings.TrimSpace(out.Model)
	}
	out.ChatPath = strings.TrimSpace(out.ChatPath)
	return out
}

// validateRuntimeForExecution 校验当前 provider 的最小执行前置条件。
func validateRuntimeForExecution(runtime runtimeConfig) error {
	switch runtime.Provider {
	case llm.ProviderOpenAI, llm.ProviderAnthropic:
		if runtime.APIKey == "" {
			return fmt.Errorf("GHOST_API_KEY is required for provider %q", runtime.Provider)
		}
	case llm.ProviderCustom:
		// custom provider 默认允许不传 API Key。
	default:
		return fmt.Errorf("invalid GHOST_PROVIDER=%q, expected one of: openai|anthropic|custom", runtime.Provider)
	}
	return nil
}

// getenvDefault 在环境变量为空时回落默认值。
func getenvDefault(name, fallback string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	return v
}

// sessionsPathFromEnv 返回会话持久化目录。
func sessionsPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_SESSIONS_PATH", defaultSessionsPath)
	}
	return valueOrEnv(fileCfg.SessionsPath, "GHOST_SESSIONS_PATH", defaultSessionsPath)
}

// memoryWarmPathFromEnv 返回 warm memory 持久化文件路径。
func memoryWarmPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_MEMORY_WARM_PATH", defaultMemoryWarmPath)
	}
	return valueOrEnv(fileCfg.MemoryWarmPath, "GHOST_MEMORY_WARM_PATH", defaultMemoryWarmPath)
}

// memoryColdPathFromEnv 返回 cold memory 根目录路径。
func memoryColdPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_MEMORY_COLD_PATH", defaultMemoryColdPath)
	}
	return valueOrEnv(fileCfg.MemoryColdPath, "GHOST_MEMORY_COLD_PATH", defaultMemoryColdPath)
}

// memoryAutoRecallEnabledFromEnv 控制 warm 自动召回是否启用。
func memoryAutoRecallEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_AUTO_RECALL_ENABLED", true)
	}
	return boolOrEnv(fileCfg.MemoryAutoRecallEnabled, "GHOST_MEMORY_AUTO_RECALL_ENABLED", true)
}

// memoryAutoRecallLimitFromEnv 返回自动召回注入条目上限。
func memoryAutoRecallLimitFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_MEMORY_AUTO_RECALL_LIMIT", defaultMemoryAutoRecallLimit)
	}
	return intOrEnv(fileCfg.MemoryAutoRecallLimit, "GHOST_MEMORY_AUTO_RECALL_LIMIT", defaultMemoryAutoRecallLimit)
}

// memoryWarmTTLFromEnv 返回 warm 层默认 TTL。
func memoryWarmTTLFromEnv() time.Duration {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseDurationEnv("GHOST_MEMORY_WARM_TTL", defaultMemoryWarmTTL)
	}
	return durationOrEnv(fileCfg.MemoryWarmTTL, "GHOST_MEMORY_WARM_TTL", defaultMemoryWarmTTL)
}

func memoryTemporalDecayEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_TEMPORAL_DECAY_ENABLED", true)
	}
	return boolOrEnv(fileCfg.MemoryTemporalDecayEnabled, "GHOST_MEMORY_TEMPORAL_DECAY_ENABLED", true)
}

func memoryTemporalDecayHalfLifeFromEnv() time.Duration {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseDurationEnv("GHOST_MEMORY_TEMPORAL_DECAY_HALF_LIFE", defaultMemoryTemporalHalfLife)
	}
	return durationOrEnv(fileCfg.MemoryTemporalDecayHalfLife, "GHOST_MEMORY_TEMPORAL_DECAY_HALF_LIFE", defaultMemoryTemporalHalfLife)
}

func memoryAnchorEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_ANCHOR_ENABLED", true)
	}
	return boolOrEnv(fileCfg.MemoryAnchorEnabled, "GHOST_MEMORY_ANCHOR_ENABLED", true)
}

func memoryAnchorMinWeightFromEnv() float64 {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseFloatEnv("GHOST_MEMORY_ANCHOR_MIN_WEIGHT", defaultMemoryAnchorMinWeight)
	}
	return floatOrEnv(fileCfg.MemoryAnchorMinWeight, "GHOST_MEMORY_ANCHOR_MIN_WEIGHT", defaultMemoryAnchorMinWeight)
}

// memoryEvolutionIntervalFromEnv 返回后台演化间隔。
func memoryEvolutionIntervalFromEnv() time.Duration {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseDurationEnv("GHOST_MEMORY_EVOLUTION_INTERVAL", defaultMemoryEvolutionInterval)
	}
	return durationOrEnv(fileCfg.MemoryEvolutionInterval, "GHOST_MEMORY_EVOLUTION_INTERVAL", defaultMemoryEvolutionInterval)
}

// memoryEvolutionEnabledFromEnv 控制后台演化协程是否启用。
func memoryEvolutionEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_EVOLUTION_ENABLED", false)
	}
	return boolOrEnv(fileCfg.MemoryEvolutionEnabled, "GHOST_MEMORY_EVOLUTION_ENABLED", false)
}

func memoryEvolutionUseWorkerFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_EVOLUTION_USE_WORKER", true)
	}
	return boolOrEnv(fileCfg.MemoryEvolutionUseWorker, "GHOST_MEMORY_EVOLUTION_USE_WORKER", true)
}

func memoryEvolutionBatchSizeFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_MEMORY_EVOLUTION_BATCH_SIZE", defaultMemoryEvolutionBatchSize)
	}
	return intOrEnv(fileCfg.MemoryEvolutionBatchSize, "GHOST_MEMORY_EVOLUTION_BATCH_SIZE", defaultMemoryEvolutionBatchSize)
}

func memoryGraphEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_GRAPH_ENABLED", false)
	}
	return boolOrEnv(fileCfg.MemoryGraphEnabled, "GHOST_MEMORY_GRAPH_ENABLED", false)
}

func memoryGraphPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_MEMORY_GRAPH_PATH", defaultMemoryGraphPath)
	}
	return valueOrEnv(fileCfg.MemoryGraphPath, "GHOST_MEMORY_GRAPH_PATH", defaultMemoryGraphPath)
}

func memoryGraphExtractOnArchiveFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_GRAPH_EXTRACT_ON_ARCHIVE", true)
	}
	return boolOrEnv(fileCfg.MemoryGraphExtractOnArchive, "GHOST_MEMORY_GRAPH_EXTRACT_ON_ARCHIVE", true)
}

func memoryGraphExtractOnEvolveFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_GRAPH_EXTRACT_ON_EVOLVE", true)
	}
	return boolOrEnv(fileCfg.MemoryGraphExtractOnEvolve, "GHOST_MEMORY_GRAPH_EXTRACT_ON_EVOLVE", true)
}

func memoryGraphMaxHopsFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_MEMORY_GRAPH_MAX_HOPS", defaultMemoryGraphMaxHops)
	}
	return intOrEnv(fileCfg.MemoryGraphMaxHops, "GHOST_MEMORY_GRAPH_MAX_HOPS", defaultMemoryGraphMaxHops)
}

func memoryGraphMaxHitsFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_MEMORY_GRAPH_MAX_HITS", defaultMemoryGraphMaxHits)
	}
	return intOrEnv(fileCfg.MemoryGraphMaxHits, "GHOST_MEMORY_GRAPH_MAX_HITS", defaultMemoryGraphMaxHits)
}

func memoryGraphMinConfidenceFromEnv() float64 {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseFloatEnv("GHOST_MEMORY_GRAPH_MIN_CONFIDENCE", defaultMemoryGraphMinConfidence)
	}
	return floatOrEnv(fileCfg.MemoryGraphMinConfidence, "GHOST_MEMORY_GRAPH_MIN_CONFIDENCE", defaultMemoryGraphMinConfidence)
}

func memoryGraphNamespaceFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_MEMORY_GRAPH_NAMESPACE", defaultMemoryGraphNamespace)
	}
	return valueOrEnv(fileCfg.MemoryGraphNamespace, "GHOST_MEMORY_GRAPH_NAMESPACE", defaultMemoryGraphNamespace)
}

func memoryGraphDebugEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_GRAPH_DEBUG_ENABLED", false)
	}
	return boolOrEnv(fileCfg.MemoryGraphDebugEnabled, "GHOST_MEMORY_GRAPH_DEBUG_ENABLED", false)
}

func memoryDecisionEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_DECISION_ENABLED", false)
	}
	return boolOrEnv(fileCfg.MemoryDecisionEnabled, "GHOST_MEMORY_DECISION_ENABLED", false)
}

func memoryDecisionPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_MEMORY_DECISION_PATH", defaultMemoryDecisionPath)
	}
	return valueOrEnv(fileCfg.MemoryDecisionPath, "GHOST_MEMORY_DECISION_PATH", defaultMemoryDecisionPath)
}

func memoryDecisionCaptureOnTurnFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_DECISION_CAPTURE_ON_TURN", defaultMemoryDecisionCaptureOnTurn)
	}
	return boolOrEnv(fileCfg.MemoryDecisionCaptureOnTurn, "GHOST_MEMORY_DECISION_CAPTURE_ON_TURN", defaultMemoryDecisionCaptureOnTurn)
}

func memoryDecisionMaxHitsFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_MEMORY_DECISION_MAX_HITS", defaultMemoryDecisionMaxHits)
	}
	return intOrEnv(fileCfg.MemoryDecisionMaxHits, "GHOST_MEMORY_DECISION_MAX_HITS", defaultMemoryDecisionMaxHits)
}

func memoryDecisionMinConfidenceFromEnv() float64 {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseFloatEnv("GHOST_MEMORY_DECISION_MIN_CONFIDENCE", defaultMemoryDecisionMinConfidence)
	}
	return floatOrEnv(fileCfg.MemoryDecisionMinConfidence, "GHOST_MEMORY_DECISION_MIN_CONFIDENCE", defaultMemoryDecisionMinConfidence)
}

func memoryDecisionMinReuseScoreFromEnv() float64 {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseFloatEnv("GHOST_MEMORY_DECISION_MIN_REUSE_SCORE", defaultMemoryDecisionMinReuseScore)
	}
	return floatOrEnv(fileCfg.MemoryDecisionMinReuseScore, "GHOST_MEMORY_DECISION_MIN_REUSE_SCORE", defaultMemoryDecisionMinReuseScore)
}

func memoryDecisionRecipeEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_DECISION_RECIPE_ENABLED", true)
	}
	return boolOrEnv(fileCfg.MemoryDecisionRecipeEnabled, "GHOST_MEMORY_DECISION_RECIPE_ENABLED", true)
}

func memoryDecisionRecipeIntervalFromEnv() time.Duration {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseDurationEnv("GHOST_MEMORY_DECISION_RECIPE_INTERVAL", defaultMemoryDecisionRecipeInterval)
	}
	return durationOrEnv(fileCfg.MemoryDecisionRecipeInterval, "GHOST_MEMORY_DECISION_RECIPE_INTERVAL", defaultMemoryDecisionRecipeInterval)
}

func memoryDecisionRecipeMinSupportFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_MEMORY_DECISION_RECIPE_MIN_SUPPORT", defaultMemoryDecisionRecipeMinSupport)
	}
	return intOrEnv(fileCfg.MemoryDecisionRecipeMinSupport, "GHOST_MEMORY_DECISION_RECIPE_MIN_SUPPORT", defaultMemoryDecisionRecipeMinSupport)
}

func memoryDecisionDebugEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_DECISION_DEBUG_ENABLED", false)
	}
	return boolOrEnv(fileCfg.MemoryDecisionDebugEnabled, "GHOST_MEMORY_DECISION_DEBUG_ENABLED", false)
}

func memoryDecisionSelectorHintEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_MEMORY_DECISION_SELECTOR_HINT_ENABLED", defaultMemoryDecisionSelectorHintEnabled)
	}
	return boolOrEnv(fileCfg.MemoryDecisionSelectorHintEnabled, "GHOST_MEMORY_DECISION_SELECTOR_HINT_ENABLED", defaultMemoryDecisionSelectorHintEnabled)
}

// parseProviderHeaders 解析自定义 Header JSON，并做 key 空值防护。
func parseProviderHeaders(raw string) (map[string]string, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, nil
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, fmt.Errorf("invalid GHOST_PROVIDER_HEADERS: expected JSON object of string values: %w", err)
	}

	out := make(map[string]string, len(parsed))
	for key, value := range parsed {
		k := strings.TrimSpace(key)
		if k == "" {
			return nil, fmt.Errorf("invalid GHOST_PROVIDER_HEADERS: header key cannot be empty")
		}
		out[k] = strings.TrimSpace(value)
	}

	return out, nil
}

func nativePersistentEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err == nil {
		return resolveNativePersistent(fileCfg.NativePersistent)
	}
	return resolveNativePersistent(nil)
}

func resolveNativePersistent(raw *bool) bool {
	if raw != nil {
		return *raw
	}
	for _, name := range []string{"GHOST_NATIVE_PERSISTENT", "GHOST_NATIVE_PERSISTENT_ENABLED"} {
		rawValue := strings.TrimSpace(os.Getenv(name))
		if rawValue == "" {
			continue
		}
		enabled, err := strconv.ParseBool(rawValue)
		if err != nil {
			return false
		}
		return enabled
	}
	return false
}

func parseBoolEnv(name string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func parsePositiveIntEnv(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func parseFloatEnv(name string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value < 0 || value > 1 {
		return fallback
	}
	return value
}

func parseDurationEnv(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
