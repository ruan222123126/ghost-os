package llm

import "strings"

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderCustom    Provider = "custom"
)

func (p Provider) Normalized() Provider {
	switch Provider(strings.ToLower(strings.TrimSpace(string(p)))) {
	case ProviderOpenAI:
		return ProviderOpenAI
	case ProviderAnthropic:
		return ProviderAnthropic
	case ProviderCustom:
		return ProviderCustom
	default:
		return ""
	}
}

func (p Provider) Valid() bool {
	return p.Normalized() != ""
}
