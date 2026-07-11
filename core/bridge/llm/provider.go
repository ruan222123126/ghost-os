package llm

import "strings"

// Provider 标识底层模型服务提供方。
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderCustom    Provider = "custom"
	ProviderCodex     Provider = "codex"
)

// Normalized 统一 provider 输入大小写/空白，未知值返回空串。
func (p Provider) Normalized() Provider {
	switch Provider(strings.ToLower(strings.TrimSpace(string(p)))) {
	case ProviderOpenAI:
		return ProviderOpenAI
	case ProviderAnthropic:
		return ProviderAnthropic
	case ProviderCustom:
		return ProviderCustom
	case ProviderCodex:
		return ProviderCodex
	default:
		return ""
	}
}
