package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"ghost-os/bridge/tools/internal/websearch"
)

const exaHighlightsMaxCharacters = 240
const (
	webSearchProviderTavily = "tavily"
	webSearchProviderExa    = "exa"
)

func (t *WebSearchTool) providersForSearch(providerHint string) ([]webSearchProvider, error) {
	if len(t.providers) > 0 {
		return providersForExplicitHint(t.providers, providerHint)
	}
	endpoint := strings.TrimSpace(t.endpoint)
	if endpoint != "" {
		return providersForExplicitHint([]webSearchProvider{{
			name:     "duckduckgo_html",
			endpoint: endpoint,
			format:   webSearchFormatDuckDuckGoHTML,
		}}, providerHint)
	}
	return t.defaultProviders(providerHint)
}

func (t *WebSearchTool) defaultProviders(providerHint string) ([]webSearchProvider, error) {
	hint := normalizeWebSearchProviderHint(providerHint)
	switch hint {
	case "":
		return t.providersForAutoSelection()
	case webSearchProviderTavily:
		return t.providersForRequestedAPIProvider(
			webSearchProviderTavily,
			t.config.TavilyAPIKey,
			resolveWebSearchProviderEndpoint(t.config.TavilyURL, defaultWebSearchTavily),
			webSearchFormatTavilyJSON,
		)
	case webSearchProviderExa:
		return t.providersForRequestedAPIProvider(
			webSearchProviderExa,
			t.config.ExaAPIKey,
			resolveWebSearchProviderEndpoint(t.config.ExaURL, defaultWebSearchExa),
			webSearchFormatExaJSON,
		)
	default:
		return nil, fmt.Errorf("unsupported web search provider %q", providerHint)
	}
}

func (t *WebSearchTool) providersForAutoSelection() ([]webSearchProvider, error) {
	tavilyAPIKey := strings.TrimSpace(t.config.TavilyAPIKey)
	exaAPIKey := strings.TrimSpace(t.config.ExaAPIKey)
	tavilyEndpoint := resolveWebSearchProviderEndpoint(t.config.TavilyURL, defaultWebSearchTavily)
	exaEndpoint := resolveWebSearchProviderEndpoint(t.config.ExaURL, defaultWebSearchExa)
	switch {
	case tavilyAPIKey != "" && exaAPIKey != "":
		return nil, fmt.Errorf("provider is required when both Tavily and Exa API keys are configured")
	case tavilyAPIKey != "":
		return []webSearchProvider{newAPIWebSearchProvider(webSearchProviderTavily, tavilyEndpoint, webSearchFormatTavilyJSON, tavilyAPIKey)}, nil
	case exaAPIKey != "":
		return []webSearchProvider{newAPIWebSearchProvider(webSearchProviderExa, exaEndpoint, webSearchFormatExaJSON, exaAPIKey)}, nil
	default:
		return defaultPublicWebSearchProviders(), nil
	}
}

func (t *WebSearchTool) providersForRequestedAPIProvider(
	providerName, apiKey, endpoint string,
	format webSearchFormat,
) ([]webSearchProvider, error) {
	trimmedAPIKey := strings.TrimSpace(apiKey)
	if trimmedAPIKey == "" {
		return nil, fmt.Errorf("web search provider %q requires its API key to be configured", providerName)
	}
	return []webSearchProvider{newAPIWebSearchProvider(providerName, endpoint, format, trimmedAPIKey)}, nil
}

func providersForExplicitHint(providers []webSearchProvider, providerHint string) ([]webSearchProvider, error) {
	hint := normalizeWebSearchProviderHint(providerHint)
	if hint == "" {
		return providers, nil
	}
	for _, provider := range providers {
		if strings.EqualFold(provider.name, hint) {
			return []webSearchProvider{provider}, nil
		}
	}
	return nil, fmt.Errorf("web search provider %q is not available", providerHint)
}

func normalizeWebSearchProviderHint(providerHint string) string {
	return strings.ToLower(strings.TrimSpace(providerHint))
}

