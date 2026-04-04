package guiagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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

func (o *DesktopOperator) Observe(
	ctx context.Context,
	request Request,
	traceID string,
) (Observation, error) {
	if o == nil || o.execution == nil {
		return Observation{}, &RunError{Code: ErrorScreenshotFailed, Message: "execution client is not configured"}
	}
	screenParams := map[string]any{}
	if request.DisplayID != nil {
		screenParams["display_id"] = *request.DisplayID
	}
	screenPayload, err := o.execution.Call(ctx, "SCREEN_CAPTURE", screenParams, traceID)
	if err != nil {
		return Observation{}, &RunError{Code: ErrorScreenshotFailed, Message: err.Error()}
	}
	windowPayload, err := o.execution.Call(ctx, "ACTIVE_WINDOW_INFO", nil, traceID)
	if err != nil {
		return Observation{}, &RunError{Code: ErrorActiveWindowInfo, Message: err.Error()}
	}
	return decodeObservation(screenPayload, windowPayload)
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

func executeClick(
	o *DesktopOperator,
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	return o.executeMouseAction(ctx, "MOUSE_CLICK", request, observation, action.Target, traceID)
}

func executeDoubleClick(
	o *DesktopOperator,
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	return o.executeMouseAction(ctx, "MOUSE_DOUBLE_CLICK", request, observation, action.Target, traceID)
}

func executeRightClick(
	o *DesktopOperator,
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	return o.executeMouseAction(ctx, "MOUSE_RIGHT_CLICK", request, observation, action.Target, traceID)
}

func executeTypeText(
	o *DesktopOperator,
	ctx context.Context,
	_ Request,
	_ Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	return o.execution.Call(ctx, "TEXT_INPUT", map[string]any{"text": action.Text}, traceID)
}

func executeHotkey(
	o *DesktopOperator,
	ctx context.Context,
	_ Request,
	_ Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	return o.execution.Call(ctx, "KEY_HOTKEY", map[string]any{"keys": action.Keys}, traceID)
}

func executeScrollAction(
	o *DesktopOperator,
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	return o.executeScroll(ctx, request, observation, action, traceID)
}

func executeDragAction(
	o *DesktopOperator,
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	return o.executeDrag(ctx, request, observation, action, traceID)
}

func (o *DesktopOperator) executeMouseAction(
	ctx context.Context,
	actionName string,
	request Request,
	observation Observation,
	target *ActionTarget,
	traceID string,
) (map[string]any, error) {
	point, err := absoluteCenterPoint(observation, target)
	if err != nil {
		return nil, err
	}
	params := mouseParamsFromPoint(point, request, observation.DisplayID)
	return o.execution.Call(ctx, actionName, params, traceID)
}

func (o *DesktopOperator) executeScroll(
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	params := map[string]any{
		"delta_x": action.DeltaX,
		"delta_y": action.DeltaY,
	}
	if action.Target != nil && action.Target.Box != nil {
		point, err := absoluteCenterPoint(observation, action.Target)
		if err != nil {
			return nil, err
		}
		params["x"] = point.X
		params["y"] = point.Y
	}
	if request.DisplayID != nil {
		params["display_id"] = *request.DisplayID
	}
	return o.execution.Call(ctx, "MOUSE_SCROLL", params, traceID)
}

func (o *DesktopOperator) executeDrag(
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	start, err := absoluteCenterPoint(observation, action.Target)
	if err != nil {
		return nil, err
	}
	end, err := absoluteCenterPoint(observation, action.Destination)
	if err != nil {
		return nil, err
	}
	params := map[string]any{
		"start_x": start.X,
		"start_y": start.Y,
		"end_x":   end.X,
		"end_y":   end.Y,
	}
	if action.DurationMs > 0 {
		params["duration_ms"] = action.DurationMs
	}
	appendWindowConstraints(params, request)
	if request.DisplayID != nil {
		params["display_id"] = *request.DisplayID
	}
	return o.execution.Call(ctx, "MOUSE_DRAG", params, traceID)
}

type absolutePoint struct {
	X int
	Y int
}

type capturePayload struct {
	ImagePath   string  `json:"image_path"`
	ImageWidth  int     `json:"image_width"`
	ImageHeight int     `json:"image_height"`
	DisplayID   int     `json:"display_id"`
	ScaleX      float64 `json:"scale_x"`
	ScaleY      float64 `json:"scale_y"`
	OriginX     int     `json:"origin_x"`
	OriginY     int     `json:"origin_y"`
}

type windowInfoPayload struct {
	Title string `json:"title"`
	Class string `json:"class"`
}

func decodeObservation(screen map[string]any, window map[string]any) (Observation, error) {
	var capture capturePayload
	if err := decodePayload(screen, &capture); err != nil {
		return Observation{}, &RunError{Code: ErrorScreenshotFailed, Message: err.Error()}
	}
	var info windowInfoPayload
	if err := decodePayload(window, &info); err != nil {
		return Observation{}, &RunError{Code: ErrorActiveWindowInfo, Message: err.Error()}
	}
	imagePath := strings.TrimSpace(capture.ImagePath)
	if imagePath == "" {
		return Observation{}, &RunError{Code: ErrorScreenshotFailed, Message: "SCREEN_CAPTURE returned empty image_path"}
	}
	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return Observation{}, &RunError{Code: ErrorScreenshotFailed, Message: fmt.Sprintf("read screenshot file: %v", err)}
	}
	return Observation{
		ImageWidth:        capture.ImageWidth,
		ImageHeight:       capture.ImageHeight,
		DisplayID:         capture.DisplayID,
		ScaleX:            capture.ScaleX,
		ScaleY:            capture.ScaleY,
		OriginX:           capture.OriginX,
		OriginY:           capture.OriginY,
		ActiveWindowTitle: info.Title,
		ActiveWindowClass: info.Class,
		ImageBytes:        imageBytes,
	}, nil
}

func absoluteCenterPoint(observation Observation, target *ActionTarget) (absolutePoint, error) {
	if target == nil || target.Box == nil {
		return absolutePoint{}, &RunError{Code: ErrorInvalidTargetBox, Message: "target.box is required"}
	}
	box := *target.Box
	centerX := int(((box[0] + box[2]) / 2) * float64(observation.ImageWidth))
	centerY := int(((box[1] + box[3]) / 2) * float64(observation.ImageHeight))
	return absolutePoint{
		X: observation.OriginX + centerX,
		Y: observation.OriginY + centerY,
	}, nil
}

func mouseParamsFromPoint(point absolutePoint, request Request, displayID int) map[string]any {
	params := map[string]any{
		"x": point.X,
		"y": point.Y,
	}
	appendWindowConstraints(params, request)
	if request.DisplayID != nil {
		params["display_id"] = *request.DisplayID
	} else if displayID >= 0 {
		params["display_id"] = displayID
	}
	return params
}

func appendWindowConstraints(params map[string]any, request Request) {
	if title := strings.TrimSpace(request.Target.WindowTitle); title != "" {
		params["ensure_active_window_title"] = title
	}
	if className := strings.TrimSpace(request.Target.WindowClass); className != "" {
		params["ensure_active_window_class"] = className
	}
}

func decodePayload(payload map[string]any, target any) error {
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, target)
}
