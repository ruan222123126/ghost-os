package tools

import (
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/tools/internal/tooljson"
)

func decodeScreenCapturePayload(payload map[string]any) (screenCapturePayload, error) {
	capture, err := tooljson.DecodePayload[screenCapturePayload](payload)
	if err != nil {
		return screenCapturePayload{}, err
	}
	if err := validateScreenCapturePayload(capture, payload); err != nil {
		return screenCapturePayload{}, err
	}
	return capture, nil
}

func validateScreenCapturePayload(capture screenCapturePayload, payload map[string]any) error {
	if strings.TrimSpace(capture.ImagePath) != "" {
		return nil
	}
	if _, legacy := payload["image_base64"]; legacy {
		return fmt.Errorf(
			"execution SCREEN_CAPTURE returned deprecated image_base64 payload without image_path; rebuild drivers/native and ensure bridge uses the new native binary",
		)
	}
	return fmt.Errorf(
		"execution SCREEN_CAPTURE returned invalid payload: image_path is empty (fields: %s)",
		payloadFields(payload),
	)
}

func payloadFields(payload map[string]any) string {
	if len(payload) == 0 {
		return "<none>"
	}
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, strings.TrimSpace(key))
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}
