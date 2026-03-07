package memory

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	intentQuotedPattern     = regexp.MustCompile("`([^`]+)`|\"([^\"]+)\"|'([^']+)'")
	intentPathPattern       = regexp.MustCompile(`(?:~?/[^\s,;:]+|[A-Za-z]:[\\/][^\s,;:]+)`)
	intentIdentifierPattern = regexp.MustCompile(`(?i)\b[a-z_][a-z0-9_.:/-]{2,}\b`)
)

var intentActionRules = []struct {
	key   string
	terms []string
}{
	{key: "fix", terms: []string{"fix", "patch", "repair", "migrate", "修复", "修", "补丁"}},
	{key: "inspect", terms: []string{"inspect", "check", "verify", "read", "debug", "investigate", "查看", "检查", "调查", "排查"}},
	{key: "explain", terms: []string{"explain", "summarize", "describe", "clarify", "解释", "总结", "说明"}},
	{key: "create", terms: []string{"create", "add", "build", "implement", "scaffold", "新增", "创建", "实现", "搭建"}},
	{key: "run", terms: []string{"run", "execute", "test", "deploy", "执行", "运行", "测试", "部署"}},
	{key: "compare", terms: []string{"compare", "diff", "contrast", "比较", "对比"}},
	{key: "plan", terms: []string{"plan", "design", "schedule", "规划", "设计"}},
	{key: "recall", terms: []string{"recall", "search", "retrieve", "query", "检索", "查询", "召回"}},
}

var intentConstraintMarkers = []string{
	"do not", "don't", "without", "must", "keep", "preserve", "only", "avoid", "不要", "不改", "不能", "保持", "仅", "只", "必须", "避免",
}

var intentRiskMarkers = []string{
	"risk", "danger", "destructive", "delete", "drop", "rm ", "permission", "security", "production", "prod", "credential", "secret", "破坏", "删除", "危险", "权限", "安全", "生产", "密钥",
}

var intentEnvironmentHints = map[string]string{
	"linux":      "platform:linux",
	"darwin":     "platform:darwin",
	"macos":      "platform:macos",
	"windows":    "platform:windows",
	"browser":    "target:browser",
	"web":        "target:web",
	"cli":        "target:cli",
	"docker":     "platform:docker",
	"k8s":        "platform:k8s",
	"kubernetes": "platform:kubernetes",
	"workspace":  "scope:workspace",
	"repo":       "scope:repo",
	"local":      "scope:local",
	"staging":    "scope:staging",
	"prod":       "scope:prod",
	"production": "scope:prod",
}

// IntentPlanner 把原始 query 归一成稳定的 query intent plan。
type IntentPlanner struct {
	enabled bool
	metrics *memoryCounters
}

func NewIntentPlanner(enabled bool, metrics *memoryCounters) *IntentPlanner {
	if !enabled {
		return nil
	}
	return &IntentPlanner{enabled: true, metrics: metrics}
}

func (p *IntentPlanner) Enabled() bool {
	return p != nil && p.enabled
}

func (p *IntentPlanner) Plan(query MemoryQuery, scope SessionScope) (QueryIntentPlan, error) {
	if !p.Enabled() {
		return QueryIntentPlan{}, nil
	}
	if p.metrics != nil {
		p.metrics.plannerRuns.Add(1)
	}

	raw := strings.TrimSpace(firstNonEmpty(query.SemanticQuery, strings.Join(query.Keywords, " ")))
	plan := QueryIntentPlan{
		IntentKey:   deriveIntentKey(query),
		Constraints: extractIntentConstraints(raw),
		Entities:    extractIntentEntities(raw, query.Keywords),
		Environment: extractIntentEnvironment(raw, query, scope),
		Risks:       extractIntentRisks(raw),
	}
	terms := append([]string(nil), query.Keywords...)
	terms = append(terms, extractKeywords(raw)...)
	terms = append(terms, plan.Entities...)
	terms = append(terms, plan.Constraints...)
	terms = append(terms, plan.Risks...)
	terms = append(terms, plan.Environment...)
	terms = append(terms, strings.TrimPrefix(plan.IntentKey, "intent."))
	plan.Terms = uniqueStrings(flattenIntentTerms(terms))

	signals := 0
	if plan.IntentKey != "" {
		signals++
	}
	if len(plan.Constraints) > 0 {
		signals++
	}
	if len(plan.Entities) > 0 {
		signals++
	}
	if len(plan.Environment) > 0 {
		signals++
	}
	if len(plan.Risks) > 0 {
		signals++
	}
	plan.Confidence = clamp01(0.3 + float64(signals)*0.11 + float64(minInt(len(plan.Terms), 6))*0.02)
	return normalizeQueryIntentPlan(plan), nil
}

