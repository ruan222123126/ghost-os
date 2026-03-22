package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	webRooterActionInternetSearch = "internet_search"
	webRooterActionResearch       = "research"
	webRooterActionAcademicSearch = "academic_search"
	webRooterActionSiteSearch     = "site_search"
	webRooterActionFetch          = "fetch"
	webRooterActionExtract        = "extract"
)

type webRooterDecodedArgs struct {
	action string
	path   string
	body   map[string]any
}

type webRooterArgsPayload struct {
	Action string          `json:"action"`
	Params json.RawMessage `json:"params"`
}

type webRooterInternetSearchParams struct {
	Query      *string `json:"query"`
	NumResults *int    `json:"num_results"`
	AutoCrawl  *bool   `json:"auto_crawl"`
}

type webRooterResearchParams struct {
	Topic    *string `json:"topic"`
	MaxPages *int    `json:"max_pages"`
}

type webRooterAcademicSearchParams struct {
	Query          *string `json:"query"`
	NumResults     *int    `json:"num_results"`
	IncludeCode    *bool   `json:"include_code"`
	FetchAbstracts *bool   `json:"fetch_abstracts"`
}

type webRooterSiteSearchParams struct {
	URL        *string `json:"url"`
	Query      *string `json:"query"`
	UseBrowser *bool   `json:"use_browser"`
}

type webRooterFetchParams struct {
	URL        *string `json:"url"`
	UseBrowser *bool   `json:"use_browser"`
}

type webRooterExtractParams struct {
	URL    *string `json:"url"`
	Target *string `json:"target"`
}

func decodeWebRooterArgs(argsJSON json.RawMessage) (webRooterDecodedArgs, error) {
	var payload webRooterArgsPayload
	if err := decodeWebRooterStrictJSON(argsJSON, &payload); err != nil {
		return webRooterDecodedArgs{}, fmt.Errorf("decode web_rooter args: %w", err)
	}

	action := strings.TrimSpace(payload.Action)
	if action == "" {
		return webRooterDecodedArgs{}, fmt.Errorf("web_rooter action is required")
	}
	if len(bytes.TrimSpace(payload.Params)) == 0 {
		return webRooterDecodedArgs{}, fmt.Errorf("web_rooter params is required")
	}

	switch action {
	case webRooterActionInternetSearch:
		return decodeWebRooterInternetSearch(payload.Params)
	case webRooterActionResearch:
		return decodeWebRooterResearch(payload.Params)
	case webRooterActionAcademicSearch:
		return decodeWebRooterAcademicSearch(payload.Params)
	case webRooterActionSiteSearch:
		return decodeWebRooterSiteSearch(payload.Params)
	case webRooterActionFetch:
		return decodeWebRooterFetch(payload.Params)
	case webRooterActionExtract:
		return decodeWebRooterExtract(payload.Params)
	default:
		return webRooterDecodedArgs{}, fmt.Errorf("unsupported web_rooter action %q", action)
	}
}

func decodeWebRooterInternetSearch(raw json.RawMessage) (webRooterDecodedArgs, error) {
	var params webRooterInternetSearchParams
	if err := decodeWebRooterStrictJSON(raw, &params); err != nil {
		return webRooterDecodedArgs{}, fmt.Errorf("decode internet_search params: %w", err)
	}
	query, err := requireWebRooterString("internet_search.query", params.Query)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	numResults, err := requireWebRooterPositiveInt("internet_search.num_results", params.NumResults)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	autoCrawl, err := requireWebRooterBool("internet_search.auto_crawl", params.AutoCrawl)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	return webRooterDecodedArgs{action: webRooterActionInternetSearch, path: "/search/internet", body: map[string]any{"query": query, "num_results": numResults, "auto_crawl": autoCrawl}}, nil
}

func decodeWebRooterResearch(raw json.RawMessage) (webRooterDecodedArgs, error) {
	var params webRooterResearchParams
	if err := decodeWebRooterStrictJSON(raw, &params); err != nil {
		return webRooterDecodedArgs{}, fmt.Errorf("decode research params: %w", err)
	}
	topic, err := requireWebRooterString("research.topic", params.Topic)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	maxPages, err := requireWebRooterPositiveInt("research.max_pages", params.MaxPages)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	return webRooterDecodedArgs{action: webRooterActionResearch, path: "/research", body: map[string]any{"topic": topic, "max_pages": maxPages}}, nil
}

