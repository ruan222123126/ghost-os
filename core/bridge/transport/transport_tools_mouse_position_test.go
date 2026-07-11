package transport

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/tools"
)

func TestHandleMousePosition(t *testing.T) {
	handler, service, _ := newTestHandlerWithService(t, nil, nil)
	tool := &fakeScreenControlTool{
		output: `{"x":320,"y":640,"display_id":3,"scale_x":2,"scale_y":2}`,
	}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.SetRuntimeFactory(proTestRuntimeFactory{
		deps: bridgeorchestration.NewRuntimeDependencies(
			bridgeconfig.Config{PromptsDir: os.Getenv("GHOST_PROMPTS_DIR")},
			nil,
			registry,
			"",
			func() {},
		),
	})

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/tools/screen/mouse-position",
		`{}`,
		nil,
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", recorder.Code, recorder.Body.String())
	}

	body := decodeResponseBody(t, recorder)
	raw, err := json.Marshal(body.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var payload bridgeorchestration.MousePositionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.X != 320 || payload.Y != 640 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.DisplayID == nil || *payload.DisplayID != 3 {
		t.Fatalf("unexpected display id: %+v", payload)
	}
	if tool.callCount != 1 {
		t.Fatalf("expected one tool call, got %d", tool.callCount)
	}
	if tool.lastArgs["mode"] != "atomic" || tool.lastArgs["action"] != "mouse_position" {
		t.Fatalf("unexpected tool args envelope: %+v", tool.lastArgs)
	}
}
