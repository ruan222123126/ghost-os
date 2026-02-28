package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ghost-os/bridge/execution"
	"ghost-os/bridge/tools"
)

// TestScriptExecutionEndToEnd 依赖本地 native binary，缺失时自动跳过。
func TestScriptExecutionEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test in short mode")
	}

	tool := tools.NewScriptExecTool(execution.NewNativeClient())
	args, _ := json.Marshal(map[string]any{
		"script": "files = tools.list_files(path='.')\nprint(f'Found {len(files)} files')",
	})

	output, err := tool.Execute(context.Background(), args, "integration-trace")
	if err != nil {
		if strings.Contains(err.Error(), "native binary not found") {
			t.Skip("native binary not found; run cargo build in drivers/native first")
		}
		t.Fatalf("script execution failed: %v", err)
	}

	if strings.TrimSpace(output) == "" {
		t.Fatal("expected non-empty output")
	}
}

func TestScriptExecutionFileToolWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test in short mode")
	}

	tempDir, err := os.MkdirTemp(".", "script-tools-*")
	if err != nil {
		t.Fatalf("create temp dir failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	targetFile := filepath.ToSlash(filepath.Join(tempDir, "sample.txt"))
	searchDir := filepath.ToSlash(tempDir)

	script := "target = '" + targetFile + "'\n" +
		"tools.write_file(path=target, content='alpha\\nbeta\\ngamma\\n')\n" +
		"tools.apply_diff(path=target, diff_text='''@@ -1,3 +1,3 @@\\n alpha\\n-beta\\n+beta2\\n gamma\\n''')\n" +
		"matches = tools.search_files(keyword='beta2', dir_path='" + searchDir + "', case_sensitive=True)\n" +
		"print(tools.read_file(path=target, start_line=1, end_line=3))\n" +
		"print(f'matches={len(matches)}')\n"

	tool := tools.NewScriptExecTool(execution.NewNativeClient())
	args, _ := json.Marshal(map[string]any{"script": script})

	output, err := tool.Execute(context.Background(), args, "integration-trace-tools")
	if err != nil {
		if strings.Contains(err.Error(), "native binary not found") {
			t.Skip("native binary not found; run cargo build in drivers/native first")
		}
		t.Fatalf("script execution failed: %v", err)
	}

	if !strings.Contains(output, "beta2") {
		t.Fatalf("expected patched content in output, got: %s", output)
	}
	if !strings.Contains(output, "matches=1") {
		t.Fatalf("expected search match count in output, got: %s", output)
	}
}
