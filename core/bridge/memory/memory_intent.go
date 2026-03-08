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
	intentSelfValuePattern  = regexp.MustCompile(`(?i)\b(my|me|mine|myself|profile|identity|preference|prefer|habit|role)\b|我的|我自己|我是|我偏好|我习惯|身份|角色|偏好|习惯`)
	intentSelfRolePattern   = regexp.MustCompile(`(?i)\b(who am i|my role|identity|profile|about me)\b|我是|我的身份|我的角色|关于我`)
	intentSelfLangPattern   = regexp.MustCompile(`(?i)\b(prefer(?:red)?\s+language|language preference|favorite language)\b|偏好语言|常用语言|喜欢.*语言`)
	intentProceduralPattern = regexp.MustCompile(`(?i)\b(step|steps|workflow|runbook|recipe|checklist|migration|deploy|rollback|validate|validation)\b|步骤|流程|方案|迁移|部署|回滚|验证|检查清单`)
	intentSemanticPattern   = regexp.MustCompile(`(?i)\b(what|why|meaning|explain|summary|note|notes|doc|docs|markdown|graph|fact|knowledge)\b|解释|含义|说明|总结|笔记|文档|事实|知识`)
	intentEpisodicPattern   = regexp.MustCompile(`(?i)\b(today|yesterday|recent|recently|latest|last|earlier|just now|session|turn|conversation|history)\b|刚才|刚刚|这次|上次|最近|今天|昨天|会话|对话|历史`)
)

const (
	plannerRecallModeProceduralFirst = "procedural-first"
	plannerRecallModeSemanticFirst   = "semantic-first"
	plannerRecallModeSelfFirst       = "self-first"
	plannerRecallModeEpisodicFirst   = "episodic-first"
	plannerRecallModeMixed           = "mixed"
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
	plan.RecallMode = plannerRecallMode(query, scope, raw, plan)
	plan.Hydration = plannerHydrationLayers(plan.RecallMode)
	plan.Truth = plannerTruthQueryOptions(query, scope, raw, plan)

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
	out.RecallMode = normalizePlannerRecallMode(out.RecallMode)
	out.Hydration = uniqueStrings(out.Hydration)
	out.Truth = normalizeTruthQueryOptions(out.Truth)
	out.Terms = uniqueStrings(flattenIntentTerms(out.Terms))
	out.Confidence = clamp01(out.Confidence)
	return out
}

func normalizePlannerRecallMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case plannerRecallModeProceduralFirst, plannerRecallModeSemanticFirst, plannerRecallModeSelfFirst, plannerRecallModeEpisodicFirst:
		return strings.TrimSpace(mode)
	default:
		return plannerRecallModeMixed
	}
}

func plannerRecallMode(query MemoryQuery, scope SessionScope, raw string, plan QueryIntentPlan) string {
	action := strings.TrimPrefix(strings.TrimSpace(plan.IntentKey), "intent.")
	action, _, _ = strings.Cut(action, "_")
	procedural := 0.0
	semantic := 0.0
	self := 0.0
	episodic := 0.0

	switch action {
	case "fix", "create", "run", "plan":
		procedural += 1.8
	case "inspect", "explain", "compare", "recall":
		semantic += 1.4
	}
	if len(plan.Constraints) > 0 || len(plan.Risks) > 0 {
		procedural += 1.1
	}
	if query.IncludeDecision || query.DecisionReuseOnly || len(query.DecisionTypes) > 0 {
		procedural += 1.2
	}
	if intentProceduralPattern.MatchString(raw) {
		procedural += 0.9
	}

	if len(plan.Entities) > 0 {
		semantic += 0.6
	}
	if query.IncludeMarkdown {
		semantic += 1.0
	}
	if query.IncludeGraph {
		semantic += 0.9
	}
	if intentSemanticPattern.MatchString(raw) {
		semantic += 1.0
	}

	if plannerHasSelfSignal(raw) {
		self += 2.8
	}
	if containsString(plan.Entities, "self") || containsString(plan.Entities, "user") || containsString(plan.Entities, "我") {
		self += 0.8
	}

	if query.PreferRecent || query.TimeRange != nil {
		episodic += 1.2
	}
	if strings.TrimSpace(firstNonEmpty(metadataString(query.Metadata, "session_id"), scope.SessionID)) != "" {
		episodic += 0.9
	}
	if len(query.SessionHints) > 0 || len(query.MonthHints) > 0 {
		episodic += 0.8
	}
	if intentEpisodicPattern.MatchString(raw) {
		episodic += 1.8
	}

	bestMode := plannerRecallModeMixed
	bestScore := 1.3
	secondScore := 0.0
	for _, candidate := range []struct {
		mode  string
		score float64
	}{
		{mode: plannerRecallModeProceduralFirst, score: procedural},
		{mode: plannerRecallModeSemanticFirst, score: semantic},
		{mode: plannerRecallModeSelfFirst, score: self},
		{mode: plannerRecallModeEpisodicFirst, score: episodic},
	} {
		if candidate.score > bestScore {
			secondScore = bestScore
			bestScore = candidate.score
			bestMode = candidate.mode
			continue
		}
		if candidate.score > secondScore {
			secondScore = candidate.score
		}
	}
	if bestMode == plannerRecallModeMixed {
		return bestMode
	}
	if query.IncludeMarkdown && query.IncludeDecision && semantic >= 2.0 && procedural >= 1.5 {
		return plannerRecallModeMixed
	}
	if secondScore >= 1.5 && bestScore-secondScore < 0.75 {
		return plannerRecallModeMixed
	}
	return bestMode
}

