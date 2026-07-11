package screen

import (
	"context"
	"fmt"
	"strings"
)

const mouseMoveUnsupportedActionMessage = "unsupported action: MOUSE_MOVE"
const mousePositionUnsupportedActionMessage = "unsupported action: MOUSE_POSITION"

func (t *ScreenActionTool) executeMouseMove(
	ctx context.Context,
	traceID string,
	params map[string]any,
) error {
	if _, err := t.execution.Call(ctx, "MOUSE_MOVE", params, traceID); err != nil {
		return wrapNativeExecutionErrorWithRebuildHint(
			"MOUSE_MOVE",
			err,
			mouseMoveUnsupportedActionMessage,
		)
	}
	return nil
}

func wrapNativeExecutionErrorWithRebuildHint(
	action string,
	err error,
	unsupportedActionMessage string,
) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), unsupportedActionMessage) {
		return fmt.Errorf(
			"execution %s failed: %w; rebuild drivers/native and ensure bridge uses the new native binary",
			action,
			err,
		)
	}
	return fmt.Errorf("execution %s failed: %w", action, err)
}
