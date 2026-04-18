package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/tools"
)

func TestHandleFindIconTemplateUpload(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	handler := newTestHandler(t, nil)

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/tools/screen/find-icon/template",
		`{"filename":"button.png","mime_type":"image/png","data_url":"data:image/png;base64,R2hvc3Q="}`,
		nil,
	)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("unexpected status: got %d body=%s", recorder.Code, recorder.Body.String())
	}

	body := decodeResponseBody(t, recorder)
	raw, err := json.Marshal(body.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var payload bridgeorchestration.FindIconTemplateUploadPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if !strings.HasPrefix(payload.TemplatePath, filepath.Join(homeDir, ".ghost-os")) {
		t.Fatalf("unexpected template path: %q", payload.TemplatePath)
	}
	if payload.TemplateName == "" {
		t.Fatal("expected template_name to be non-empty")
	}
	if len(payload.SHA256) != 64 {
		t.Fatalf("unexpected sha256 length: %q", payload.SHA256)
	}
	if _, err := os.Stat(payload.TemplatePath); err != nil {
		t.Fatalf("stored template file does not exist: %v", err)
	}
}

func TestHandleFindIconPreview(t *testing.T) {
	handler, service, _ := newTestHandlerWithService(t, nil, nil)
	tool := &fakeScreenControlTool{
		output: `{"display_id":7,"region":{"x":1,"y":2,"width":300,"height":200},"matches":[{"score":0.99,"center":{"x":88,"y":99}}]}`,
	}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.SetRuntimeFactory(proTestRuntimeFactory{
		deps: bridgeorchestration.NewRuntimeDependencies(
			bridgeconfig.Config{},
			nil,
			registry,
			"",
			func() {},
		),
	})

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/tools/screen/find-icon/preview",
		`{"template_path":"/tmp/icon.png","threshold":0.91,"max_results":2}`,
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
	var payload bridgeorchestration.FindIconPreviewPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if !payload.Exists || payload.MatchCount != 1 {
		t.Fatalf("unexpected preview payload: %+v", payload)
	}

	if tool.callCount != 1 {
		t.Fatalf("expected one tool call, got %d", tool.callCount)
	}
	params := decodeMapValue(t, tool.lastArgs["params"], "params")
	if params["template_path"] != "/tmp/icon.png" {
		t.Fatalf("unexpected params.template_path: %+v", params)
	}
	if params["threshold"] != 0.91 || params["max_results"] != float64(2) {
		t.Fatalf("unexpected find_icon params: %+v", params)
	}
	if tool.lastArgs["mode"] != "atomic" || tool.lastArgs["action"] != "find_icon" {
		t.Fatalf("unexpected tool args envelope: %+v", tool.lastArgs)
	}
}

func TestHandleFindIconPreviewHoverAfterMatchMovesMouse(t *testing.T) {
	handler, service, _ := newTestHandlerWithService(t, nil, nil)
	tool := &fakeScreenControlTool{
		output: `{"display_id":7,"matches":[{"score":0.99,"center":{"x":88,"y":99}}]}`,
	}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.SetRuntimeFactory(proTestRuntimeFactory{
		deps: bridgeorchestration.NewRuntimeDependencies(
			bridgeconfig.Config{},
			nil,
			registry,
			"",
			func() {},
		),
	})

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/tools/screen/find-icon/preview",
		`{"template_path":"/tmp/icon.png","hover_after_match":true}`,
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
	var payload bridgeorchestration.FindIconPreviewPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if !payload.Hovered {
		t.Fatalf("expected hovered=true, got %+v", payload)
	}
	if tool.callCount != 2 {
		t.Fatalf("expected two tool calls, got %d", tool.callCount)
	}

	firstCall := tool.allArgs[0]
	if firstCall["action"] != "find_icon" {
		t.Fatalf("unexpected first action: %+v", firstCall)
	}
	secondCall := tool.allArgs[1]
	if secondCall["action"] != "click_icon" {
		t.Fatalf("unexpected second action: %+v", secondCall)
	}
	hoverParams := decodeMapValue(t, secondCall["params"], "params")
	if hoverParams["hover_only"] != true || hoverParams["x"] != float64(88) || hoverParams["y"] != float64(99) {
		t.Fatalf("unexpected hover params: %+v", hoverParams)
	}
}

type fakeScreenControlTool struct {
	output    string
	err       error
	callCount int
	lastArgs  map[string]any
	allArgs   []map[string]any
}

func (f *fakeScreenControlTool) Name() string {
	return "screen_control"
}

func (f *fakeScreenControlTool) Description() string {
	return "fake screen control"
}

func (f *fakeScreenControlTool) Parameters() json.RawMessage {
	return json.RawMessage(`{}`)
}

func (f *fakeScreenControlTool) Execute(
	_ context.Context,
	argsJSON json.RawMessage,
	_ string,
) (string, error) {
	f.callCount += 1
	f.lastArgs = map[string]any{}
	if err := json.Unmarshal(argsJSON, &f.lastArgs); err != nil {
		return "", err
	}
	snapshot := map[string]any{}
	for key, value := range f.lastArgs {
		snapshot[key] = value
	}
	f.allArgs = append(f.allArgs, snapshot)
	return f.output, f.err
}

func decodeMapValue(t *testing.T, value any, label string) map[string]any {
	t.Helper()
	record, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected %s to be object, got %T", label, value)
	}
	return record
}
