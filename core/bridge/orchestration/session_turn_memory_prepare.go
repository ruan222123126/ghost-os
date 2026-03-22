package orchestration

import (
	"context"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/memoryaug"
	"ghost-os/bridge/session"
)

type turnMemoryContext struct {
	Decision memoryaug.PlannerDecision
	Recall   memoryaug.RecallOutput
}

func (p *sessionTurnPreparer) prepareTurnMemory(
	ctx context.Context,
	deps agentRuntimeDependencies,
	sess *session.Session,
	history *agent.History,
	userMessage string,
) (*turnMemoryContext, string, error) {
	if deps.memoryPlan == nil || deps.memoryRecall == nil {
		return nil, "", nil
	}
	sessionID := ""
	if sess != nil {
		sessionID = strings.TrimSpace(sess.ID)
	}
	decision, err := deps.memoryPlan.Plan(ctx, memoryaug.PlannerInput{
		SessionID:      sessionID,
		UserScope:      deps.cfg.MemoryAugmentation.UserScopeID,
		UserMessage:    userMessage,
		RecentMessages: buildPlannerRecentMessages(history, ""),
		ProjectRoot:    deps.cfg.ProjectRoot,
	})
	if err != nil {
		return nil, "", err
	}
	recall, err := deps.memoryRecall.Recall(ctx, memoryaug.RecallInput{
		SessionID:      sessionID,
		PrimaryEventID: decision.PrimaryEvent.EventID,
		ActiveEventIDs: decision.RecallPlan.EventIDs,
		FocusText:      userMessage,
		RecallPlan:     decision.RecallPlan,
	})
	if err != nil {
		return nil, "", err
	}
	return &turnMemoryContext{
		Decision: decision,
		Recall:   recall,
	}, recall.PromptBlock, nil
}
