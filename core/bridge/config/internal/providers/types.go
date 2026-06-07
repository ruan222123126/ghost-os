package providers

import (
	"errors"

	"ghost-os/bridge/llm"
)

const (
	defaultProvider         = llm.ProviderOpenAI
	defaultBaseURL          = "https://api.openai.com/v1"
	defaultAnthropicBaseURL = "https://api.anthropic.com"
)

var (
	ErrNameRequired    = errors.New("provider name is required")
	ErrTypeInvalid     = errors.New("provider type must be one of: openai|anthropic|custom|codex")
	ErrBaseURLRequired = errors.New("provider base_url is required")
	ErrNotFound        = errors.New("provider not found")
	ErrExists          = errors.New("provider already exists")
)

type Record struct {
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

type FileRecord struct {
	Type                       llm.Provider
	BaseURL                    string
	APIKey                     *string
	Models                     []string
	ContextWindowTokens        int
	ResponseReserveTokens      int
	ModelContextWindowTokens   map[string]int
	ModelResponseReserveTokens map[string]int
}

type RuntimeSnapshot struct {
	ProviderName string
	Provider     llm.Provider
	APIKey       string
	BaseURL      string
	Model        string
}

type State struct {
	Records        []Record
	ActiveProvider string
	Model          string
}

type Patch struct {
	Records        []Record
	ActiveProvider *string
	Model          *string
}

type RuntimePatchRequest struct {
	Provider *string
	APIKey   *string
	BaseURL  *string
}