func normalizeQueryIntentPlan(plan QueryIntentPlan) QueryIntentPlan {
	out := plan
	out.IntentKey = strings.TrimSpace(out.IntentKey)
	out.Constraints = uniqueStrings(out.Constraints)
	out.Entities = uniqueStrings(out.Entities)
	out.Environment = uniqueStrings(out.Environment)
	out.Risks = uniqueStrings(out.Risks)
	out.Terms = uniqueStrings(flattenIntentTerms(out.Terms))
	out.Confidence = clamp01(out.Confidence)
	return out
}

func deriveIntentKey(query MemoryQuery) string {
	raw := strings.ToLower(strings.TrimSpace(firstNonEmpty(query.SemanticQuery, strings.Join(query.Keywords, " "))))
	if raw == "" {
		return ""
	}
	action := "query"
	for _, rule := range intentActionRules {
		for _, term := range rule.terms {
			if strings.Contains(raw, strings.ToLower(term)) {
				action = rule.key
				break
			}
		}
		if action == rule.key {
			break
		}
	}
	fragments := make([]string, 0, 3)
	for _, token := range flattenIntentTerms(append(query.Keywords, extractKeywords(raw)...)) {
		fragment := sanitizeIntentKeyFragment(token)
		if fragment == "" || fragment == action || isIntentStopFragment(fragment) {
			continue
		}
		fragments = append(fragments, fragment)
		if len(fragments) >= 3 {
			break
		}
	}
	if len(fragments) == 0 {
		return "intent." + action
	}
	return "intent." + action + "_" + strings.Join(fragments, "_")
}

func extractIntentConstraints(raw string) []string {
	clauses := splitIntentClauses(raw)
	out := make([]string, 0, len(clauses))
	for _, clause := range clauses {
		lowered := strings.ToLower(clause)
		for _, marker := range intentConstraintMarkers {
			if strings.Contains(lowered, strings.ToLower(marker)) {
				out = append(out, summarizeLine(clause, 120))
				break
			}
		}
	}
	return uniqueStrings(out)
}

func extractIntentEntities(raw string, keywords []string) []string {
	out := make([]string, 0, 8)
	for _, match := range intentQuotedPattern.FindAllString(raw, -1) {
		out = append(out, strings.Trim(match, "`\"' "))
	}
	for _, match := range intentPathPattern.FindAllString(raw, -1) {
		out = append(out, strings.TrimSpace(match))
	}
	for _, match := range intentIdentifierPattern.FindAllString(raw, -1) {
		cleaned := strings.TrimSpace(match)
		if isIntentStopFragment(sanitizeIntentKeyFragment(cleaned)) {
			continue
		}
		if len(cleaned) < 3 {
			continue
		}
		out = append(out, cleaned)
	}
	for _, keyword := range keywords {
		fragment := sanitizeIntentKeyFragment(keyword)
		if fragment == "" || isIntentStopFragment(fragment) {
			continue
		}
		out = append(out, strings.TrimSpace(keyword))
	}
	return uniqueStrings(out)
}

