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
	agentadapter "ghost-os/bridge/orchestration/internal/adapters/agent"
	tooladapter "ghost-os/bridge/orchestration/internal/adapters/toolregistry"
	apporchestrations "ghost-os/bridge/orchestration/internal/app/orchestrations"
	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type orchestrationOwnerDecisionTurnExecutor struct {
	service *bridgeService
}

func (r orchestrationTaskRunner) ownerDecisionRunner() ports.OwnerDecisionRunner {
	return agentadapter.OwnerDecisionRunner{
		Executor: orchestrationOwnerDecisionTurnExecutor{service: r.adapter.service},
	}
}

func (e orchestrationOwnerDecisionTurnExecutor) RunOwnerDecisionTurn(
	ctx context.Context,
	req ports.OwnerDecisionTurnRequest,
) (group.DispatchCommand, string, error) {
	if e.service == nil || e.service.runtimeFactory == nil {
		return group.DispatchCommand{}, req.SessionID, errors.New("owner dispatch runtime is not configured")
	}
	deps, err := e.buildOwnerDispatchDeps(req.RuntimeOverrides)
	if err != nil {
		return group.DispatchCommand{}, req.SessionID, err
	}
	defer deps.Close()
	systemPrompt := ownerDecisionSystemPrompt(deps.systemPrompt, req)
	sess, err := e.loadOwnerDispatchSession(req.SessionID, deps, systemPrompt)
	if err != nil {
		return group.DispatchCommand{}, req.SessionID, err
	}
	response, runErr := e.runOwnerDispatchAgent(ctx, ownerDispatchRunRequest{
		request:      req,
		deps:         deps,
		sess:         sess,
		systemPrompt: systemPrompt,
	})
	if err := e.saveOwnerDispatchSession(sess); err != nil {
		return group.DispatchCommand{}, sess.ID, err
	}
	return decodeOwnerDecisionResponse(response, runErr, sess.ID)
}

func (e orchestrationOwnerDecisionTurnExecutor) buildOwnerDispatchDeps(
	runtimeOverrides *TaskRuntimeOverrides,
) (agentRuntimeDependencies, error) {
	preparer := e.newOwnerDispatchPreparer()
	deps, _, _, err := preparer.buildPrepareDependencies(runtimeOverrides)
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	return deps, nil
}

func (e orchestrationOwnerDecisionTurnExecutor) loadOwnerDispatchSession(
	sessionID string,
	deps agentRuntimeDependencies,
	systemPrompt string,
) (*session.Session, error) {
	sess, created, err := newSessionHistoryBuilder(
		deps.cfg.Provider,
		systemPrompt,
		e.service.sessionStore,
		deps.cfg.ToolSearch.IdleTurns,
		deps.cfg.MicrocompactEnabled,
		"",
	).LoadOrCreateSession(sessionID)
	if err != nil {
		return nil, err
	}
	if created {
		return sess, e.saveOwnerDispatchSession(sess)
	}
	return sess, nil
}

func (e orchestrationOwnerDecisionTurnExecutor) runOwnerDispatchAgent(
	ctx context.Context,
	req ownerDispatchRunRequest,
) (string, error) {
	preparer := e.newOwnerDispatchPreparer()
	runAgent := buildOwnerDispatchAgent(newOwnerDispatchAgentConfig(req))
	preparer.attachDynamicPromptRefresh(runAgent, req.deps, req.sess, req.request.Catalog, func() (string, error) {
		return req.systemPrompt, nil
	})
	execCtx, cleanup, err := preparer.prepareExecutionContext(ctx, req.sess, req.deps.registry, req.request.TraceID)
	if err != nil {
		return "", err
	}
	defer cleanup()
	response, runErr := runAgent.RunMessageWithTraceID(execCtx, ownerUserMessage(req.request), req.request.TraceID)
	persistOwnerAgentMessages(req.sess, runAgent)
	return response, runErr
}

func (e orchestrationOwnerDecisionTurnExecutor) saveOwnerDispatchSession(
	sess *session.Session,
) error {
	if e.service.sessionStore == nil {
		return nil
	}
	return e.service.sessionStore.Save(sess)
}

