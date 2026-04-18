package guiagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

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

func (o *DesktopOperator) Observe(
	ctx context.Context,
	request Request,
	traceID string,
) (Observation, error) {
	if o == nil || o.execution == nil {
		return Observation{}, &RunError{Code: ErrorScreenshotFailed, Message: "execution client is not configured"}
	}
	screenPayload, err := o.captureScreen(ctx, request, traceID)
	if err != nil {
		return Observation{}, err
	}
	windowPayload, err := o.execution.Call(ctx, "ACTIVE_WINDOW_INFO", nil, traceID)
	if err != nil {
		return Observation{}, &RunError{Code: ErrorActiveWindowInfo, Message: err.Error()}
	}
	return decodeObservation(screenPayload, windowPayload)
}

func (o *DesktopOperator) captureScreen(
	ctx context.Context,
	request Request,
	traceID string,
) (map[string]any, error) {
	screenParams := map[string]any{}
	if request.DisplayID != nil {
		screenParams["display_id"] = *request.DisplayID
	}
	screenPayload, err := o.execution.Call(ctx, "SCREEN_CAPTURE", screenParams, traceID)
	if err != nil {
		return nil, &RunError{Code: ErrorScreenshotFailed, Message: err.Error()}
	}
	return screenPayload, nil
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

func decodePayload(payload map[string]any, target any) error {
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, target)
}
