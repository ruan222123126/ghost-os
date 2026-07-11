package runtime

import (
	"context"
	"encoding/json"
	"testing"

	"ghost-os/bridge/tools"
)

type mockRuntimeExecutionClient struct {
	callFunc func(
		ctx context.Context,
		action string,
		params map[string]any,
		traceID string,
	) (map[string]any, error)
}

func (m mockRuntimeExecutionClient) Call(
	ctx context.Context,
	action string,
	params map[string]any,
	traceID string,
) (map[string]any, error) {
	return m.callFunc(ctx, action, params, traceID)
}

func TestRegisterCoreToolsRoutesScreenActionsWithoutWorkspaceCWD(t *testing.T) {
	workspaceCalls := make([]string, 0, 1)
	interactionCalls := make([]string, 0, 2)
	registry := tools.NewRegistry()

	registerCoreTools(coreToolOptions{
		cfg:      Config{},
		registry: registry,
		resources: runtimeToolResources{
			workspaceExecutionClient: mockRuntimeExecutionClient{
				callFunc: func(
					_ context.Context,
					action string,
					_ map[string]any,
					_ string,
				) (map[string]any, error) {
					workspaceCalls = append(workspaceCalls, action)
					switch action {
					case "LIST_FILES":
						return map[string]any{
							"path":    ".",
							"entries": []any{"core/"},
						}, nil
					default:
						t.Fatalf("unexpected workspace action: %s", action)
						return nil, nil
					}
				},
			},
			interactionExecutionClient: mockRuntimeExecutionClient{
				callFunc: func(
					_ context.Context,
					action string,
					_ map[string]any,
					_ string,
				) (map[string]any, error) {
					interactionCalls = append(interactionCalls, action)
					switch action {
					case "SCREEN_CAPTURE":
						return map[string]any{
							"image_path":   "/tmp/capture.png",
							"image_width":  100,
							"image_height": 80,
							"display_id":   1,
							"scale_x":      1,
							"scale_y":      1,
							"origin_x":     0,
							"origin_y":     0,
							"region": map[string]any{
								"x":      0,
								"y":      0,
								"width":  100,
								"height": 80,
							},
						}, nil
					case "TEMPLATE_MATCH_IMAGE":
						return map[string]any{"matches": []any{}}, nil
					default:
						t.Fatalf("unexpected interaction action: %s", action)
						return nil, nil
					}
				},
			},
		},
	})

	listFiles := registry.Get("list_files")
	if listFiles == nil {
		t.Fatal("expected list_files tool")
	}
	if _, err := listFiles.Execute(
		context.Background(),
		json.RawMessage(`{"path":"."}`),
		"trace-list",
	); err != nil {
		t.Fatalf("list_files execute: %v", err)
	}

	screenControl := registry.Get("screen_control")
	if screenControl == nil {
		t.Fatal("expected screen_control tool")
	}
	if _, err := screenControl.Execute(
		context.Background(),
		json.RawMessage(`{"action":"find_icon","params":{"template_path":"/tmp/icon.png"}}`),
		"trace-screen",
	); err != nil {
		t.Fatalf("screen_control execute: %v", err)
	}

	assertRuntimeActionCalls(t, workspaceCalls, "LIST_FILES")
	assertRuntimeActionCalls(t, interactionCalls, "SCREEN_CAPTURE", "TEMPLATE_MATCH_IMAGE")
}

func assertRuntimeActionCalls(t *testing.T, calls []string, want ...string) {
	t.Helper()

	if len(calls) != len(want) {
		t.Fatalf("unexpected action count: got %v want %v", calls, want)
	}
	for index := range want {
		if calls[index] != want[index] {
			t.Fatalf("unexpected action order: got %v want %v", calls, want)
		}
	}
}
