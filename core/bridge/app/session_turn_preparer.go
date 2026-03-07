package app

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/memory"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

// sessionTurnPreparer 只负责单轮运行前的装配，避免 runner 继续吸收 selector / memory / inflight 细节。
type sessionTurnPreparer struct {
	runtimeFactory      AgentRuntimeFactory
	configStore         *ConfigStore
	sessionStore        *session.Store
	sharedMemoryManager *memory.MemoryManager
	runRegistry         *RunRegistry
	selectorFactory     func(Config) selectorEngine
	decisionHintBuilder func(*memory.MemoryManager, memory.SessionScope, string) (string, []memory.DecisionHit, error)
}

func newSessionTurnPreparer(
	runtimeFactory AgentRuntimeFactory,
	configStore *ConfigStore,
	sessionStore *session.Store,
	sharedMemoryManager *memory.MemoryManager,
	runRegistry *RunRegistry,
	selectorFactory func(Config) selectorEngine,
) *sessionTurnPreparer {
	if runtimeFactory == nil {
		runtimeFactory = newAgentRuntimeFactory()
	}
	return &sessionTurnPreparer{
		runtimeFactory:      runtimeFactory,
		configStore:         configStore,
		sessionStore:        sessionStore,
		sharedMemoryManager: sharedMemoryManager,
		runRegistry:         runRegistry,
		selectorFactory:     selectorFactory,
	}
}

func (p *sessionTurnPreparer) prepare(ctx context.Context, userMessage string, sessionID string, traceID string) (*sessionTurnState, error) {
	if p == nil {
		p = newSessionTurnPreparer(nil, nil, nil, nil, nil, nil)
	}
	turnStartedAt := time.Now().UTC()

	deps, err := p.runtimeFactory.Build(p.configStore)
	if err != nil {
		return nil, err
	}

	historyBuilder := newSessionHistoryBuilder(deps.cfg, deps.systemPrompt, p.sessionStore)
	persistence := newSessionTurnCommitter(p.sessionStore, p.memoryManager(deps.cfg), traceID)

	sess, err := historyBuilder.LoadOrCreateSession(sessionID)
	if err != nil {
		deps.Close()
		return nil, &sessionTurnSetupError{
			sessionID: strings.TrimSpace(sessionID),
			err:       err,
		}
	}

	execCtx, cleanup, err := p.registerRun(ctx, sess.ID, traceID)
	if err != nil {
		deps.Close()
		return nil, &sessionTurnSetupError{
			sessionID:  strings.TrimSpace(sess.ID),
			statusCode: http.StatusConflict,
			err:        err,
		}
	}

	preTurnMessages := llm.CloneMessages(sess.Messages)
	askHumanContinuation := hasAnsweredHumanResponse(sess)
	history, answeredQuestions := historyBuilder.BuildHistoryWithResolvedQuestions(sess)
	selectorEnv := buildDecisionEnvironment(deps.cfg, strings.TrimSpace(userMessage), preTurnMessages, deps.registry)
	catalog, systemPrompt := p.selectToolsForTurn(execCtx, deps, sess.ID, history, userMessage, askHumanContinuation, traceID, selectorEnv)
	if systemPrompt != "" {
		history.UpdateSystemPrompt(systemPrompt)
	}
	environment := buildDecisionEnvironment(deps.cfg, strings.TrimSpace(userMessage), preTurnMessages, catalog)
	p.injectAutoRecall(history, userMessage, sess.ID, traceID, environment, persistence.memoryManager)

	return &sessionTurnState{
		sessionStore:      p.sessionStore,
		deps:              deps,
		persistence:       persistence,
		sess:              sess,
		agent:             agent.NewAgentWithHistory(deps.client, catalog, history, deps.cfg.MaxTurns),
		execCtx:           tools.WithSession(execCtx, sess),
		traceID:           strings.TrimSpace(traceID),
		userMessage:       strings.TrimSpace(userMessage),
		preTurnMessages:   preTurnMessages,
		answeredQuestions: answeredQuestions,
		turnStartedAt:     turnStartedAt,
		environment:       environment,
		cleanup:           cleanup,
	}, nil
}

