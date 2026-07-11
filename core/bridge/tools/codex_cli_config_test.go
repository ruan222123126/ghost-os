package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestCodexCLIToolStartPassesConfiguredExecutablePaths(t *testing.T) {
	client := &mockCodexCLIExecutionClient{
		callFunc: func(context.Context, string, map[string]any, string) (map[string]any, error) {
			return map[string]any{
				"output_path":    "/tmp/codex-cli.log",
				"exit_code_path": "/tmp/codex-cli.log.exit",
			}, nil
		},
	}
	tool, ok := NewCodexCLITool(client, " /opt/codex/bin/codex ", " /opt/node/bin/node ").(*CodexCLITool)
	if !ok {
		t.Fatal("expected *CodexCLITool")
	}

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"op":"start","prompt":"hello","wait_ms_before_async":1}`),
		"trace-start",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if len(client.calls) != 1 {
		t.Fatalf("expected one execution call, got %d", len(client.calls))
	}
	assertCodexCLIParamValue(t, client.calls[0].params, "codex_executable_path", "/opt/codex/bin/codex")
	assertCodexCLIParamValue(t, client.calls[0].params, "node_executable_path", "/opt/node/bin/node")
}

func TestCodexCLIToolStartDoesNotExposeExecutablePathsInSchema(t *testing.T) {
	raw := NewCodexCLITool(nil, "", "").Parameters()
	if string(raw) == "" {
		t.Fatal("expected parameters schema")
	}
	if strings.Contains(string(raw), "codex_executable_path") {
		t.Fatalf("unexpected codex_executable_path in tool schema: %s", string(raw))
	}
	if strings.Contains(string(raw), "node_executable_path") {
		t.Fatalf("unexpected node_executable_path in tool schema: %s", string(raw))
	}
}
