package orchestration

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	tooladapter "ghost-os/bridge/orchestration/internal/adapters/toolregistry"
	"ghost-os/bridge/orchestration/internal/ports"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
	"ghost-os/bridge/tools"
)

const (
	ownerMissingDispatchMessage    = "owner turn ended without orchestration_dispatch; orchestration cannot advance yet"
	ownerDispatchRepairTraceSuffix = "-repair"
)

type ownerRuntimeCatalog struct {
	base     tools.ToolCatalog
	dispatch tools.Tool
}

func buildOwnerRuntimeCatalog(
	deps agentRuntimeDependencies,
	sess *session.Session,
	req ports.OwnerDecisionTurnRequest,
) tools.ToolCatalog {
	base := tools.NewPromptOverrideCatalog(deps.registry, deps.cfg.ToolSelector.PromptOverrides)
	catalog := ownerRuntimeCatalog{
		base: base,
		dispatch: tooladapter.DispatchTool{
			GroupNode:   req.GroupNode,
			MemberOrder: append([]string(nil), req.MemberOrder...),
		},
	}
	staticNames := ownerRuntimeStaticTools(deps, base)
	return bridgeruntime.NewSessionTurnCatalog(
		catalog,
		appendOwnerRuntimeTool(staticNames, tooladapter.DispatchToolName),
		sess,
		deps.cfg.ToolSearch.IdleTurns,
		false,
	)
}

func ownerRuntimeStaticTools(
	deps agentRuntimeDependencies,
	base tools.ToolCatalog,
) []string {
	policy := bridgeruntime.NewToolSelectionPolicy(deps.cfg)
	return toolCatalogNames(policy.ResidentCatalog(base))
}

func appendOwnerRuntimeTool(names []string, toolName string) []string {
	trimmed := strings.TrimSpace(toolName)
	if trimmed == "" {
		return append([]string(nil), names...)
	}
	for _, name := range names {
		if strings.TrimSpace(name) == trimmed {
			return append([]string(nil), names...)
		}
	}
	return append(append([]string(nil), names...), trimmed)
}

func (c ownerRuntimeCatalog) Get(name string) tools.Tool {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil
	}
	if c.dispatch != nil && trimmed == c.dispatch.Name() {
		return c.dispatch
	}
	if c.base == nil {
		return nil
	}
	return c.base.Get(trimmed)
}

func (c ownerRuntimeCatalog) ToolDefs() []llm.ToolDef {
	defs := ownerCatalogDefs(c.base, c.dispatch)
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Name < defs[j].Name
	})
	return defs
}

func ownerCatalogDefs(base tools.ToolCatalog, dispatch tools.Tool) []llm.ToolDef {
	defs := make([]llm.ToolDef, 0, 8)
	seen := make(map[string]bool, 8)
	appendDef := func(def llm.ToolDef) {
		name := strings.TrimSpace(def.Name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		defs = append(defs, def)
	}
	if base != nil {
		for _, def := range base.ToolDefs() {
			appendDef(def)
		}
	}
	if dispatch != nil {
		appendDef(tools.ToolDefFromTool(dispatch))
	}
	return defs
}

func ownerRunCardStatus(response string, runErr error) (string, string, string) {
	handoffErr, err := ownerDispatchHandoffFromRunErr(runErr)
	if err != nil {
		return taskRunStatusError, "", err.Error()
	}
	if handoffErr != nil {
		return taskRunStatusSuccess, strings.TrimSpace(handoffErr.Did), ""
	}
	if runErr != nil {
		return taskRunStatusError, "", runErr.Error()
	}
	preview := strings.TrimSpace(response)
	if preview == "" {
		return taskRunStatusError, "", ownerMissingDispatchMessage
	}
	return taskRunStatusError, preview, ownerMissingDispatchMessage
}

func (e orchestrationOwnerDecisionTurnExecutor) startOwnerRunCard(
	ctx context.Context,
	req ports.OwnerDecisionTurnRequest,
	sess *session.Session,
) (*taskRunCardHandle, error) {
	recorder := taskRunCardRecorderFromContext(ctx)
	if recorder == nil {
		return nil, errors.New("orchestration owner task run card recorder is not configured")
	}
	title := req.OwnerNode.ID
	if req.OwnerNode.Agent != nil && strings.TrimSpace(req.OwnerNode.Agent.Title) != "" {
		title = strings.TrimSpace(req.OwnerNode.Agent.Title)
	}
	return recorder.StartCard(ctx, taskRunCardStartInput{
		kind:            bridgeTasks.RunCardKindOrchestrationOwner,
		title:           title,
		nodeID:          strings.TrimSpace(req.OwnerNode.ID),
		nodeType:        "agent",
		round:           req.Round,
		sourceSessionID: strings.TrimSpace(sess.ID),
		startedAt:       time.Now().UTC(),
	})
}

func (e orchestrationOwnerDecisionTurnExecutor) finishOwnerRunCard(
	ctx context.Context,
	handle *taskRunCardHandle,
	sessionID string,
	response string,
	runErr error,
) error {
	status, preview, errorText := ownerRunCardStatus(response, runErr)
	return handle.Finish(ctx, taskRunCardFinishInput{
		status:          status,
		preview:         preview,
		errorText:       errorText,
		sourceSessionID: strings.TrimSpace(sessionID),
		finishedAt:      time.Now().UTC(),
	})
}

func runOwnerDispatchWithRepair(
	ctx context.Context,
	req ownerDispatchRunRequest,
	runAgent *agent.Agent,
	sink streaming.Sink,
) (string, error) {
	output, runErr := runAgent.RunMessageStreamWithTraceID(ctx, ownerUserMessage(req.request), req.request.TraceID, sink)
	handoffErr, err := ownerDispatchHandoffFromRunErr(runErr)
	if err != nil {
		return "", err
	}
	if handoffErr != nil {
		return output, runErr
	}
	return repairOwnerDispatchRound(ctx, runAgent, req.request.TraceID, output, sink)
}

func repairOwnerDispatchRound(
	ctx context.Context,
	runAgent *agent.Agent,
	traceID string,
	previousOutput string,
	sink streaming.Sink,
) (string, error) {
	repairMessage := llm.Message{
		Role: llm.RoleUser,
		Text: buildOwnerDispatchRepairPrompt(previousOutput),
	}
	return runAgent.RunMessageStreamWithTraceID(ctx, repairMessage, ownerDispatchRepairTraceID(traceID), sink)
}

func buildOwnerDispatchRepairPrompt(previousOutput string) string {
	output := strings.TrimSpace(previousOutput)
	if output == "" {
		output = "(empty response)"
	}
	return strings.TrimSpace(fmt.Sprintf(
		"Your previous reply did not advance the owner-led orchestration yet because it ended without calling `orchestration_dispatch`.\n\nPrevious reply:\n%s\n\nContinue this owner turn. You may keep working freely and use other visible tools if needed. When you are ready to advance the orchestration, call `orchestration_dispatch` once with one action: public_once, private_once, private_send, or end_group.",
		output,
	))
}

func ownerDispatchRepairTraceID(traceID string) string {
	trimmed := strings.TrimSpace(traceID)
	if trimmed == "" {
		return "owner-dispatch" + ownerDispatchRepairTraceSuffix
	}
	return trimmed + ownerDispatchRepairTraceSuffix
}

func ownerDispatchHandoffFromRunErr(runErr error) (*agent.ErrIterationHandoff, error) {
	if runErr == nil {
		return nil, nil
	}
	var handoffErr *agent.ErrIterationHandoff
	if errors.As(runErr, &handoffErr) {
		return handoffErr, nil
	}
	return nil, runErr
}
