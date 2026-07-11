package storage

import "testing"

func TestResolveCodexCLIPathLoadsFileValue(t *testing.T) {
	path := " /opt/tools/codex "
	cfg := FileConfig{CodexCLIPath: &path}

	if got := ResolveCodexCLIPath(cfg, EnvSnapshot{}); got != "/opt/tools/codex" {
		t.Fatalf("unexpected codex_cli_path: got %q want %q", got, "/opt/tools/codex")
	}
}

func TestResolveCodexCLIPathEnvOverridesFile(t *testing.T) {
	path := "/file/codex"
	cfg := FileConfig{CodexCLIPath: &path}
	env := EnvSnapshot{"GHOST_CODEX_CLI_PATH": " /env/codex "}

	if got := ResolveCodexCLIPath(cfg, env); got != "/env/codex" {
		t.Fatalf("unexpected codex_cli_path: got %q want %q", got, "/env/codex")
	}
}

func TestResolveNodeBinPathLoadsFileValue(t *testing.T) {
	path := " /opt/tools/node "
	cfg := FileConfig{NodeBinPath: &path}

	if got := ResolveNodeBinPath(cfg, EnvSnapshot{}); got != "/opt/tools/node" {
		t.Fatalf("unexpected node_bin_path: got %q want %q", got, "/opt/tools/node")
	}
}

func TestResolveNodeBinPathEnvOverridesFile(t *testing.T) {
	path := "/file/node"
	cfg := FileConfig{NodeBinPath: &path}
	env := EnvSnapshot{"GHOST_NODE_BIN_PATH": " /env/node "}

	if got := ResolveNodeBinPath(cfg, env); got != "/env/node" {
		t.Fatalf("unexpected node_bin_path: got %q want %q", got, "/env/node")
	}
}
