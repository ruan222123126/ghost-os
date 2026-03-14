package memoryaug

import "strings"

type learningPolicy struct {
	shouldLearn bool
	allowFact   bool
}

var explicitRememberMarkers = []string{
	"remember", "keep in mind", "note this", "记住", "记一下", "记下来",
}

var stablePreferenceMarkers = []string{
	"prefer", "preference", "default", "by default", "always", "reply in",
	"respond in", "concise", "verbose", "tone", "style", "approval",
	"destructive", "auto apply", "以后", "默认", "习惯", "偏好", "请用",
	"简洁", "详细", "语气", "风格", "危险操作", "自动应用", "批准",
}

var stableWorkflowMarkers = []string{
	"use pnpm", "use npm", "use cargo", "use go test", "run tests first",
	"build command", "test command", "start command", "workflow", "步骤",
	"流程", "先跑", "先执行", "命令", "构建命令", "测试命令", "启动命令",
}

func deriveLearningPolicy(messages []TurnMessage) learningPolicy {
	policy := learningPolicy{}
	for _, message := range messages {
		if normalizeRole(message.Role) != "user" {
			continue
		}
		text := strings.TrimSpace(message.Text)
		if text == "" {
			continue
		}
		if hasMarker(text, explicitRememberMarkers) {
			policy.shouldLearn = true
			policy.allowFact = true
			continue
		}
		if !isInformativeDurableText(text) {
			continue
		}
		if hasMarker(text, stablePreferenceMarkers) ||
			hasMarker(text, stableWorkflowMarkers) {
			policy.shouldLearn = true
		}
	}
	return policy
}

func hasMarker(text string, markers []string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func isInformativeDurableText(text string) bool {
	normalized := normalizeText(text)
	if normalized == "" {
		return false
	}
	return len([]rune(normalized)) >= 8 || len(strings.Fields(normalized)) >= 3
}