func (e orchestrationOwnerDecisionTurnExecutor) newOwnerDispatchPreparer() *sessionTurnPreparer {
	return newSessionTurnPreparer(
		e.service.runtimeFactory,
		e.service.configStore,
		e.service.sessionStore,
		e.service.runRegistry,
		nil,
	)
}

type ownerDispatchRunRequest struct {
	request      ports.OwnerDecisionTurnRequest
	deps         agentRuntimeDependencies
	sess         *session.Session
	systemPrompt string
}

type ownerDispatchAgentConfig struct {
	deps         agentRuntimeDependencies
	sess         *session.Session
	catalog      tools.ToolCatalog
	systemPrompt string
}

func buildOwnerDispatchAgent(cfg ownerDispatchAgentConfig) *agent.Agent {
	history := agent.NewHistory("")
	history.SetConversationState(cfg.sess.ConversationState)
	for _, message := range cfg.sess.Messages {
		history.Append(message)
	}
	history.UpdateSystemPrompt(cfg.systemPrompt)
	runAgent := agent.NewAgentWithHistory(cfg.deps.client, cfg.catalog, history, cfg.deps.cfg.MaxTurns)
	runAgent.SetResponseOptions(llm.CloneResponseOptions(cfg.deps.cfg.ResponseOptions))
	runAgent.SetCompletionRetryPolicy(agent.NewCompletionRetryPolicy(
		cfg.deps.cfg.LLMCompletionRetryCount,
		time.Duration(cfg.deps.cfg.LLMCompletionRetryIntervalMS)*time.Millisecond,
	))
	return runAgent
}

func newOwnerDispatchAgentConfig(req ownerDispatchRunRequest) ownerDispatchAgentConfig {
	return ownerDispatchAgentConfig{
		deps:         req.deps,
		sess:         req.sess,
		catalog:      req.request.Catalog,
		systemPrompt: req.systemPrompt,
	}
}

func ownerDecisionSystemPrompt(basePrompt string, req ports.OwnerDecisionTurnRequest) string {
	return apporchestrations.BuildOwnerControlPrompt(apporchestrations.OwnerControlPromptRequest{
		BasePrompt:       basePrompt,
		OwnerNode:        req.OwnerNode,
		GroupNode:        req.GroupNode,
		MemberNodes:      req.MemberNodes,
		MemberOrder:      append([]string(nil), req.MemberOrder...),
		PublicTranscript: req.PublicTranscript,
		LastDispatch:     req.LastDispatch,
		Round:            req.Round,
		DispatchToolName: tooladapter.DispatchToolName,
	})
}

func ownerUserMessage(req ports.OwnerDecisionTurnRequest) llm.Message {
	return llm.Message{Role: llm.RoleUser, Text: req.UserPrompt}
}

func persistOwnerAgentMessages(sess *session.Session, runAgent *agent.Agent) {
	sess.ConversationState = runAgent.GetConversationState()
	for _, message := range runAgent.GetNewMessages() {
		sess.AddMessage(message)
	}
}

func decodeOwnerDecisionResponse(
	response string,
	runErr error,
	sessionID string,
) (group.DispatchCommand, string, error) {
	if runErr != nil {
		return decodeOwnerDecisionRunError(runErr, sessionID)
	}
	if strings.TrimSpace(response) == "" {
		return group.DispatchCommand{}, sessionID, errors.New("owner must call orchestration_dispatch")
	}
	return group.DispatchCommand{}, sessionID, errors.New("owner must call orchestration_dispatch")
}

func decodeOwnerDecisionRunError(
	runErr error,
	sessionID string,
) (group.DispatchCommand, string, error) {
	var handoff *agent.ErrIterationHandoff
	if !errors.As(runErr, &handoff) {
		return group.DispatchCommand{}, sessionID, runErr
	}
	var request group.DispatchCommand
	if err := json.Unmarshal([]byte(strings.TrimSpace(handoff.Did)), &request); err != nil {
		return group.DispatchCommand{}, sessionID, fmt.Errorf("decode owner dispatch handoff: %w", err)
	}
	return request, sessionID, nil
}
