package orchestration

import (
	"context"
	"strings"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

func (p *sessionTurnPreparer) buildCompletionSystemPrompt(
	deps agentRuntimeDependencies,
	sess *session.Session,
	catalog tools.ToolCatalog,
	systemPrompt string,
) (string, error) {
	if deps.systemPromptOverride {
		return strings.TrimSpace(deps.systemPrompt), nil
	}
	if deps.systemPromptFiles != nil {
		prompt, err := bridgeruntime.BuildSystemPromptForSessionWithFiles(
			deps.cfg,
			catalog,
			sess,
			deps.cfg.ToolSearch.IdleTurns,
			*deps.systemPromptFiles,
		)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(prompt), nil
	}
	basePrompt := strings.TrimSpace(systemPrompt)
	if basePrompt == "" {
		prompt, err := bridgeruntime.BuildSystemPromptForSession(
			deps.cfg,
			catalog,
			sess,
			deps.cfg.ToolSearch.IdleTurns,
		)
		if err != nil {
			return "", err
		}
		basePrompt = strings.TrimSpace(prompt)
	}
	if basePrompt == "" {
		basePrompt = strings.TrimSpace(deps.systemPrompt)
	}
	return basePrompt, nil
}

func (p *sessionTurnPreparer) buildTurnSystemPrompt(
	deps agentRuntimeDependencies,
	sess *session.Session,
	catalog tools.ToolCatalog,
	systemPrompt string,
) (string, error) {
	return p.buildCompletionSystemPrompt(deps, sess, catalog, systemPrompt)
}

func (p *sessionTurnPreparer) attachCompletionPromptRefresh(
	runAgent *agent.Agent,
	deps agentRuntimeDependencies,
	sess *session.Session,
	catalog tools.ToolCatalog,
) {
	p.attachDynamicPromptRefresh(runAgent, deps, sess, catalog, func() (string, error) {
		return p.buildCompletionSystemPrompt(deps, sess, catalog, "")
	})
}

func (p *sessionTurnPreparer) attachDynamicPromptRefresh(
	runAgent *agent.Agent,
	deps agentRuntimeDependencies,
	sess *session.Session,
	catalog tools.ToolCatalog,
	promptBuilder func() (string, error),
) {
	if p == nil || runAgent == nil || sess == nil {
		return
	}
	runAgent.SetBeforeCompletionHook(func(_ context.Context, _ int, history *agent.History) error {
		if history == nil {
			return nil
		}
		pruneInvisibleSessionSkills(deps.cfg, sess)
		if promptBuilder == nil {
			return nil
		}
		prompt, err := promptBuilder()
		if err != nil {
			return err
		}
		history.UpdateSystemPrompt(prompt)
		return nil
	})
}

func pruneInvisibleSessionSkills(cfg bridgeconfig.Config, sess *session.Session) {
	if sess == nil {
		return
	}
	sess.PruneInvisibleDynamicSkills(bridgeruntime.VisibleSkillNames(cfg))
}
