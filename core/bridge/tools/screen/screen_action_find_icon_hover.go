package screen

import (
	"context"

	"ghost-os/bridge/tools/internal/toolparams"
)

func shouldHoverAfterFindIcon(params map[string]any) bool {
	return toolparams.OptionalBool(params, "hover_after_match", false)
}

func (t *ScreenActionTool) hoverFirstFindIconMatch(
	ctx context.Context,
	traceID string,
	payload iconMatchPayload,
	params map[string]any,
) (bool, error) {
	if !shouldHoverAfterFindIcon(params) || len(payload.Matches) == 0 {
		return false, nil
	}
	if err := t.moveMouseToIconMatch(ctx, traceID, payload.Matches[0], payload, params); err != nil {
		return false, err
	}
	return true, nil
}

func (t *ScreenActionTool) moveMouseToIconMatch(
	ctx context.Context,
	traceID string,
	selected iconMatch,
	payload iconMatchPayload,
	params map[string]any,
) error {
	movePayload := map[string]any{
		"x": selected.Center.X,
		"y": selected.Center.Y,
	}
	appendDisplayIDFromIcon(payload, movePayload)
	appendActiveWindowConstraints(params, movePayload)
	return t.executeMouseMove(ctx, traceID, movePayload)
}

func applyFindIconHoverResult(result map[string]any, hovered bool) map[string]any {
	if hovered {
		result["hovered"] = true
	}
	return result
}