func resolveWebSearchProviderEndpoint(customEndpoint, officialEndpoint string) string {
	trimmed := strings.TrimSpace(customEndpoint)
	if trimmed != "" {
		return trimmed
	}
	return officialEndpoint
}

func newAPIWebSearchProvider(name, endpoint string, format webSearchFormat, apiKey string) webSearchProvider {
	return webSearchProvider{
		name:     name,
		endpoint: endpoint,
		format:   format,
		apiKey:   apiKey,
	}
}

func defaultPublicWebSearchProviders() []webSearchProvider {
	return []webSearchProvider{
		{name: "duckduckgo_html", endpoint: defaultWebSearchEndpoint, format: webSearchFormatDuckDuckGoHTML},
		{name: "bing_rss", endpoint: defaultWebSearchBingRSS, format: webSearchFormatBingRSS},
	}
}

func acceptHeaderForProvider(format webSearchFormat) string {
	switch format {
	case webSearchFormatTavilyJSON, webSearchFormatExaJSON:
		return "application/json"
	case webSearchFormatBingRSS:
		return "application/rss+xml,application/xml,text/xml;q=0.9,*/*;q=0.8"
	default:
		return "text/html,application/xhtml+xml"
	}
}

func parseProviderResults(format webSearchFormat, body string, maxResults int) []websearch.Result {
	switch format {
	case webSearchFormatTavilyJSON:
		return websearch.ParseTavilyJSON(body, maxResults)
	case webSearchFormatExaJSON:
		return websearch.ParseExaJSON(body, maxResults)
	case webSearchFormatBingRSS:
		return websearch.ParseRSS(body, maxResults)
	default:
		return websearch.ParseHTML(body, maxResults)
	}
}

func buildWebSearchRequest(
	ctx context.Context,
	provider webSearchProvider,
	requestURL *url.URL,
	query string,
	maxResults int,
) (*http.Request, error) {
	switch provider.format {
	case webSearchFormatTavilyJSON:
		return buildTavilySearchRequest(ctx, provider, requestURL, query, maxResults)
	case webSearchFormatExaJSON:
		return buildExaSearchRequest(ctx, provider, requestURL, query, maxResults)
	default:
		return buildPublicSearchRequest(ctx, provider, requestURL, query)
	}
}

func buildTavilySearchRequest(
	ctx context.Context,
	provider webSearchProvider,
	requestURL *url.URL,
	query string,
	maxResults int,
) (*http.Request, error) {
	payload := map[string]any{
		"query":        query,
		"topic":        "general",
		"search_depth": "basic",
		"max_results":  maxResults,
	}
	req, err := newJSONWebSearchRequest(ctx, requestURL, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(provider.apiKey))
	return req, nil
}

func buildExaSearchRequest(
	ctx context.Context,
	provider webSearchProvider,
	requestURL *url.URL,
	query string,
	maxResults int,
) (*http.Request, error) {
	payload := map[string]any{
		"query":      query,
		"numResults": maxResults,
		"contents": map[string]any{
			"highlights": map[string]any{
				"query":         query,
				"maxCharacters": exaHighlightsMaxCharacters,
			},
		},
	}
	req, err := newJSONWebSearchRequest(ctx, requestURL, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", strings.TrimSpace(provider.apiKey))
	return req, nil
}

func newJSONWebSearchRequest(
	ctx context.Context,
	requestURL *url.URL,
	payload map[string]any,
) (*http.Request, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		requestURL.String(),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func buildPublicSearchRequest(
	ctx context.Context,
	provider webSearchProvider,
	requestURL *url.URL,
	query string,
) (*http.Request, error) {
	requestQuery := requestURL.Query()
	requestQuery.Set("q", query)
	if provider.format == webSearchFormatBingRSS {
		requestQuery.Set("format", "rss")
	}
	requestURL.RawQuery = requestQuery.Encode()
	return http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
}
