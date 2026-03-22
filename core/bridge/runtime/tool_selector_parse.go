package runtime

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"ghost-os/bridge/tools"
)

type selectorResponseEnvelope struct {
	Mode       string   `json:"mode"`
	Tools      []string `json:"tools"`
	Confidence float64  `json:"confidence"`
	Reason     string   `json:"reason"`
}

func (ts *ToolSelector) parseResponse(response string, traceID string, latencyMS int64) ToolSelectorResult {
	parsed, err := parseSelectorEnvelope(response)
	if err != nil {
		logSelectorParseError(traceID, latencyMS, err)
		return ToolSelectorResult{Mode: "all", Fallback: true, Error: err}
	}

	return ts.evaluateParsedResponse(parsed, traceID, latencyMS)
}

func (ts *ToolSelector) evaluateParsedResponse(parsed selectorResponseEnvelope, traceID string, latencyMS int64) ToolSelectorResult {
	trimmedTraceID := strings.TrimSpace(traceID)
	if err := validateSelectorEnvelope(parsed); err != nil {
		logSelectorValidationError(trimmedTraceID, latencyMS, parsed, err)
		return ToolSelectorResult{Mode: "all", Fallback: true, Error: err}
	}
	if parsed.Mode == "all" {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=all latency_ms=%d confidence=%.2f reason=%q", trimmedTraceID, latencyMS, parsed.Confidence, parsed.Reason)
		return ToolSelectorResult{Mode: "all", Confidence: parsed.Confidence, Reason: parsed.Reason}
	}

	selected, result := ts.resolveSelectedTools(parsed, trimmedTraceID, latencyMS)
	if result != nil {
		return *result
	}
	log.Printf("trace_id=%s action=TOOL_SELECTOR status=subset_selected latency_ms=%d tools=%v confidence=%.2f reason=%q", trimmedTraceID, latencyMS, selected, parsed.Confidence, parsed.Reason)
	return ToolSelectorResult{Mode: "subset", Tools: selected, Confidence: parsed.Confidence, Reason: parsed.Reason}
}

func parseSelectorEnvelope(response string) (selectorResponseEnvelope, error) {
	var parsed selectorResponseEnvelope
	if err := json.Unmarshal([]byte(extractJSONObject(response)), &parsed); err != nil {
		return selectorResponseEnvelope{}, err
	}
	parsed.Mode = strings.ToLower(strings.TrimSpace(parsed.Mode))
	parsed.Reason = strings.TrimSpace(parsed.Reason)
	return parsed, nil
}

func validateSelectorEnvelope(parsed selectorResponseEnvelope) error {
	if parsed.Mode != "subset" && parsed.Mode != "all" {
		return fmt.Errorf("invalid selector mode %q", parsed.Mode)
	}
	if parsed.Confidence < 0 || parsed.Confidence > 1 {
		return fmt.Errorf("invalid confidence %.2f", parsed.Confidence)
	}
	return nil
}

func (ts *ToolSelector) resolveSelectedTools(parsed selectorResponseEnvelope, traceID string, latencyMS int64) ([]string, *ToolSelectorResult) {
	selected := normalizeToolNames(parsed.Tools)
	if len(selected) == 0 {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=empty_tools latency_ms=%d", traceID, latencyMS)
		result := ToolSelectorResult{Mode: "all", Fallback: true, Error: fmt.Errorf("selector returned empty tool list")}
		return nil, &result
	}
	if parsed.Confidence < ts.cfg.ToolSelector.Confidence {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=low_confidence latency_ms=%d confidence=%.2f threshold=%.2f", traceID, latencyMS, parsed.Confidence, ts.cfg.ToolSelector.Confidence)
		result := ToolSelectorResult{Mode: "all", Fallback: true, Confidence: parsed.Confidence, Reason: parsed.Reason}
		return nil, &result
	}

	if err := ts.validateSelectedTools(selected); err != nil {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=unknown_tool latency_ms=%d error=%v", traceID, latencyMS, err)
		result := ToolSelectorResult{Mode: "all", Fallback: true, Error: err}
		return nil, &result
	}
	return selected, nil
}

func logSelectorParseError(traceID string, latencyMS int64, err error) {
	log.Printf("trace_id=%s action=TOOL_SELECTOR status=parse_error latency_ms=%d error=%v", strings.TrimSpace(traceID), latencyMS, err)
}

func logSelectorValidationError(traceID string, latencyMS int64, parsed selectorResponseEnvelope, err error) {
	status := "invalid_mode"
	if strings.Contains(err.Error(), "confidence") {
		status = "invalid_confidence"
	}
	log.Printf("trace_id=%s action=TOOL_SELECTOR status=%s latency_ms=%d mode=%q confidence=%.2f error=%v", traceID, status, latencyMS, parsed.Mode, parsed.Confidence, err)
}

func (ts *ToolSelector) validateSelectedTools(selected []string) error {
	valid := ts.validTools
	if len(valid) == 0 {
		if ts == nil || !ts.fallback {
			return fmt.Errorf("no visible tools available")
		}
		valid = validSelectorToolNames()
	}
	for _, name := range selected {
		if !valid[name] {
			return fmt.Errorf("unknown tool %q", name)
		}
	}
	return nil
}

func validSelectorToolNames() map[string]bool {
	valid := make(map[string]bool, len(tools.GetToolMetadata()))
	for _, item := range tools.GetToolMetadata() {
		if name := strings.TrimSpace(item.Name); name != "" && name != tools.ToolSearchToolName && !item.OnDemand {
			valid[name] = true
		}
	}
	return valid
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
