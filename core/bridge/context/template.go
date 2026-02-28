package context

import "regexp"

var templateVariablePattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

// RenderTemplate 执行简单的 {{variable}} 替换，缺失变量时保留占位符。
func RenderTemplate(template string, vars map[string]string) string {
	if template == "" {
		return ""
	}
	if len(vars) == 0 {
		return template
	}

	return templateVariablePattern.ReplaceAllStringFunc(template, func(token string) string {
		matches := templateVariablePattern.FindStringSubmatch(token)
		if len(matches) != 2 {
			return token
		}

		value, ok := vars[matches[1]]
		if !ok {
			return token
		}
		return value
	})
}
