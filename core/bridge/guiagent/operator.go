package guiagent

import (
	"context"
	"fmt"
	"strings"
)

type ExecutionClient interface {
	Call(ctx context.Context, action string, params map[string]any, traceID string) (map[string]any, error)
}

type Operator interface {
	Observe(ctx context.Context, request Request, traceID string) (Observation, error)
	Execute(ctx context.Context, request Request, observation Observation, action Action, traceID string) (map[string]any, error)
}

type DesktopOperator struct {
	execution ExecutionClient
}

func NewDesktopOperator(client ExecutionClient) *DesktopOperator {
	return &DesktopOperator{execution: client}
}

type desktopActionHandler func(
	*DesktopOperator,
	context.Context,
	Request,
	Observation,
	Action,
	string,
) (map[string]any, error)

var desktopActionHandlers = map[string]desktopActionHandler{
	ActionClick:       executeClick,
	ActionDoubleClick: executeDoubleClick,
	ActionRightClick:  executeRightClick,
	ActionTypeText:    executeTypeText,
	ActionHotkey:      executeHotkey,
	ActionScroll:      executeScrollAction,
	ActionDrag:        executeDragAction,
}

func (o *DesktopOperator) Execute(
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	if o == nil || o.execution == nil {
		return nil, &RunError{Code: ErrorOperatorExecute, Message: "execution client is not configured"}
	}
	actionType := strings.TrimSpace(action.Type)
	handler, ok := desktopActionHandlers[actionType]
	if !ok {
		return nil, &RunError{Code: ErrorOperatorExecute, Message: fmt.Sprintf("unsupported action %q", actionType)}
	}
	return handler(o, ctx, request, observation, action, traceID)
}
