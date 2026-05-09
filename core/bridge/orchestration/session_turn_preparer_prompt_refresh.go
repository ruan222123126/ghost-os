package orchestration

import (
	"context"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

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
