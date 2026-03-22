package webrooter

import (
	"net/http"
	"strings"
	"time"
)

const (
	PinnedVersion      = "0.2.4"
	PinnedCommit       = "77b0104137ed4b173ba19e6b5e3a23817ec5e198"
	DefaultHTTPTimeout = 90 * time.Second
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Config struct {
	BaseURL    string
	APIToken   string
	Timeout    time.Duration
	HTTPClient HTTPClient
}

type Client struct {
	baseURL    string
	apiToken   string
	httpClient HTTPClient
}

type Request struct {
	Action  string
	Path    string
	Body    map[string]any
	TraceID string
}

type Result struct {
	Payload        map[string]any
	Citations      []any
	ReferencesText string
}

type versionResponse struct {
	Version string `json:"version"`
}

func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultHTTPTimeout
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}

	return &Client{
		baseURL:    normalizeBaseURL(cfg.BaseURL),
		apiToken:   strings.TrimSpace(cfg.APIToken),
		httpClient: httpClient,
	}
}

func normalizeBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "http://127.0.0.1:8765"
	}
	return strings.TrimRight(trimmed, "/")
}