func (p *sessionTurnPreparer) selectToolsForTurn(
	ctx context.Context,
	deps agentRuntimeDependencies,
	sessionID string,
	history *agent.History,
	userMessage string,
	askHumanContinuation bool,
	traceID string,
	selectorEnv memory.DecisionEnvFingerprint,
) (tools.ToolCatalog, string) {
	if askHumanContinuation {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=ask_human_continuation", strings.TrimSpace(traceID))
		return deps.registry, ""
	}
	if !deps.cfg.ToolSelectorEnabled {
		return deps.registry, ""
	}

	selector := p.newSelector(deps.cfg)
	if selector == nil {
		return deps.registry, ""
	}

	recentMessages := getRecentMessages(history, deps.cfg.ToolSelectorRecentMsgs)
	decisionHint := p.buildDecisionSelectorHint(deps.cfg, sessionID, history, userMessage, askHumanContinuation, traceID, selectorEnv, p.memoryManager(deps.cfg))
	result := selector.SelectTools(ctx, userMessage, recentMessages, decisionHint, traceID)
	if deps.cfg.ToolSelectorShadow {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=shadow mode=%s tools=%v confidence=%.2f fallback=%t reason=%q error=%v hint_present=%t hint_chars=%d hint_lines=%d", strings.TrimSpace(traceID), result.Mode, result.Tools, result.Confidence, result.Fallback, result.Reason, result.Error, strings.TrimSpace(decisionHint) != "", len([]rune(strings.TrimSpace(decisionHint))), selectorHintLineCount(decisionHint))
		return deps.registry, ""
	}
	if result.Mode != "subset" || result.Fallback {
		return deps.registry, ""
	}

	scoped := tools.NewScopedCatalog(deps.registry, result.Tools)
	return scoped, buildSystemPromptForCatalog(deps.cfg, scoped)
}

func (p *sessionTurnPreparer) buildDecisionSelectorHint(
	cfg Config,
	sessionID string,
	history *agent.History,
	userMessage string,
	askHumanContinuation bool,
	traceID string,
	selectorEnv memory.DecisionEnvFingerprint,
	memoryManager *memory.MemoryManager,
) string {
	if askHumanContinuation || !cfg.ToolSelectorEnabled || !cfg.MemoryDecisionSelectorHintEnabled || memoryManager == nil {
		return ""
	}
	scope := memory.SessionScope{
		SessionID:   strings.TrimSpace(sessionID),
		History:     history,
		Environment: &selectorEnv,
	}
	builder := memoryManager.BuildDecisionSelectorHintWithScope
	if p != nil && p.decisionHintBuilder != nil {
		builder = func(scope memory.SessionScope, userInput string) (string, []memory.DecisionHit, error) {
			return p.decisionHintBuilder(memoryManager, scope, userInput)
		}
	}
	hint, _, err := builder(scope, userMessage)
	if err != nil {
		log.Printf("trace_id=%s action=TOOL_SELECTOR_HINT status=error session_id=%s error=%v", strings.TrimSpace(traceID), strings.TrimSpace(sessionID), err)
		return ""
	}
	return strings.TrimSpace(hint)
}

func (p *sessionTurnPreparer) newSelector(cfg Config) selectorEngine {
	if p != nil && p.selectorFactory != nil {
		return p.selectorFactory(cfg)
	}
	return newToolSelectorFromConfig(cfg)
}

func selectorHintLineCount(hint string) int {
	trimmed := strings.TrimSpace(hint)
	if trimmed == "" {
		return 0
	}
	return strings.Count(trimmed, "\n") + 1
}

func getRecentMessages(history *agent.History, limit int) []llm.Message {
	if history == nil || limit <= 0 {
		return nil
	}

	messages := history.Messages()
	filtered := make([]llm.Message, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case llm.RoleUser, llm.RoleAssistant:
			filtered = append(filtered, msg)
		}
	}
	if len(filtered) <= limit {
		return llm.CloneMessages(filtered)
	}
	return llm.CloneMessages(filtered[len(filtered)-limit:])
}

func hasAnsweredHumanResponse(sess *session.Session) bool {
	return sess != nil && len(sess.HumanAnswers) > 0
}

func (p *sessionTurnPreparer) registerRun(ctx context.Context, sessionID string, traceID string) (context.Context, func(), error) {
	if p == nil || p.runRegistry == nil {
		return ctx, func() {}, nil
	}

	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return ctx, func() {}, nil
	}

	execCtx, cancel := context.WithCancel(ctx)
	if err := p.runRegistry.Register(trimmedSessionID, strings.TrimSpace(traceID), cancel); err != nil {
		cancel()
		return ctx, func() {}, err
	}

	return execCtx, func() {
		p.runRegistry.Unregister(trimmedSessionID)
		cancel()
	}, nil
}

func (p *sessionTurnPreparer) memoryManager(cfg Config) *memory.MemoryManager {
	if p == nil {
		return nil
	}
	if p.sharedMemoryManager != nil {
		return p.sharedMemoryManager
	}

	return memory.NewMemoryManager(memoryManagerConfigFromAppConfig(cfg, p.sessionStore, newMemoryWorkerSummarizer(p.configStore)))
}

// injectAutoRecall 在执行前把 warm 层召回上下文注入为 system 消息。
func (p *sessionTurnPreparer) injectAutoRecall(history *agent.History, userMessage string, sessionID string, traceID string, environment memory.DecisionEnvFingerprint, memoryManager *memory.MemoryManager) {
	if memoryManager == nil || history == nil {
		return
	}

	contextWindow, err := memoryManager.BuildContextWindowWithScope(memory.SessionScope{
		SessionID:   sessionID,
		History:     history,
		Environment: &environment,
	}, userMessage)
	if err != nil {
		log.Printf(
			"trace_id=%s action=MEMORY_AUTO_RECALL status=error session_id=%s error=%v",
			strings.TrimSpace(traceID),
			strings.TrimSpace(sessionID),
			err,
		)
		return
	}
	for _, msg := range contextWindow {
		history.Append(msg)
	}
	if len(contextWindow) > 0 {
		log.Printf(
			"trace_id=%s action=MEMORY_AUTO_RECALL status=success session_id=%s injected=%d",
			strings.TrimSpace(traceID),
			strings.TrimSpace(sessionID),
			len(contextWindow),
		)
	}
}

