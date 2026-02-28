package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"ghost-os/bridge/llm"
)

// Config 描述 bridge 在运行时依赖的最小配置集合。
type Config struct {
	Provider           llm.Provider
	APIKey             string
	BaseURL            string
	Model              string
	ChatPath           string
	PromptsPath        string
	SessionsPath       string
	ProviderHeaders    map[string]string
	AnthropicVersion   string
	AnthropicMaxTokens int
	MaxTurns           int
}

type runtimeConfig struct {
	Provider llm.Provider
	APIKey   string
	BaseURL  string
	Model    string
	ChatPath string
}

const (
	defaultProvider           = llm.ProviderOpenAI
	defaultBaseURL            = "https://api.openai.com/v1"
	defaultModel              = "gpt-4o"
	defaultPromptsPath        = "prompts.yaml"
	defaultSessionsPath       = "~/.ghost-os/sessions"
	defaultAnthropicVersion   = "2023-06-01"
	defaultAnthropicMaxTokens = 1024
	defaultMaxTurns           = 20
)

// LoadConfig 从环境变量加载配置并做基础校验与归一化。
func LoadConfig() (Config, error) {
	runtime, err := runtimeConfigFromEnv()
	if err != nil {
		return Config{}, err
	}
	return loadConfigWithRuntime(runtime)
}

func loadConfigWithRuntime(runtime runtimeConfig) (Config, error) {
	runtime = normalizeRuntimeConfig(runtime)
	if err := validateRuntimeForExecution(runtime); err != nil {
		return Config{}, err
	}

	headers, err := parseProviderHeaders(os.Getenv("GHOST_PROVIDER_HEADERS"))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Provider:           runtime.Provider,
		APIKey:             runtime.APIKey,
		BaseURL:            runtime.BaseURL,
		Model:              runtime.Model,
		ChatPath:           runtime.ChatPath,
		PromptsPath:        getenvDefault("GHOST_PROMPTS_PATH", defaultPromptsPath),
		SessionsPath:       sessionsPathFromEnv(),
		ProviderHeaders:    headers,
		AnthropicVersion:   getenvDefault("GHOST_ANTHROPIC_VERSION", defaultAnthropicVersion),
		AnthropicMaxTokens: defaultAnthropicMaxTokens,
		MaxTurns:           defaultMaxTurns,
	}

	if raw := strings.TrimSpace(os.Getenv("GHOST_MAX_TURNS")); raw != "" {
		maxTurns, err := strconv.Atoi(raw)
		if err != nil || maxTurns <= 0 {
			return Config{}, fmt.Errorf("invalid GHOST_MAX_TURNS=%q, expected positive integer", raw)
		}
		cfg.MaxTurns = maxTurns
	}

	if raw := strings.TrimSpace(os.Getenv("GHOST_ANTHROPIC_MAX_TOKENS")); raw != "" {
		maxTokens, err := strconv.Atoi(raw)
		if err != nil || maxTokens <= 0 {
			return Config{}, fmt.Errorf("invalid GHOST_ANTHROPIC_MAX_TOKENS=%q, expected positive integer", raw)
		}
		cfg.AnthropicMaxTokens = maxTokens
	}

	return cfg, nil
}

func runtimeConfigFromEnv() (runtimeConfig, error) {
	provider := normalizeProvider(getenvDefault("GHOST_PROVIDER", string(defaultProvider)))
	if !provider.Valid() {
		return runtimeConfig{}, fmt.Errorf("invalid GHOST_PROVIDER=%q, expected one of: openai|anthropic|custom", provider)
	}

	return normalizeRuntimeConfig(runtimeConfig{
		Provider: provider,
		APIKey:   strings.TrimSpace(os.Getenv("GHOST_API_KEY")),
		BaseURL:  strings.TrimSpace(os.Getenv("GHOST_BASE_URL")),
		Model:    strings.TrimSpace(os.Getenv("GHOST_MODEL")),
		ChatPath: strings.TrimSpace(os.Getenv("GHOST_CHAT_PATH")),
	}), nil
}

func normalizeProvider(raw string) llm.Provider {
	return llm.Provider(strings.ToLower(strings.TrimSpace(raw)))
}

func normalizeRuntimeConfig(runtime runtimeConfig) runtimeConfig {
	out := runtime
	if out.Provider == "" {
		out.Provider = defaultProvider
	}
	out.APIKey = strings.TrimSpace(out.APIKey)
	if strings.TrimSpace(out.BaseURL) == "" {
		out.BaseURL = defaultBaseURL
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

func sessionsPathFromEnv() string {
	return getenvDefault("GHOST_SESSIONS_PATH", defaultSessionsPath)
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
