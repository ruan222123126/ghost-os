package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"

	ctxmgr "ghost-os/bridge/context"
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
	cfg      Config
	worker   toolSelectorCompleter
	metadata string
}

func NewToolSelector(cfg Config, worker toolSelectorCompleter) *ToolSelector {
	return NewToolSelectorForCatalog(cfg, worker, nil)
}

func NewToolSelectorForCatalog(cfg Config, worker toolSelectorCompleter, catalog tools.ToolCatalog) *ToolSelector {
	metadata := tools.FormatMetadataForCatalog(catalog)
	if strings.TrimSpace(metadata) == "" {
		metadata = tools.FormatMetadataForSelector()
	}
	return &ToolSelector{
		cfg:      cfg,
		worker:   worker,
		metadata: metadata,
	}
}

func newToolSelectorFromConfig(cfg Config, catalog tools.ToolCatalog) selectorEngine {
	if !cfg.ToolSelector.Enabled {
		return nil
	}
	if mode := strings.ToLower(strings.TrimSpace(cfg.ToolSelector.Mode)); mode != "" && mode != "llm" {
		return nil
	}

	client := llm.NewClientWithOptions(providerClientOptions(cfg, toolSelectorModel(cfg)))
	return NewToolSelectorForCatalog(cfg, client, catalog)
}

func toolSelectorModel(cfg Config) string {
	if model := strings.TrimSpace(cfg.ToolSelector.Model); model != "" {
		return model
	}
	if model := strings.TrimSpace(cfg.Worker.Model); model != "" {
		return model
	}
	return strings.TrimSpace(cfg.Provider.Model)
}

