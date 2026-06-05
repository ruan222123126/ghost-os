package orchestration

import (
	"context"
	"log"
	"sort"
	"strings"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type turnSelectorCatalogs struct {
	base      tools.ToolCatalog
	selection tools.ToolCatalog
	selector  tools.ToolCatalog
}

func (p *sessionTurnPreparer) selectToolsForTurn(
	ctx context.Context,
	deps agentRuntimeDependencies,
	sess *session.Session,
	history *agent.History,
	userMessage string,
	askHumanContinuation bool,
	traceID string,
) (tools.ToolCatalog, string, error) {
	catalogs := newTurnSelectorCatalogs(deps, sess)
	if askHumanContinuation {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=ask_human_continuation", strings.TrimSpace(traceID))
		return catalogs.base, "", nil
	}
	if !deps.cfg.ToolSelector.Enabled {
		return catalogs.base, "", nil
	}
	return p.selectScopedTools(ctx, deps, sess, history, userMessage, traceID, catalogs)
}

func newTurnSelectorCatalogs(deps agentRuntimeDependencies, sess *session.Session) turnSelectorCatalogs {
	policy := bridgeruntime.NewToolSelectionPolicy(deps.cfg)
	baseCatalog := tools.NewPromptOverrideCatalog(deps.registry, deps.cfg.ToolSelector.PromptOverrides)
	residentStaticNames := toolCatalogNames(policy.ResidentCatalog(baseCatalog))
	selectorStaticNames := toolCatalogNames(policy.SelectorCatalog(baseCatalog))
	return turnSelectorCatalogs{
		base:      bridgeruntime.NewSessionTurnCatalog(baseCatalog, residentStaticNames, sess, deps.cfg.ToolSearch.IdleTurns, false),
		selection: bridgeruntime.NewSessionTurnCatalog(baseCatalog, selectorStaticNames, sess, deps.cfg.ToolSearch.IdleTurns, false),
		selector:  bridgeruntime.NewSessionTurnCatalog(baseCatalog, selectorStaticNames, sess, deps.cfg.ToolSearch.IdleTurns, true),
	}
}

func (p *sessionTurnPreparer) selectScopedTools(
	ctx context.Context,
	deps agentRuntimeDependencies,
	sess *session.Session,
	history *agent.History,
	userMessage string,
	traceID string,
	catalogs turnSelectorCatalogs,
) (tools.ToolCatalog, string, error) {
	selector := p.newSelector(deps.cfg, catalogs.selector)
	if selector == nil {
		return catalogs.base, "", nil
	}
	recentMessages := getRecentMessages(history, deps.cfg.ToolSelector.RecentMsgs)
	result := selector.SelectTools(ctx, userMessage, recentMessages, "", traceID)
	if deps.cfg.ToolSelector.Shadow {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=shadow mode=%s tools=%v confidence=%.2f fallback=%t reason=%q error=%v", strings.TrimSpace(traceID), result.Mode, result.Tools, result.Confidence, result.Fallback, result.Reason, result.Error)
		return catalogs.base, "", nil
	}
	if result.Mode != "subset" || result.Fallback {
		return catalogs.base, "", nil
	}
	return buildScopedTurnCatalog(deps, sess, catalogs.selection, result.Tools)
}

func buildScopedTurnCatalog(
	deps agentRuntimeDependencies,
	sess *session.Session,
	selectionCatalog tools.ToolCatalog,
	selected []string,
) (tools.ToolCatalog, string, error) {
	policy := bridgeruntime.NewToolSelectionPolicy(deps.cfg)
	scoped := bridgeruntime.NewSessionTurnCatalog(
		deps.registry,
		policy.Apply(toolCatalogNames(selectionCatalog), selected),
		sess,
		deps.cfg.ToolSearch.IdleTurns,
		false,
	)
	systemPrompt, err := bridgeruntime.BuildSystemPromptForSession(
		deps.cfg,
		scoped,
		sess,
		deps.cfg.ToolSearch.IdleTurns,
	)
	if err != nil {
		return nil, "", err
	}
	return scoped, systemPrompt, nil
}

func (p *sessionTurnPreparer) newSelector(cfg bridgeconfig.Config, catalog tools.ToolCatalog) bridgeruntime.SelectorEngine {
	if p != nil && p.selectorFactory != nil {
		return p.selectorFactory(cfg, catalog)
	}
	return bridgeruntime.NewSelectorFromConfig(cfg, catalog)
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

func autoResumePendingHumanTools(
	ctx context.Context,
	registry *tools.Registry,
	sess *session.Session,
	traceID string,
) error {
	if registry == nil || sess == nil || len(sess.HumanAnswers) == 0 {
		return nil
	}
	for _, questionID := range sortedAnsweredQuestionIDs(sess.HumanAnswers) {
		if err := resumeAnsweredHumanTool(ctx, registry, sess, questionID, traceID); err != nil {
			return err
		}
	}
	return nil
}

func resumeAnsweredHumanTool(
	ctx context.Context,
	registry *tools.Registry,
	sess *session.Session,
	questionID string,
	traceID string,
) error {
	question, ok := sess.PendingQuestions[questionID]
	if !ok {
		return nil
	}
	resumer := resolveHumanAnswerResumer(registry, question.ToolName)
	if resumer == nil {
		return nil
	}
	toolCtx := tools.WithToolCallID(ctx, question.ToolCallID)
	answer := sess.HumanAnswers[questionID]
	output, meta, handled, err := resumer.ResumeFromHumanAnswer(toolCtx, questionID, answer, traceID)
	if err != nil {
		return err
	}
	if !handled {
		return nil
	}
	item, ok := sess.ConsumeAnsweredQuestion(questionID)
	if !ok {
		return nil
	}
	if meta.AwaitingHuman != nil {
		return awaitingHumanError(meta.AwaitingHuman)
	}
	sess.AddMessage(agentMessageForResolvedHumanTool(
		item.Question.ToolCallID,
		item.Question.ToolName,
		item.Question.TraceID,
		output,
	))
	return nil
}

func awaitingHumanError(payload *tools.AwaitingHumanSignal) error {
	return &agent.ErrAwaitingHuman{
		QuestionID:    strings.TrimSpace(payload.QuestionID),
		Prompt:        strings.TrimSpace(payload.Prompt),
		SelectionMode: strings.TrimSpace(payload.SelectionMode),
		Options:       append([]tools.AskHumanOption(nil), payload.Options...),
	}
}

func resolveHumanAnswerResumer(
	registry *tools.Registry,
	toolName string,
) tools.HumanAnswerAutoResumer {
	if registry == nil {
		return nil
	}
	tool := registry.Get(strings.TrimSpace(toolName))
	if tool == nil {
		return nil
	}
	resumer, _ := tool.(tools.HumanAnswerAutoResumer)
	return resumer
}

func sortedAnsweredQuestionIDs(answers map[string]string) []string {
	if len(answers) == 0 {
		return nil
	}
	ids := make([]string, 0, len(answers))
	for questionID := range answers {
		if trimmed := strings.TrimSpace(questionID); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	sort.Strings(ids)
	return ids
}