func buildDecisionEnvironment(cfg Config, userMessage string, preTurnMessages []llm.Message, catalog tools.ToolCatalog) memory.DecisionEnvFingerprint {
	toolNames := toolCatalogNames(catalog)
	workspaceRoot := currentWorkspaceRoot()
	pathHints := collectDecisionPathHints(userMessage, preTurnMessages)
	return memory.DecisionEnvFingerprint{
		OS:               runtime.GOOS,
		Platform:         runtime.GOOS + "/" + runtime.GOARCH,
		Shell:            strings.TrimSpace(os.Getenv("SHELL")),
		WorkspaceRoot:    workspaceRoot,
		Provider:         string(cfg.Provider),
		Model:            strings.TrimSpace(cfg.Model),
		GraphNamespace:   strings.TrimSpace(cfg.MemoryGraphNamespace),
		Domain:           inferDecisionDomain(toolNames),
		ToolsetSignature: strings.Join(toolNames, ","),
		PathHints:        pathHints,
		TargetAppOrSite:  inferDecisionTargetAppOrSite(userMessage, pathHints),
		NativePersistent: cfg.NativePersistent,
		ToolNames:        toolNames,
	}
}

func toolCatalogNames(catalog tools.ToolCatalog) []string {
	if catalog == nil {
		return nil
	}
	defs := catalog.ToolDefs()
	if len(defs) == 0 {
		return nil
	}
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		if name := strings.TrimSpace(def.Name); name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func inferDecisionDomain(toolNames []string) string {
	seen := make(map[string]struct{}, len(toolNames))
	for _, name := range toolNames {
		seen[strings.TrimSpace(name)] = struct{}{}
	}
	for _, name := range []string{"read_file", "read_and_summarize", "search_files", "apply_diff", "bash_exec", "script_exec", "list_files"} {
		if _, ok := seen[name]; ok {
			return "coding"
		}
	}
	for _, name := range []string{"browser_action", "web_search"} {
		if _, ok := seen[name]; ok {
			return "browser"
		}
	}
	if _, ok := seen["ask_human"]; ok {
		return "general"
	}
	return ""
}

func currentWorkspaceRoot() string {
	workingDir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Clean(workingDir)
}

func collectDecisionPathHints(userMessage string, preTurnMessages []llm.Message) []string {
	texts := []string{userMessage}
	for i := len(preTurnMessages) - 1; i >= 0 && len(texts) < 5; i-- {
		msg := preTurnMessages[i]
		if msg.Role != llm.RoleUser && msg.Role != llm.RoleAssistant {
			continue
		}
		if text := strings.TrimSpace(msg.Text); text != "" {
			texts = append(texts, text)
		}
	}
	seen := make(map[string]struct{}, 8)
	out := make([]string, 0, 6)
	for _, text := range texts {
		for _, token := range strings.Fields(text) {
			candidate := strings.Trim(token, `"'(),;[]{}<>`)
			if !looksLikeDecisionPathHint(candidate) {
				continue
			}
			candidate = strings.TrimSpace(candidate)
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			out = append(out, candidate)
			if len(out) >= 6 {
				return out
			}
		}
	}
	return out
}

func looksLikeDecisionPathHint(token string) bool {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return true
	}
	if strings.HasPrefix(trimmed, "~/") || strings.HasPrefix(trimmed, "./") || strings.HasPrefix(trimmed, "../") {
		return true
	}
	if strings.Contains(trimmed, "/") {
		return true
	}
	return strings.Contains(trimmed, ".") && len(trimmed) > 3
}

func inferDecisionTargetAppOrSite(userMessage string, pathHints []string) string {
	if host := extractDecisionURLHost(userMessage); host != "" {
		return host
	}
	for _, hint := range pathHints {
		trimmed := strings.TrimSpace(hint)
		if strings.HasPrefix(trimmed, "apps/") || strings.HasPrefix(trimmed, "core/") || strings.HasPrefix(trimmed, "drivers/") {
			return trimmed
		}
	}
	return ""
}

func extractDecisionURLHost(text string) string {
	for _, token := range strings.Fields(text) {
		trimmed := strings.Trim(token, `"'(),;[]{}<>`)
		if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
			continue
		}
		parsed, err := url.Parse(trimmed)
		if err != nil {
			continue
		}
		if host := strings.TrimSpace(parsed.Host); host != "" {
			return host
		}
	}
	return ""
}