func plannerHydrationLayers(mode string) []string {
	switch normalizePlannerRecallMode(mode) {
	case plannerRecallModeProceduralFirst:
		return []string{"decision", "warm"}
	case plannerRecallModeSemanticFirst:
		return []string{"markdown", "graph"}
	case plannerRecallModeSelfFirst:
		return []string{"decision", "markdown"}
	case plannerRecallModeEpisodicFirst:
		return []string{"hot", "warm", "cold"}
	default:
		return []string{"hot", "warm", "cold", "markdown", "decision", "graph"}
	}
}

func plannerTruthQueryOptions(query MemoryQuery, scope SessionScope, raw string, plan QueryIntentPlan) TruthQueryOptions {
	options := TruthQueryOptions{ActiveOnly: true}
	sessionID := strings.TrimSpace(firstNonEmpty(metadataString(query.Metadata, "session_id"), scope.SessionID))
	switch normalizePlannerRecallMode(plan.RecallMode) {
	case plannerRecallModeProceduralFirst:
		options.Value = firstNonEmpty(strings.TrimSpace(plan.IntentKey), firstNonEmpty(plan.Constraints...), firstNonEmpty(plan.Risks...), firstNonEmpty(plan.Entities...))
	case plannerRecallModeSemanticFirst:
		options.Value = firstNonEmpty(firstNonEmpty(plan.Entities...), strings.TrimSpace(metadataString(query.Metadata, "entity_id")))
	case plannerRecallModeSelfFirst:
		options.Predicate = plannerSelfPredicate(raw)
		options.Value = firstNonEmpty(firstNonEmpty(plan.Entities...), strings.TrimSpace(metadataString(query.Metadata, "truth_value")))
	case plannerRecallModeEpisodicFirst:
		options.Subject = sessionID
		options.Value = firstNonEmpty(strings.TrimSpace(plan.IntentKey), firstNonEmpty(plan.Entities...), firstNonEmpty(plan.Constraints...), firstNonEmpty(plan.Risks...))
		options.IncludeHistorical = true
		options.ActiveOnly = false
	default:
		options.Value = firstNonEmpty(strings.TrimSpace(plan.IntentKey), firstNonEmpty(plan.Entities...), firstNonEmpty(plan.Constraints...), firstNonEmpty(plan.Risks...))
	}
	return normalizeTruthQueryOptions(options)
}

func plannerHasSelfSignal(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if intentSelfRolePattern.MatchString(raw) || intentSelfLangPattern.MatchString(raw) {
		return true
	}
	lowered := strings.ToLower(raw)
	return intentSelfValuePattern.MatchString(raw) && (strings.Contains(lowered, "prefer") || strings.Contains(lowered, "identity") || strings.Contains(lowered, "role") || strings.Contains(raw, "偏好") || strings.Contains(raw, "习惯") || strings.Contains(raw, "身份") || strings.Contains(raw, "角色"))
}

func plannerSelfPredicate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if intentSelfLangPattern.MatchString(raw) {
		return "preference.language"
	}
	if intentSelfRolePattern.MatchString(raw) {
		return "identity.self"
	}
	if intentSelfValuePattern.MatchString(raw) {
		return "preference.has"
	}
	return ""
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
