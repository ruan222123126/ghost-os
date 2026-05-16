package config

import "testing"

func TestResolveConfigLoadsCodexCLIPathFromFile(t *testing.T) {
	cfg, err := resolveConfig(
		bridgeFileConfig{
			CodexCLIPath: stringPointer(" /opt/tools/codex "),
		},
		envSnapshot{"GHOST_PROVIDER": "custom"},
	)
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.CodexCLIPath != "/opt/tools/codex" {
		t.Fatalf("unexpected codex_cli_path: got %q want %q", cfg.CodexCLIPath, "/opt/tools/codex")
	}
}

func TestResolveConfigEnvCodexCLIPathOverridesFile(t *testing.T) {
	cfg, err := resolveConfig(
		bridgeFileConfig{
			CodexCLIPath: stringPointer("/file/codex"),
		},
		envSnapshot{
			"GHOST_PROVIDER":       "custom",
			"GHOST_CODEX_CLI_PATH": " /env/codex ",
		},
	)
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.CodexCLIPath != "/env/codex" {
		t.Fatalf("unexpected codex_cli_path: got %q want %q", cfg.CodexCLIPath, "/env/codex")
	}
}

func TestResolveConfigLoadsNodeBinPathFromFile(t *testing.T) {
	cfg, err := resolveConfig(
		bridgeFileConfig{
			NodeBinPath: stringPointer(" /opt/tools/node "),
		},
		envSnapshot{"GHOST_PROVIDER": "custom"},
	)
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.NodeBinPath != "/opt/tools/node" {
		t.Fatalf("unexpected node_bin_path: got %q want %q", cfg.NodeBinPath, "/opt/tools/node")
	}
}

func TestResolveConfigEnvNodeBinPathOverridesFile(t *testing.T) {
	cfg, err := resolveConfig(
		bridgeFileConfig{
			NodeBinPath: stringPointer("/file/node"),
		},
		envSnapshot{
			"GHOST_PROVIDER":      "custom",
			"GHOST_NODE_BIN_PATH": " /env/node ",
		},
	)
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.NodeBinPath != "/env/node" {
		t.Fatalf("unexpected node_bin_path: got %q want %q", cfg.NodeBinPath, "/env/node")
	}
}
