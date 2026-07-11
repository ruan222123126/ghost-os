package owner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	tooladapter "ghost-os/bridge/orchestration/internal/adapters/toolregistry"
	"ghost-os/bridge/orchestration/internal/domain/group"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/taskdefs"
	"ghost-os/bridge/tools"
)

const (
	MissingDispatchMessage    = "owner turn ended without orchestration_dispatch; orchestration cannot advance yet"
	dispatchRepairTraceSuffix = "-repair"
)

type RuntimeCatalogRequest struct {
	Config      bridgeconfig.Config
	Registry    *tools.Registry
	Session     *session.Session
	GroupNode   taskdefs.OrchestrationNode
	MemberOrder []string
}

func BuildRuntimeCatalog(req RuntimeCatalogRequest) tools.ToolCatalog {
	base := tools.NewPromptOverrideCatalog(req.Registry, req.Config.ToolSelector.PromptOverrides)
	catalog := runtimeCatalog{
		base: base,
		dispatch: tooladapter.DispatchTool{
			GroupNode:   req.GroupNode,
			MemberOrder: append([]string(nil), req.MemberOrder...),
		},
	}
	staticNames := RuntimeStaticTools(req.Config, base)
	return bridgeruntime.NewSessionTurnCatalog(
		catalog,
		AppendRuntimeTool(staticNames, tooladapter.DispatchToolName),
		req.Session,
		req.Config.ToolSearch.IdleTurns,
		false,
	)
}

type runtimeCatalog struct {
	base     tools.ToolCatalog
	dispatch tools.Tool
}

func RuntimeStaticTools(cfg bridgeconfig.Config, base tools.ToolCatalog) []string {
	policy := bridgeruntime.NewToolSelectionPolicy(cfg)
	return toolCatalogNames(policy.ResidentCatalog(base))
}

func AppendRuntimeTool(names []string, toolName string) []string {
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

func (c runtimeCatalog) Get(name string) tools.Tool {
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

func (c runtimeCatalog) ToolDefs() []llm.ToolDef {
	defs := catalogDefs(c.base, c.dispatch)
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Name < defs[j].Name
	})
	return defs
}

func catalogDefs(base tools.ToolCatalog, dispatch tools.Tool) []llm.ToolDef {
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

func RunCardStatus(response string, runErr error) (string, string, string) {
	handoffErr, err := DispatchHandoffFromRunErr(runErr)
	if err != nil {
		return taskdefs.RunStatusError, "", err.Error()
	}
	if handoffErr != nil {
		return taskdefs.RunStatusSuccess, strings.TrimSpace(handoffErr.Did), ""
	}
	if runErr != nil {
		return taskdefs.RunStatusError, "", runErr.Error()
	}
	preview := strings.TrimSpace(response)
	if preview == "" {
		return taskdefs.RunStatusError, "", MissingDispatchMessage
	}
	return taskdefs.RunStatusError, preview, MissingDispatchMessage
}

type DispatchRunRequest struct {
	UserPrompt string
	TraceID    string
}

func RunDispatchWithRepair(
	ctx context.Context,
	runAgent *agent.Agent,
	req DispatchRunRequest,
	sink streaming.Sink,
) (string, error) {
	output, runErr := runAgent.RunMessageStreamWithTraceID(ctx, UserMessage(req.UserPrompt), req.TraceID, sink)
	handoffErr, err := DispatchHandoffFromRunErr(runErr)
	if err != nil {
		return "", err
	}
	if handoffErr != nil {
		return output, runErr
	}
	return repairDispatchRound(ctx, runAgent, req.TraceID, output, sink)
}

func repairDispatchRound(
	ctx context.Context,
	runAgent *agent.Agent,
	traceID string,
	previousOutput string,
	sink streaming.Sink,
) (string, error) {
	repairMessage := llm.Message{
		Role: llm.RoleUser,
		Text: BuildDispatchRepairPrompt(previousOutput),
	}
	return runAgent.RunMessageStreamWithTraceID(ctx, repairMessage, DispatchRepairTraceID(traceID), sink)
}

func BuildDispatchRepairPrompt(previousOutput string) string {
	output := strings.TrimSpace(previousOutput)
	if output == "" {
		output = "(empty response)"
	}
	return strings.TrimSpace(fmt.Sprintf(
		"Your previous reply did not advance the owner-led orchestration yet because it ended without calling `orchestration_dispatch`.\n\nPrevious reply:\n%s\n\nContinue this owner turn. You may keep working freely and use other visible tools if needed. When you are ready to advance the orchestration, call `orchestration_dispatch` once with one action: public_once, private_once, private_send, or end_group.",
		output,
	))
}

func DispatchRepairTraceID(traceID string) string {
	trimmed := strings.TrimSpace(traceID)
	if trimmed == "" {
		return "owner-dispatch" + dispatchRepairTraceSuffix
	}
	return trimmed + dispatchRepairTraceSuffix
}

func DispatchHandoffFromRunErr(runErr error) (*agent.ErrIterationHandoff, error) {
	if runErr == nil {
		return nil, nil
	}
	var handoffErr *agent.ErrIterationHandoff
	if errors.As(runErr, &handoffErr) {
		return handoffErr, nil
	}
	return nil, runErr
}

func UserMessage(userPrompt string) llm.Message {
	return llm.Message{Role: llm.RoleUser, Text: userPrompt}
}

func DecodeDecisionResponse(
	response string,
	runErr error,
	sessionID string,
) (group.DispatchCommand, string, error) {
	if runErr != nil {
		return decodeDecisionRunError(runErr, sessionID)
	}
	if strings.TrimSpace(response) == "" {
		return group.DispatchCommand{}, sessionID, errors.New(MissingDispatchMessage)
	}
	return group.DispatchCommand{}, sessionID, errors.New(MissingDispatchMessage)
}

func decodeDecisionRunError(
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