func extractIntentEnvironment(raw string, query MemoryQuery, scope SessionScope) []string {
	out := make([]string, 0, 10)
	if env := plannerEnvironmentFingerprint(query, scope); env != nil {
		if value := strings.TrimSpace(env.Domain); value != "" {
			out = append(out, "domain:"+value)
		}
		if value := strings.TrimSpace(firstNonEmpty(env.Platform, env.OS)); value != "" {
			out = append(out, "platform:"+value)
		}
		if value := strings.TrimSpace(env.WorkspaceRoot); value != "" {
			out = append(out, "workspace:"+value)
		}
		if value := strings.TrimSpace(env.GraphNamespace); value != "" {
			out = append(out, "graph:"+value)
		}
		if value := strings.TrimSpace(env.TargetAppOrSite); value != "" {
			out = append(out, "target:"+value)
		}
		if value := strings.TrimSpace(env.Provider); value != "" {
			out = append(out, "provider:"+value)
		}
		if value := strings.TrimSpace(env.Model); value != "" {
			out = append(out, "model:"+value)
		}
		if value := strings.TrimSpace(env.Shell); value != "" {
			out = append(out, "shell:"+value)
		}
		for _, tool := range env.ToolNames {
			out = append(out, "tool:"+strings.TrimSpace(tool))
		}
	}
	lowered := strings.ToLower(raw)
	for key, value := range intentEnvironmentHints {
		if strings.Contains(lowered, key) {
			out = append(out, value)
		}
	}
	return uniqueStrings(out)
}

func extractIntentRisks(raw string) []string {
	clauses := splitIntentClauses(raw)
	out := make([]string, 0, len(clauses))
	for _, clause := range clauses {
		lowered := strings.ToLower(clause)
		for _, marker := range intentRiskMarkers {
			if strings.Contains(lowered, strings.ToLower(marker)) {
				out = append(out, summarizeLine(clause, 120))
				break
			}
		}
	}
	return uniqueStrings(out)
}

func plannerEnvironmentFingerprint(query MemoryQuery, scope SessionScope) *DecisionEnvFingerprint {
	if scope.Environment != nil {
		env := cloneDecisionEnvFingerprint(*scope.Environment)
		return &env
	}
	if query.Environment != nil {
		env := cloneDecisionEnvFingerprint(*query.Environment)
		return &env
	}
	return nil
}

func splitIntentClauses(raw string) []string {
	normalized := strings.NewReplacer("\n", ",", "，", ",", "；", ";", "。", ",", "、", ",").Replace(strings.TrimSpace(raw))
	if normalized == "" {
		return nil
	}
	parts := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == ',' || r == ';'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		clause := strings.TrimSpace(part)
		if clause == "" {
			continue
		}
		out = append(out, clause)
	}
	return out
}

func flattenIntentTerms(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values)*2)
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
		for _, token := range strings.FieldsFunc(strings.ToLower(trimmed), func(r rune) bool {
			return unicode.IsSpace(r) || strings.ContainsRune(",.!?:;()[]{}\"'`|/\\_-", r)
		}) {
			if strings.TrimSpace(token) == "" {
				continue
			}
			out = append(out, token)
		}
	}
	return out
}

func sanitizeIntentKeyFragment(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if lastUnderscore {
			continue
		}
		b.WriteByte('_')
		lastUnderscore = true
	}
	return strings.Trim(b.String(), "_")
}

func isIntentStopFragment(fragment string) bool {
	if fragment == "" {
		return true
	}
	for _, rule := range intentActionRules {
		if fragment == rule.key {
			return true
		}
		for _, term := range rule.terms {
			if fragment == sanitizeIntentKeyFragment(term) {
				return true
			}
		}
	}
	for key := range intentEnvironmentHints {
		if fragment == sanitizeIntentKeyFragment(key) {
			return true
		}
	}
	return fragment == "and" || fragment == "with" || fragment == "into" || fragment == "that" || fragment == "this"
}
