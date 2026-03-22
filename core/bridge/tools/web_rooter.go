package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/tools/internal/webrooter"
)

const (
	webRooterToolName = "web_rooter"
)

const webRooterDescription = "Call the pinned web-rooter v0.2.4 HTTP service for stateless internet research, academic search, fetch, and extract operations. Use exactly one action with an explicit params object."

const webRooterSchema = `{
	"type":"object",
	"properties":{
		"action":{"type":"string","enum":["internet_search","research","academic_search","site_search","fetch","extract"]},
		"params":{
			"type":"object",
			"description":"Action-specific parameters. Provide the full explicit parameter set for the chosen action: internet_search(query,num_results,auto_crawl), research(topic,max_pages), academic_search(query,num_results,include_code,fetch_abstracts), site_search(url,query,use_browser), fetch(url,use_browser), extract(url,target).",
			"properties":{
				"query":{"type":"string","description":"Query string for internet_search, academic_search, or site_search."},
				"num_results":{"type":"integer","minimum":1,"description":"Number of results for internet_search or academic_search."},
				"auto_crawl":{"type":"boolean","description":"Whether internet_search should auto crawl result pages."},
				"topic":{"type":"string","description":"Research topic for research."},
				"max_pages":{"type":"integer","minimum":1,"description":"Maximum pages for research."},
				"include_code":{"type":"boolean","description":"Whether academic_search should include code projects."},
				"fetch_abstracts":{"type":"boolean","description":"Whether academic_search should fetch abstracts."},
				"url":{"type":"string","description":"Target URL for site_search, fetch, or extract."},
				"use_browser":{"type":"boolean","description":"Whether site_search or fetch should use browser rendering."},
				"target":{"type":"string","description":"Extraction target description for extract."}
			},
			"additionalProperties":false
		}
	},
	"required":["action","params"],
	"additionalProperties":false
}`

type WebRooterConfig struct {
	BaseURL    string
	APIToken   string
	Timeout    time.Duration
	HTTPClient webrooter.HTTPClient
}

type WebRooterTool struct {
	client *webrooter.Client
}

type webRooterResultEnvelope struct {
	Provider       string         `json:"provider"`
	Action         string         `json:"action"`
	Payload        map[string]any `json:"payload"`
	Citations      []any          `json:"citations"`
	ReferencesText string         `json:"references_text"`
	TraceID        string         `json:"trace_id"`
}

func NewWebRooterTool(cfg WebRooterConfig) Tool {
	return &WebRooterTool{
		client: webrooter.NewClient(webrooter.Config{
			BaseURL:    cfg.BaseURL,
			APIToken:   cfg.APIToken,
			Timeout:    cfg.Timeout,
			HTTPClient: cfg.HTTPClient,
		}),
	}
}

func (WebRooterTool) Name() string {
	return webRooterToolName
}

func (WebRooterTool) Description() string {
	return webRooterDescription
}

func (WebRooterTool) Parameters() json.RawMessage {
	return json.RawMessage(webRooterSchema)
}

func (t *WebRooterTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t == nil || t.client == nil {
		return "", fmt.Errorf("web_rooter client is not configured")
	}

	args, err := decodeWebRooterArgs(argsJSON)
	if err != nil {
		return "", err
	}

	result, err := t.client.Execute(ctx, webrooter.Request{
		Action:  args.action,
		Path:    args.path,
		Body:    args.body,
		TraceID: traceID,
	})
	if err != nil {
		return "", err
	}
	envelope := webRooterResultEnvelope{
		Provider:       webRooterToolName,
		Action:         args.action,
		Payload:        result.Payload,
		Citations:      result.Citations,
		ReferencesText: result.ReferencesText,
		TraceID:        strings.TrimSpace(traceID),
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("encode web_rooter result: %w", err)
	}
	return string(encoded), nil
}