func buildSystemPromptForCatalog(cfg Config, catalog tools.ToolCatalog) (string, error) {
	promptManager, err := ctxmgr.NewPromptManagerWithOptions(ctxmgr.PromptLoadOptions{
		ConfigPath: cfg.PromptsPath,
		CoreDir:    cfg.PromptsDir,
		CoreFiles:  cfg.PromptsCoreFiles,
	})
	if err != nil {
		if len(cfg.PromptsCoreFiles) == 0 {
			promptManager = ctxmgr.NewPromptManagerWithDefault()
		} else {
			return "", err
		}
	}
	contextBuilder := ctxmgr.NewBuilder(promptManager, catalog)
	return contextBuilder.BuildSystemPrompt(map[string]string{
		"os_type":     runtime.GOOS,
		"tools_count": strconv.Itoa(len(catalog.ToolDefs())),
		"max_turns":   strconv.Itoa(cfg.MaxTurns),
	}), nil
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

func (ts *ToolSelector) selectorSystemPrompt() string {
	return `You are a tool selector for Ghost-OS.

Choose the MINIMAL sufficient tool subset for the current turn.

Rules:
- ALWAYS include ask_human.
- Decision hints are advisory, not mandatory.
- Prefer the current request and available tools over historical hints when they conflict.
- Do not infer unavailable tools from hints.
- Return mode="all" when uncertain, mixed-domain, or broad.
- Return mode="all" when hints are weak, stale, or not clearly applicable.
- Return mode="subset" only for focused requests.
- Confidence must be 0.0-1.0.
- Output JSON only.

JSON schema:
{"mode":"subset|all","tools":["tool_name"],"confidence":0.85,"reason":"brief explanation"}`
}

func (ts *ToolSelector) buildSelectorPrompt(userMessage string, recentHistory []llm.Message, decisionHint string) string {
	var sb strings.Builder
	sb.WriteString("Available tools:\n")
	sb.WriteString(ts.metadata)
	sb.WriteString("\n\nRecent context:\n")

	if len(recentHistory) == 0 {
		sb.WriteString("(none)\n")
	} else {
		for _, msg := range recentHistory {
			switch msg.Role {
			case llm.RoleUser:
				sb.WriteString(fmt.Sprintf("User: %s\n", truncateSelectorText(msg.Text, 200)))
			case llm.RoleAssistant:
				sb.WriteString(fmt.Sprintf("Assistant: %s\n", truncateSelectorText(msg.Text, 200)))
			}
		}
	}

	trimmedHint := truncateSelectorText(decisionHint, 500)
	if trimmedHint != "" {
		sb.WriteString("\nPrior similar experience:\n")
		sb.WriteString(trimmedHint)
		sb.WriteString("\n")
	}

	sb.WriteString("\nCurrent request:\n")
	sb.WriteString(truncateSelectorText(userMessage, 500))
	return sb.String()
}

func (ts *ToolSelector) parseResponse(response string, traceID string, latencyMS int64) ToolSelectorResult {
	trimmedTraceID := strings.TrimSpace(traceID)
	parsedJSON := extractJSONObject(response)

	var parsed struct {
		Mode       string   `json:"mode"`
		Tools      []string `json:"tools"`
		Confidence float64  `json:"confidence"`
		Reason     string   `json:"reason"`
	}
	if err := json.Unmarshal([]byte(parsedJSON), &parsed); err != nil {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=parse_error latency_ms=%d error=%v", trimmedTraceID, latencyMS, err)
		return ToolSelectorResult{Mode: "all", Fallback: true, Error: err}
	}

	parsed.Mode = strings.ToLower(strings.TrimSpace(parsed.Mode))
	parsed.Reason = strings.TrimSpace(parsed.Reason)
	if parsed.Mode != "subset" && parsed.Mode != "all" {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=invalid_mode latency_ms=%d mode=%q", trimmedTraceID, latencyMS, parsed.Mode)
		return ToolSelectorResult{Mode: "all", Fallback: true, Error: fmt.Errorf("invalid selector mode %q", parsed.Mode)}
	}
	if parsed.Confidence < 0 || parsed.Confidence > 1 {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=invalid_confidence latency_ms=%d confidence=%.2f", trimmedTraceID, latencyMS, parsed.Confidence)
		return ToolSelectorResult{Mode: "all", Fallback: true, Error: fmt.Errorf("invalid confidence %.2f", parsed.Confidence)}
	}
	if parsed.Mode == "all" {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=all latency_ms=%d confidence=%.2f reason=%q", trimmedTraceID, latencyMS, parsed.Confidence, parsed.Reason)
		return ToolSelectorResult{Mode: "all", Confidence: parsed.Confidence, Reason: parsed.Reason}
	}

	selected := normalizeToolNames(parsed.Tools)
	if len(selected) == 0 {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=empty_tools latency_ms=%d", trimmedTraceID, latencyMS)
		return ToolSelectorResult{Mode: "all", Fallback: true, Error: fmt.Errorf("selector returned empty tool list")}
	}
	if parsed.Confidence < ts.cfg.ToolSelector.Confidence {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=low_confidence latency_ms=%d confidence=%.2f threshold=%.2f", trimmedTraceID, latencyMS, parsed.Confidence, ts.cfg.ToolSelector.Confidence)
		return ToolSelectorResult{Mode: "all", Fallback: true, Confidence: parsed.Confidence, Reason: parsed.Reason}
	}
	if !containsToolName(selected, "ask_human") {
		selected = append(selected, "ask_human")
	}

	validTools := make(map[string]bool, len(tools.GetToolMetadata()))
	for _, item := range tools.GetToolMetadata() {
		validTools[item.Name] = true
	}
	for _, name := range selected {
		if !validTools[name] {
			log.Printf("trace_id=%s action=TOOL_SELECTOR status=unknown_tool latency_ms=%d tool=%q", trimmedTraceID, latencyMS, name)
			return ToolSelectorResult{Mode: "all", Fallback: true, Error: fmt.Errorf("unknown tool %q", name)}
		}
	}

	log.Printf("trace_id=%s action=TOOL_SELECTOR status=subset_selected latency_ms=%d tools=%v confidence=%.2f reason=%q", trimmedTraceID, latencyMS, selected, parsed.Confidence, parsed.Reason)
	return ToolSelectorResult{
		Mode:       "subset",
		Tools:      selected,
		Confidence: parsed.Confidence,
		Reason:     parsed.Reason,
	}
}

func extractJSONObject(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return trimmed
	}
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end >= start {
		return trimmed[start : end+1]
	}
	return trimmed
}

func normalizeToolNames(names []string) []string {
	if len(names) == 0 {
		return nil
	}

	result := make([]string, 0, len(names))
	seen := make(map[string]bool, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	return result
}

func containsToolName(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}

func truncateSelectorText(text string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxLen {
		return string(runes)
	}
	return string(runes[:maxLen]) + "..."
}