func decodeWebRooterAcademicSearch(raw json.RawMessage) (webRooterDecodedArgs, error) {
	var params webRooterAcademicSearchParams
	if err := decodeWebRooterStrictJSON(raw, &params); err != nil {
		return webRooterDecodedArgs{}, fmt.Errorf("decode academic_search params: %w", err)
	}
	query, err := requireWebRooterString("academic_search.query", params.Query)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	numResults, err := requireWebRooterPositiveInt("academic_search.num_results", params.NumResults)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	includeCode, err := requireWebRooterBool("academic_search.include_code", params.IncludeCode)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	fetchAbstracts, err := requireWebRooterBool("academic_search.fetch_abstracts", params.FetchAbstracts)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	return webRooterDecodedArgs{action: webRooterActionAcademicSearch, path: "/search/academic", body: map[string]any{"query": query, "num_results": numResults, "include_code": includeCode, "fetch_abstracts": fetchAbstracts}}, nil
}

func decodeWebRooterSiteSearch(raw json.RawMessage) (webRooterDecodedArgs, error) {
	var params webRooterSiteSearchParams
	if err := decodeWebRooterStrictJSON(raw, &params); err != nil {
		return webRooterDecodedArgs{}, fmt.Errorf("decode site_search params: %w", err)
	}
	url, err := requireWebRooterString("site_search.url", params.URL)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	query, err := requireWebRooterString("site_search.query", params.Query)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	useBrowser, err := requireWebRooterBool("site_search.use_browser", params.UseBrowser)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	return webRooterDecodedArgs{action: webRooterActionSiteSearch, path: "/search/site", body: map[string]any{"url": url, "query": query, "use_browser": useBrowser}}, nil
}

func decodeWebRooterFetch(raw json.RawMessage) (webRooterDecodedArgs, error) {
	var params webRooterFetchParams
	if err := decodeWebRooterStrictJSON(raw, &params); err != nil {
		return webRooterDecodedArgs{}, fmt.Errorf("decode fetch params: %w", err)
	}
	url, err := requireWebRooterString("fetch.url", params.URL)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	useBrowser, err := requireWebRooterBool("fetch.use_browser", params.UseBrowser)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	return webRooterDecodedArgs{action: webRooterActionFetch, path: "/fetch", body: map[string]any{"url": url, "use_browser": useBrowser}}, nil
}

func decodeWebRooterExtract(raw json.RawMessage) (webRooterDecodedArgs, error) {
	var params webRooterExtractParams
	if err := decodeWebRooterStrictJSON(raw, &params); err != nil {
		return webRooterDecodedArgs{}, fmt.Errorf("decode extract params: %w", err)
	}
	url, err := requireWebRooterString("extract.url", params.URL)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	target, err := requireWebRooterString("extract.target", params.Target)
	if err != nil {
		return webRooterDecodedArgs{}, err
	}
	return webRooterDecodedArgs{action: webRooterActionExtract, path: "/extract", body: map[string]any{"url": url, "target": target}}, nil
}

func decodeWebRooterStrictJSON(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return fmt.Errorf("unexpected trailing JSON content")
	}
	return nil
}

func requireWebRooterString(name string, value *string) (string, error) {
	if value == nil {
		return "", fmt.Errorf("%s is required", name)
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return "", fmt.Errorf("%s must not be empty", name)
	}
	return trimmed, nil
}

func requireWebRooterPositiveInt(name string, value *int) (int, error) {
	if value == nil {
		return 0, fmt.Errorf("%s is required", name)
	}
	if *value <= 0 {
		return 0, fmt.Errorf("%s must be > 0", name)
	}
	return *value, nil
}

func requireWebRooterBool(name string, value *bool) (bool, error) {
	if value == nil {
		return false, fmt.Errorf("%s is required", name)
	}
	return *value, nil
}
