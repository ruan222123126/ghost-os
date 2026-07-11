package externalagent

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveCodexExecutablePrefersExplicitRunnablePath(t *testing.T) {
	dir := t.TempDir()
	codexPath := createExecutable(t, dir, codexCandidateNames()[0], "#!/bin/sh\n")

	resolved, err := resolveCodexExecutable(codexPath, "")
	if err != nil {
		t.Fatalf("resolveCodexExecutable: %v", err)
	}
	if resolved != codexPath {
		t.Fatalf("unexpected codex path: got %q want %q", resolved, codexPath)
	}
}

func TestResolveCodexExecutableReportsInvalidExplicitPath(t *testing.T) {
	dir := t.TempDir()
	missingPath := filepath.Join(dir, codexCandidateNames()[0])

	_, err := resolveCodexExecutable(missingPath, "")
	if err == nil {
		t.Fatal("expected invalid explicit path to fail")
	}
	if !strings.Contains(err.Error(), "codex_cli_path or GHOST_CODEX_CLI_PATH") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveCodexExecutableSearchesNodeSiblingDirectory(t *testing.T) {
	dir := t.TempDir()
	codexPath := createExecutable(t, dir, codexCandidateNames()[0], "#!/bin/sh\n")
	nodePath := createExecutable(t, dir, nodeCandidateNames()[0], "")
	t.Setenv("PATH", "")

	resolved, err := resolveCodexExecutable("", nodePath)
	if err != nil {
		t.Fatalf("resolveCodexExecutable: %v", err)
	}
	if resolved != codexPath {
		t.Fatalf("unexpected codex path: got %q want %q", resolved, codexPath)
	}
}

func TestResolveCodexLaunchSpecInjectsNodeDirectoryForEnvNodeLauncher(t *testing.T) {
	dir := t.TempDir()
	codexPath := createExecutable(t, dir, codexCandidateNames()[0], "#!/usr/bin/env node\nconsole.log('hi')\n")
	nodePath := createExecutable(t, dir, nodeCandidateNames()[0], "")

	spec, err := resolveCodexLaunchSpec(codexPath, nodePath)
	if err != nil {
		t.Fatalf("resolveCodexLaunchSpec: %v", err)
	}
	if spec.executablePath != codexPath {
		t.Fatalf("unexpected executable path: got %q want %q", spec.executablePath, codexPath)
	}
	entries := filepath.SplitList(spec.pathOverride)
	if len(entries) == 0 {
		t.Fatal("expected path override")
	}
	if entries[0] != filepath.Dir(nodePath) {
		t.Fatalf("unexpected first PATH entry: got %q want %q", entries[0], filepath.Dir(nodePath))
	}
}

func TestResolveCodexLaunchSpecReportsMissingNodeForEnvLauncher(t *testing.T) {
	dir := t.TempDir()
	codexPath := createExecutable(t, dir, codexCandidateNames()[0], "#!/usr/bin/env node\nconsole.log('hi')\n")
	missingNode := filepath.Join(dir, "missing-node")

	_, err := resolveCodexLaunchSpec(codexPath, missingNode)
	if err == nil {
		t.Fatal("expected missing node to fail")
	}
	if !strings.Contains(err.Error(), "node_bin_path or GHOST_NODE_BIN_PATH") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrependPathDirPutsNodeFirstWithoutDuplication(t *testing.T) {
	root := t.TempDir()
	nodeDir := filepath.Join(root, "node")
	usrDir := filepath.Join(root, "usr")
	optDir := filepath.Join(root, "opt")
	for _, dir := range []string{nodeDir, usrDir, optDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	current := strings.Join([]string{nodeDir, usrDir, optDir}, string(os.PathListSeparator))

	joined, err := prependPathDir(nodeDir, current)
	if err != nil {
		t.Fatalf("prependPathDir: %v", err)
	}
	entries := filepath.SplitList(joined)
	expected := []string{nodeDir, usrDir, optDir}
	if len(entries) != len(expected) {
		t.Fatalf("unexpected PATH entry count: got %d want %d", len(entries), len(expected))
	}
	for i, want := range expected {
		if entries[i] != want {
			t.Fatalf("unexpected PATH entry[%d]: got %q want %q", i, entries[i], want)
		}
	}
}

func createExecutable(t *testing.T, dir string, name string, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0o755); err != nil {
			t.Fatalf("chmod %s: %v", path, err)
		}
	}
	return path
}
