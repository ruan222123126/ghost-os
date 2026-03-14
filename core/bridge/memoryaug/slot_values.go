package memoryaug

import (
	"fmt"
	"strings"

	"ghost-os/bridge/memorystore"
)

const (
	slotMetadataValueKey   = "value"
	slotMetadataVersionKey = "slot_version"
	slotMetadataReasonKey  = "learn_reason"
)

func slotValueFromEntry(entry memorystore.MemoryEntry) string {
	if entry.Metadata != nil {
		value, ok := entry.Metadata[slotMetadataValueKey]
		if ok {
			text, ok := value.(string)
			if ok {
				return strings.TrimSpace(text)
			}
		}
	}
	spec, ok := slotSpecForKey(entry.MemoryKey)
	if !ok {
		return ""
	}
	return normalizeSlotValue(spec, Candidate{
		Value:   "",
		Summary: entry.Summary,
		Content: entry.Content,
	})
}

func normalizeSlotValue(spec SlotSpec, candidate Candidate) string {
	joined := strings.Join([]string{candidate.Value, candidate.Summary, candidate.Content}, "\n")
	switch spec.Key {
	case "reply_language":
		return normalizeReplyLanguageValue(joined)
	case "response_style":
		return normalizeResponseStyleValue(joined)
	case "approval_style":
		return normalizeApprovalStyleValue(joined)
	case "package_manager":
		return normalizePackageManagerValue(joined)
	case "build_command", "test_command":
		return normalizeCommandValue(candidate.Value, candidate.Content)
	default:
		return ""
	}
}

func buildSlotMetadata(value string, reason string) map[string]any {
	metadata := map[string]any{
		slotMetadataValueKey:   strings.TrimSpace(value),
		slotMetadataVersionKey: slotVersion,
	}
	if strings.TrimSpace(reason) != "" {
		metadata[slotMetadataReasonKey] = strings.TrimSpace(reason)
	}
	return metadata
}

func renderSlotContent(spec SlotSpec, value string) string {
	switch spec.Key {
	case "reply_language":
		if value == "zh-CN" {
			return "Reply in Chinese by default."
		}
		if value == "en-US" {
			return "Reply in English by default."
		}
		return "Reply bilingually when it helps."
	case "response_style":
		if value == "concise" {
			return "Keep responses concise."
		}
		if value == "detailed" {
			return "Provide detailed responses."
		}
		return "Explain things step by step."
	case "approval_style":
		if value == "ask_before_destructive" {
			return "Ask before destructive changes."
		}
		return "Auto-apply safe changes without asking."
	case "package_manager":
		return fmt.Sprintf("This project uses %s.", value)
	case "build_command":
		return fmt.Sprintf("Use `%s` as the build command.", value)
	case "test_command":
		return fmt.Sprintf("Use `%s` as the test command.", value)
	default:
		return strings.TrimSpace(value)
	}
}

func normalizeReplyLanguageValue(raw string) string {
	normalized := normalizeText(raw)
	switch {
	case containsAny(normalized, "双语", "bilingual", "both languages", "中英"):
		return "bilingual"
	case containsAny(normalized, "中文", "chinese", "mandarin", "zh cn"):
		return "zh-CN"
	case containsAny(normalized, "英文", "english", "en us"):
		return "en-US"
	default:
		return ""
	}
}

func normalizeResponseStyleValue(raw string) string {
	normalized := normalizeText(raw)
	switch {
	case containsAny(normalized, "step by step", "stepbystep", "分步", "逐步"):
		return "step_by_step"
	case containsAny(normalized, "concise", "brief", "short", "简洁", "精简"):
		return "concise"
	case containsAny(normalized, "detailed", "verbose", "详细", "展开"):
		return "detailed"
	default:
		return ""
	}
}

func normalizeApprovalStyleValue(raw string) string {
	normalized := normalizeText(raw)
	switch {
	case containsAny(normalized, "ask before destructive", "before destructive changes", "危险操作前先问", "先确认再删", "先问"):
		return "ask_before_destructive"
	case containsAny(normalized, "auto apply safe", "autoapply safe", "自动应用安全修改", "自动应用安全变更"):
		return "auto_apply_safe_changes"
	default:
		return ""
	}
}

func normalizePackageManagerValue(raw string) string {
	normalized := normalizeText(raw)
	switch {
	case containsAny(normalized, "pnpm"):
		return "pnpm"
	case containsAny(normalized, "npm"):
		return "npm"
	case containsAny(normalized, "yarn"):
		return "yarn"
	case containsAny(normalized, "cargo"):
		return "cargo"
	case containsAny(normalized, "go mod", "go test", "golang"):
		return "go"
	default:
		return ""
	}
}

func normalizeCommandValue(value string, content string) string {
	joined := strings.TrimSpace(value)
	if joined == "" {
		joined = extractBacktickValue(content)
	}
	if joined == "" {
		joined = strings.TrimSpace(content)
	}
	return collapseWhitespace(joined)
}

func extractBacktickValue(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	parts := strings.Split(text, "`")
	if len(parts) < 3 {
		return ""
	}
	for idx := 1; idx < len(parts); idx += 2 {
		value := collapseWhitespace(parts[idx])
		if value != "" {
			return value
		}
	}
	return ""
}

func collapseWhitespace(raw string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
}

func containsAny(text string, values ...string) bool {
	for _, value := range values {
		if containsNormalized(text, value) {
			return true
		}
	}
	return false
}
