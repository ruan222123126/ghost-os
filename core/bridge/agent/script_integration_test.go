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

	tool := tools.NewScriptExecTool(execution.NewNativeClient(), 0)
	args, _ := json.Marshal(map[string]any{
		"script": "files = list_files(path='.')\nprint(f'Found {len(files)} files')",
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

	script := "target = '" + targetFile + "'\n" +
		"write_file(path=target, content='alpha\\nbeta\\ngamma\\n')\n" +
		"apply_diff(path=target, diff_text='''@@ -1,3 +1,3 @@\\n alpha\\n-beta\\n+beta2\\n gamma\\n''')\n" +
		"import json\n" +
		"matches = search_files(query='beta2', path='" + filepath.ToSlash(tempDir) + "', max_results=5)\n" +
		"print(read_file(path=target, start_line=1, end_line=3))\n" +
		"print(json.dumps(matches))\n"

	tool := tools.NewScriptExecTool(execution.NewNativeClient(), 0)
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
	if !strings.Contains(output, `"line": 2`) {
		t.Fatalf("expected search_files output to include patched line, got: %s", output)
	}
}
