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
	ProviderHeaders    map[string]string
	AnthropicVersion   string
	AnthropicMaxTokens int
	MaxTurns           int
}

// LoadConfig 从环境变量加载配置并做基础校验与归一化。
func LoadConfig() (Config, error) {
	provider := llm.Provider(strings.ToLower(getenvDefault("GHOST_PROVIDER", "openai")))
	if !provider.Valid() {
		return Config{}, fmt.Errorf("invalid GHOST_PROVIDER=%q, expected one of: openai|anthropic|custom", provider)
	}

	headers, err := parseProviderHeaders(os.Getenv("GHOST_PROVIDER_HEADERS"))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		// 默认保持 OpenAI 兼容路径，避免本地最小链路启动失败。
		Provider:           provider,
		APIKey:             strings.TrimSpace(os.Getenv("GHOST_API_KEY")),
		BaseURL:            getenvDefault("GHOST_BASE_URL", "https://api.openai.com/v1"),
		Model:              getenvDefault("GHOST_MODEL", "gpt-4o"),
		ChatPath:           strings.TrimSpace(os.Getenv("GHOST_CHAT_PATH")),
		ProviderHeaders:    headers,
		AnthropicVersion:   getenvDefault("GHOST_ANTHROPIC_VERSION", "2023-06-01"),
		AnthropicMaxTokens: 1024,
		MaxTurns:           20,
	}

	switch cfg.Provider {
	case llm.ProviderOpenAI, llm.ProviderAnthropic:
		// 官方 provider 默认要求 API Key。
		if cfg.APIKey == "" {
			return Config{}, fmt.Errorf("GHOST_API_KEY is required for provider %q", cfg.Provider)
		}
	case llm.ProviderCustom:
		// custom provider 默认允许不传 API Key。
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

// getenvDefault 在环境变量为空时回落默认值。
func getenvDefault(name, fallback string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	return v
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
