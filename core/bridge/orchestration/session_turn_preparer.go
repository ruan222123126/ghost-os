package orchestration

import (
	"context"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/memoryaug"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

// sessionTurnPreparer 只负责单轮运行前的装配，避免 runner 继续吸收 selector / memory / inflight 细节。
type sessionTurnPreparer struct {
	runtimeFactory  AgentRuntimeFactory
	configStore     *ConfigStore
	sessionStore    *session.Store
	runRegistry     *RunRegistry
	selectorFactory func(Config, tools.ToolCatalog) selectorEngine
}

func newSessionTurnPreparer(
	runtimeFactory AgentRuntimeFactory,
	configStore *ConfigStore,
	sessionStore *session.Store,
	runRegistry *RunRegistry,
	selectorFactory func(Config, tools.ToolCatalog) selectorEngine,
) *sessionTurnPreparer {
	if runtimeFactory == nil {
		runtimeFactory = newAgentRuntimeFactory()
	}
	return &sessionTurnPreparer{
		runtimeFactory:  runtimeFactory,
		configStore:     configStore,
		sessionStore:    sessionStore,
		runRegistry:     runRegistry,
		selectorFactory: selectorFactory,
	}
}

func (p *sessionTurnPreparer) prepare(ctx context.Context, userMessage string, sessionID string, traceID string) (*sessionTurnState, error) {
	if p == nil {
		p = newSessionTurnPreparer(nil, nil, nil, nil, nil)
	}
	turnStartedAt := time.Now().UTC()
	rawUserMessage := userMessage
	trimmedUserMessage := strings.TrimSpace(userMessage)
	trimmedTraceID := strings.TrimSpace(traceID)

	deps, err := p.runtimeFactory.Build(p.configStore)
	if err != nil {
		return nil, err
	}
	historyBuilder := newSessionHistoryBuilder(deps.cfg.Provider, deps.systemPrompt, p.sessionStore)
	persistence := newSessionTurnCommitter(p.sessionStore, deps.memoryLearn)

	sess, err := historyBuilder.LoadOrCreateSession(sessionID)
	if err != nil {
		deps.Close()
		return nil, &sessionTurnSetupError{
			sessionID: strings.TrimSpace(sessionID),
			err:       err,
		}
	}

	execCtx, cleanup, err := p.registerRun(ctx, sess.ID, trimmedTraceID)
	if err != nil {
		deps.Close()
		return nil, &sessionTurnSetupError{
			sessionID:  strings.TrimSpace(sess.ID),
			statusCode: http.StatusConflict,
			err:        err,
		}
	}

	history, preTurnMessages, catalog, err := p.prepareHistoryAndEnvironment(execCtx, deps, historyBuilder, sess, rawUserMessage, trimmedUserMessage, trimmedTraceID)
	if err != nil {
		cleanup()
		deps.Close()
		return nil, &sessionTurnSetupError{
			sessionID: strings.TrimSpace(sess.ID),
			err:       err,
		}
	}

	return &sessionTurnState{
		sessionStore:    p.sessionStore,
		deps:            deps,
		persistence:     persistence,
		sess:            sess,
		agent:           agent.NewAgentWithHistory(deps.client, catalog, history, deps.cfg.MaxTurns),
		execCtx:         tools.WithSession(execCtx, sess),
		traceID:         trimmedTraceID,
		userMessage:     trimmedUserMessage,
		preTurnMessages: preTurnMessages,
		turnStartedAt:   turnStartedAt,
		cleanup:         cleanup,
	}, nil
}

func (p *sessionTurnPreparer) prepareHistoryAndEnvironment(
	ctx context.Context,
	deps agentRuntimeDependencies,
	historyBuilder *SessionHistoryBuilder,
	sess *session.Session,
	rawUserMessage string,
	trimmedUserMessage string,
	traceID string,
) (*agent.History, []llm.Message, tools.ToolCatalog, error) {
	preTurnMessages := llm.CloneMessages(sess.Messages)
	askHumanContinuation := hasAnsweredHumanResponse(sess)
	history := historyBuilder.BuildHistory(sess)

	catalog, systemPrompt, err := p.selectToolsForTurn(ctx, deps, sess.ID, history, rawUserMessage, askHumanContinuation, traceID)
	if err != nil {
		return nil, nil, nil, err
	}
	finalPrompt, err := p.buildSystemPromptWithMemory(ctx, deps, sess.ID, trimmedUserMessage, systemPrompt)
	if err != nil {
		return nil, nil, nil, err
	}
	if finalPrompt != "" {
		history.UpdateSystemPrompt(finalPrompt)
	}

	return history, preTurnMessages, catalog, nil
}

func (p *sessionTurnPreparer) buildSystemPromptWithMemory(
	ctx context.Context,
	deps agentRuntimeDependencies,
	sessionID string,
	userMessage string,
	systemPrompt string,
) (string, error) {
	basePrompt := strings.TrimSpace(systemPrompt)
	if basePrompt == "" {
		basePrompt = strings.TrimSpace(deps.systemPrompt)
	}
	memoryBlock, err := p.buildMemoryBlock(ctx, deps, sessionID, userMessage)
	if err != nil {
		return "", err
	}
	if memoryBlock == "" {
		return basePrompt, nil
	}
	if basePrompt == "" {
		return memoryBlock, nil
	}
	return basePrompt + "\n\n" + memoryBlock, nil
}

func (p *sessionTurnPreparer) buildMemoryBlock(
	ctx context.Context,
	deps agentRuntimeDependencies,
	sessionID string,
	userMessage string,
) (string, error) {
	if deps.memoryRecall == nil {
		return "", nil
	}
	items, err := deps.memoryRecall.Recall(ctx, memoryaug.RecallInput{
		SessionID: sessionID,
		Query:     userMessage,
	})
	if err != nil {
		return "", err
	}
	return memoryaug.FormatPromptBlock(items), nil
}

func (p *sessionTurnPreparer) selectToolsForTurn(
	ctx context.Context,
	deps agentRuntimeDependencies,
	sessionID string,
	history *agent.History,
	userMessage string,
	askHumanContinuation bool,
	traceID string,
) (tools.ToolCatalog, string, error) {
	policy := newToolSelectionPolicy(deps.cfg.ToolSelector)
	baseCatalog := policy.scopeCatalog(deps.registry)
	if askHumanContinuation {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=ask_human_continuation", strings.TrimSpace(traceID))
		return baseCatalog, "", nil
	}
	if !deps.cfg.ToolSelector.Enabled {
		return baseCatalog, "", nil
	}

	selector := p.newSelector(deps.cfg, baseCatalog)
	if selector == nil {
		return baseCatalog, "", nil
	}

	recentMessages := getRecentMessages(history, deps.cfg.ToolSelector.RecentMsgs)
	result := selector.SelectTools(ctx, userMessage, recentMessages, "", traceID)
	if deps.cfg.ToolSelector.Shadow {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=shadow mode=%s tools=%v confidence=%.2f fallback=%t reason=%q error=%v", strings.TrimSpace(traceID), result.Mode, result.Tools, result.Confidence, result.Fallback, result.Reason, result.Error)
		return baseCatalog, "", nil
	}
	if result.Mode != "subset" || result.Fallback {
		return baseCatalog, "", nil
	}

	scoped := tools.NewScopedCatalog(baseCatalog, policy.apply(toolCatalogNames(baseCatalog), result.Tools))
	systemPrompt, err := buildSystemPromptForCatalog(deps.cfg, scoped)
	if err != nil {
		return nil, "", err
	}
	return scoped, systemPrompt, nil
}

func (p *sessionTurnPreparer) newSelector(cfg Config, catalog tools.ToolCatalog) selectorEngine {
	if p != nil && p.selectorFactory != nil {
		return p.selectorFactory(cfg, catalog)
	}
	return newToolSelectorFromConfig(cfg, catalog)
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
