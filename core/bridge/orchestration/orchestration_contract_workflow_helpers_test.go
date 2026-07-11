package orchestration

import (
	"context"
	"math/rand/v2"
	"time"

	runtimeadapter "ghost-os/bridge/orchestration/internal/adapters/runtime"
	workflowadapter "ghost-os/bridge/orchestration/internal/adapters/workflow"
	appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"
)

const (
	workflowScreenControlAtomicMode            = appworkflows.ScreenControlAtomicMode
	workflowScreenControlStepsKey              = appworkflows.ScreenControlWorkflowStepsKey
	workflowScreenControlActionKey             = appworkflows.ScreenControlActionKey
	workflowScreenControlParamsKey             = appworkflows.ScreenControlParamsKey
	workflowScreenControlCoordinateRefKey      = appworkflows.ScreenControlCoordinateRefKey
	workflowScreenControlFindIconCoordinateRef = appworkflows.ScreenControlFindIconRef
	workflowScreenControlStepDelayMinMS        = appworkflows.ScreenControlStepDelayMinMS
	workflowScreenControlStepDelayMaxMS        = appworkflows.ScreenControlStepDelayMaxMS
	workflowFindIconDataURLParam               = appworkflows.FindIconDataURLParam
	workflowLegacyFindIconDataURLParam         = appworkflows.LegacyFindIconDataURLParam
	workflowLegacyFindIconDataURLAlias         = appworkflows.LegacyFindIconDataURLAlias
	workflowFindIconDefaultTemplateName        = appworkflows.FindIconDefaultTemplateName
	workflowFindIconLegacyDataURLMessage       = appworkflows.LegacyFindIconDataURLMessage
)

type workflowNodeOutcome struct {
	status        string
	sessionID     string
	preview       string
	outputText    string
	outputValue   any
	inputSnapshot any
	err           error
}

func executeWorkflowToolNode(
	ctx context.Context,
	deps agentRuntimeDependencies,
	node WorkflowNode,
	traceID string,
) workflowNodeOutcome {
	outcome := appworkflows.ExecuteToolNode(
		ctx,
		runtimeadapter.ToWorkflowDependencies(deps),
		node,
		traceID,
		workflowadapter.TemplateUploader{},
	)
	return fromAppWorkflowOutcome(outcome)
}

func prepareWorkflowToolArguments(toolName string, arguments map[string]any) (map[string]any, error) {
	return appworkflows.PrepareToolArguments(toolName, arguments, workflowadapter.TemplateUploader{})
}

func workflowMapString(record map[string]any, key string) string {
	raw, ok := record[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}

func randomWorkflowScreenControlStepDelay() time.Duration {
	span := workflowScreenControlStepDelayMaxMS - workflowScreenControlStepDelayMinMS
	delayMS := workflowScreenControlStepDelayMinMS
	if span > 0 {
		delayMS += rand.IntN(span + 1)
	}
	return time.Duration(delayMS) * time.Millisecond
}

func fromAppWorkflowOutcome(outcome appworkflows.NodeOutcome) workflowNodeOutcome {
	return workflowNodeOutcome{
		status:        outcome.Status,
		sessionID:     outcome.SessionID,
		preview:       outcome.Preview,
		outputText:    outcome.OutputText,
		outputValue:   outcome.OutputValue,
		inputSnapshot: outcome.InputSnapshot,
		err:           outcome.Err,
	}
}
