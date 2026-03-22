package runtime

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type toolSelectorCompleter interface {
	Complete(context.Context, llm.CompletionRequest) (*llm.CompletionResponse, error)
}

type selectorEngine interface {
	SelectTools(ctx context.Context, userMessage string, recentHistory []llm.Message, decisionHint string, traceID string) ToolSelectorResult
}

// ToolSelectorResult 描述 selector 的输出与是否触发了安全回退。
type ToolSelectorResult struct {
	Mode       string
	Tools      []string
	Confidence float64
	Reason     string
	Fallback   bool
	Error      error
}

// ToolSelector 使用轻量 prompt 让次级模型挑选最小充分工具集。
type ToolSelector struct {
	cfg        Config
	worker     toolSelectorCompleter
	metadata   string
	validTools map[string]bool
	fallback   bool
}

func NewToolSelector(cfg Config, worker toolSelectorCompleter) *ToolSelector {
	return NewToolSelectorForCatalog(cfg, worker, nil)
}

func NewToolSelectorForCatalog(cfg Config, worker toolSelectorCompleter, catalog tools.ToolCatalog) *ToolSelector {
	metadata := tools.FormatMetadataForCatalog(catalog)
	validTools := selectorValidToolsForCatalog(catalog)
	if catalog == nil {
		if strings.TrimSpace(metadata) == "" {
			metadata = tools.FormatMetadataForSelector()
		}
		if len(validTools) == 0 {
			validTools = validSelectorToolNames()
		}
	}
	return &ToolSelector{
		cfg:        cfg,
		worker:     worker,
		metadata:   metadata,
		validTools: validTools,
		fallback:   catalog == nil,
	}
}

func selectorValidToolsForCatalog(catalog tools.ToolCatalog) map[string]bool {
	validTools := make(map[string]bool)
	for _, name := range tools.CatalogToolNames(catalog) {
		validTools[name] = true
	}
	return validTools
}

func (ts *ToolSelector) SelectTools(ctx context.Context, userMessage string, recentHistory []llm.Message, decisionHint string, traceID string) ToolSelectorResult {
	if ts == nil || !ts.cfg.ToolSelector.Enabled {
		return ToolSelectorResult{Mode: "all", Fallback: true}
	}
	if mode := strings.ToLower(strings.TrimSpace(ts.cfg.ToolSelector.Mode)); mode != "" && mode != "llm" {
		err := fmt.Errorf("unsupported tool selector mode %q", ts.cfg.ToolSelector.Mode)
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=unsupported_mode mode=%q", strings.TrimSpace(traceID), ts.cfg.ToolSelector.Mode)
		return ToolSelectorResult{Mode: "all", Fallback: true, Error: err}
	}
	if ts.worker == nil {
		return ToolSelectorResult{Mode: "all", Fallback: true, Error: fmt.Errorf("selector worker is not configured")}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(ts.cfg.ToolSelector.TimeoutMS)*time.Millisecond)
	defer cancel()

	start := time.Now()
	resp, err := ts.worker.Complete(timeoutCtx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Text: ts.selectorSystemPrompt()},
			{Role: llm.RoleUser, Text: ts.buildSelectorPrompt(userMessage, recentHistory, decisionHint)},
		},
	})
	latencyMS := time.Since(start).Milliseconds()
	if err != nil {
		status := "error"
		if timeoutCtx.Err() == context.DeadlineExceeded {
			status = "timeout"
		}
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=%s latency_ms=%d error=%v", strings.TrimSpace(traceID), status, latencyMS, err)
		return ToolSelectorResult{Mode: "all", Fallback: true, Error: err}
	}
	if resp == nil {
		err := fmt.Errorf("selector returned nil response")
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=nil_response latency_ms=%d", strings.TrimSpace(traceID), latencyMS)
		return ToolSelectorResult{Mode: "all", Fallback: true, Error: err}
	}

	return ts.parseResponse(resp.Message.Text, traceID, latencyMS)
}
