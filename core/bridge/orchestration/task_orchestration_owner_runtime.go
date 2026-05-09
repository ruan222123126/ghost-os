package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

func (r orchestrationTaskRunner) runOwnerDispatch(
	ctx context.Context,
	groupNode OrchestrationNode,
	memberOrder []string,
	publicTranscript orchestrationTranscript,
	lastDispatch orchestrationDispatchResult,
	sessionID string,
	round int,
) (orchestrationDispatchRequest, string, error) {
	ownerNode, ok := r.plan.nodes[groupNode.Group.OwnerAgentID]
	if !ok || ownerNode.Agent == nil {
		return orchestrationDispatchRequest{}, sessionID, fmt.Errorf("owner agent %q is not available", groupNode.Group.OwnerAgentID)
	}
	registry := tools.NewRegistry()
	registry.Register(r.newOwnerDispatchTool(groupNode, memberOrder))
	catalog := tools.NewScopedCatalog(registry, []string{orchestrationDispatchToolName})
	return r.runOwnerDispatchTurn(ctx, ownerNode, groupNode, catalog, publicTranscript, lastDispatch, sessionID, round)
}

func (r orchestrationTaskRunner) runOwnerDispatchTurn(
	ctx context.Context,
	ownerNode OrchestrationNode,
	groupNode OrchestrationNode,
	catalog tools.ToolCatalog,
	publicTranscript orchestrationTranscript,
	lastDispatch orchestrationDispatchResult,
	sessionID string,
	round int,
) (orchestrationDispatchRequest, string, error) {
	if r.adapter.service == nil || r.adapter.service.runtimeFactory == nil {
		return orchestrationDispatchRequest{}, sessionID, errors.New("owner dispatch runtime is not configured")
	}
	deps, err := r.buildOwnerDispatchDeps(ownerNode)
	if err != nil {
		return orchestrationDispatchRequest{}, sessionID, err
	}
	defer deps.Close()
	systemPrompt := buildOwnerControlPrompt(
		deps.systemPrompt,
		ownerNode,
		groupNode,
		r.plan,
		publicTranscript,
		lastDispatch,
		round,
	)

	sess, created, err := newSessionHistoryBuilder(
		deps.cfg.Provider,
		systemPrompt,
		r.adapter.service.sessionStore,
		deps.cfg.ToolSearch.IdleTurns,
		deps.cfg.MicrocompactEnabled,
		"",
	).LoadOrCreateSession(sessionID)
	if err != nil {
		return orchestrationDispatchRequest{}, sessionID, err
	}
	if created && r.adapter.service.sessionStore != nil {
		if err := r.adapter.service.sessionStore.Save(sess); err != nil {
			return orchestrationDispatchRequest{}, sessionID, err
		}
	}

	preparer := newSessionTurnPreparer(
		r.adapter.service.runtimeFactory,
		r.adapter.service.configStore,
		r.adapter.service.sessionStore,
		r.adapter.service.runRegistry,
		nil,
	)
	history := agent.NewHistory("")
	history.SetConversationState(sess.ConversationState)
	for _, message := range sess.Messages {
		history.Append(message)
	}
	history.UpdateSystemPrompt(systemPrompt)
	runAgent := agent.NewAgentWithHistory(deps.client, catalog, history, deps.cfg.MaxTurns)
	runAgent.SetResponseOptions(llm.CloneResponseOptions(deps.cfg.ResponseOptions))
	runAgent.SetCompletionRetryPolicy(agent.NewCompletionRetryPolicy(
		deps.cfg.LLMCompletionRetryCount,
		time.Duration(deps.cfg.LLMCompletionRetryIntervalMS)*time.Millisecond,
	))
	preparer.attachDynamicPromptRefresh(runAgent, deps, sess, catalog, func() (string, error) {
		return systemPrompt, nil
	})

	userMessage := llm.Message{Role: llm.RoleUser, Text: buildOwnerControlUserPrompt(round)}
	execCtx, cleanup, err := preparer.prepareExecutionContext(ctx, sess, deps.registry, r.traceID)
	if err != nil {
		return orchestrationDispatchRequest{}, sessionID, err
	}
	defer cleanup()
	response, runErr := runAgent.RunMessageWithTraceID(execCtx, userMessage, r.traceID)
	newMessages := runAgent.GetNewMessages()
	sess.ConversationState = runAgent.GetConversationState()
	for _, message := range newMessages {
		sess.AddMessage(message)
	}
	if r.adapter.service.sessionStore != nil {
		if saveErr := r.adapter.service.sessionStore.Save(sess); saveErr != nil {
			return orchestrationDispatchRequest{}, sessionID, saveErr
		}
	}
	if runErr != nil {
		var handoff *agent.ErrIterationHandoff
		if errors.As(runErr, &handoff) {
			var request orchestrationDispatchRequest
			if err := json.Unmarshal([]byte(strings.TrimSpace(handoff.Did)), &request); err != nil {
				return orchestrationDispatchRequest{}, sess.ID, fmt.Errorf("decode owner dispatch handoff: %w", err)
			}
			return request, sess.ID, nil
		}
		return orchestrationDispatchRequest{}, sess.ID, runErr
	}
	if strings.TrimSpace(response) == "" {
		return orchestrationDispatchRequest{}, sess.ID, errors.New("owner must call orchestration_dispatch")
	}
	return orchestrationDispatchRequest{}, sess.ID, errors.New("owner must call orchestration_dispatch")
}

func (r orchestrationTaskRunner) buildOwnerDispatchDeps(ownerNode OrchestrationNode) (agentRuntimeDependencies, error) {
	if r.adapter.service == nil || r.adapter.service.runtimeFactory == nil {
		return agentRuntimeDependencies{}, errors.New("owner dispatch runtime is not configured")
	}
	overrides := cloneTaskRuntimeOverrides(ownerNode.Agent.RuntimeOverrides)
	normalized, err := normalizeTaskRuntimeOverrides(overrides)
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	preparer := newSessionTurnPreparer(
		r.adapter.service.runtimeFactory,
		r.adapter.service.configStore,
		r.adapter.service.sessionStore,
		r.adapter.service.runRegistry,
		nil,
	)
	deps, _, _, err := preparer.buildPrepareDependencies(normalized)
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	return deps, nil
}
