package externalagent

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	codexNotFoundMessage = "codex executable not found; searched PATH, NODE_BIN_PATH sibling, and common Node/npm install directories; set codex_cli_path or GHOST_CODEX_CLI_PATH to a valid codex executable"
	nodeNotFoundMessage  = "node executable not found; searched GHOST_NODE_BIN_PATH override, NODE_BIN_PATH, PATH, and common Node install directories; set node_bin_path or GHOST_NODE_BIN_PATH to a valid node executable"
)

type codexLaunchSpec struct {
	executablePath string
	pathOverride   string
}

func resolveCodexLaunchSpec(codexExecutablePath string, nodeExecutablePath string) (codexLaunchSpec, error) {
	executablePath, err := resolveCodexExecutable(codexExecutablePath, nodeExecutablePath)
	if err != nil {
		return codexLaunchSpec{}, err
	}
	requiresNode, err := launcherRequiresNode(executablePath)
	if err != nil {
		return codexLaunchSpec{}, err
	}
	if !requiresNode {
		return codexLaunchSpec{executablePath: executablePath}, nil
	}
	nodePath, err := resolveNodeExecutable(nodeExecutablePath)
	if err != nil {
		return codexLaunchSpec{}, err
	}
	pathOverride, err := prependPathDir(filepath.Dir(nodePath), os.Getenv("PATH"))
	if err != nil {
		return codexLaunchSpec{}, err
	}
	return codexLaunchSpec{
		executablePath: executablePath,
		pathOverride:   pathOverride,
	}, nil
}

func resolveCodexExecutable(explicitPath string, nodeExecutablePath string) (string, error) {
	if path := normalizeOptionalPath(explicitPath); path != "" {
		return resolveExplicitPath(path, "codex", "codex_cli_path or GHOST_CODEX_CLI_PATH")
	}
	if path := lookPath("codex"); path != "" {
		return path, nil
	}
	if path := findNamedExecutableInDirs(runtimeCandidateDirs(nodeExecutablePath), codexCandidateNames()); path != "" {
		return path, nil
	}
	return "", errors.New(codexNotFoundMessage)
}

func resolveNodeExecutable(explicitPath string) (string, error) {
	if path := normalizeOptionalPath(explicitPath); path != "" {
		return resolveExplicitPath(path, "node", "node_bin_path or GHOST_NODE_BIN_PATH")
	}
	if path := nodeBinPathFromValue(os.Getenv("NODE_BIN_PATH")); path != "" {
		return path, nil
	}
	if path := lookPath("node"); path != "" {
		return path, nil
	}
	if path := findNamedExecutableInDirs(commonNodeDirs(), nodeCandidateNames()); path != "" {
		return path, nil
	}
	return "", errors.New(nodeNotFoundMessage)
}

func normalizeOptionalPath(input string) string {
	return strings.TrimSpace(input)
}

func resolveExplicitPath(path string, kind string, hint string) (string, error) {
	if isRunnableFile(path) {
		return path, nil
	}
	return "", fmt.Errorf(
		"configured %s executable path is not runnable: %s; set %s to a valid %s executable",
		kind,
		path,
		hint,
		kind,
	)
}

func lookPath(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

func runtimeCandidateDirs(nodeExecutablePath string) []string {
	dirs := []string{}
	if dir := codexDirFromNodeBinPath(nodeExecutablePath); dir != "" {
		dirs = append(dirs, dir)
	}
	if dir := codexDirFromNodeBinPath(os.Getenv("NODE_BIN_PATH")); dir != "" {
		dirs = append(dirs, dir)
	}
	dirs = append(dirs, commonCodexDirs()...)
	return dedupeDirs(dirs)
}

func nodeBinPathFromValue(value string) string {
	path := strings.TrimSpace(value)
	if isRunnableFile(path) {
		return path
	}
	return ""
}

func codexDirFromNodeBinPath(nodeBinPath string) string {
	path := strings.TrimSpace(nodeBinPath)
	if path == "" {
		return ""
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return ""
	}
	return dir
}

func commonCodexDirs() []string {
	dirs := []string{}
	if runtime.GOOS == "windows" {
		dirs = append(dirs, windowsCodexDirs()...)
		return dedupeDirs(dirs)
	}
	dirs = append(dirs, "/usr/local/bin", "/usr/bin")
	if runtime.GOOS == "darwin" {
		dirs = append(dirs, "/opt/homebrew/bin")
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		dirs = append(dirs, nvmVersionBinDirs(home)...)
		dirs = append(dirs,
			filepath.Join(home, ".volta", "bin"),
			filepath.Join(home, ".asdf", "shims"),
			filepath.Join(home, ".local", "share", "mise", "shims"),
		)
	}
	if path := strings.TrimSpace(os.Getenv("NVM_BIN")); path != "" {
		dirs = append(dirs, path)
	}
	if path := strings.TrimSpace(os.Getenv("VOLTA_HOME")); path != "" {
		dirs = append(dirs, filepath.Join(path, "bin"))
	}
	return dedupeDirs(dirs)
}

func commonNodeDirs() []string {
	dirs := []string{}
	if runtime.GOOS == "windows" {
		dirs = append(dirs, windowsNodeDirs()...)
		return dedupeDirs(dirs)
	}
	dirs = append(dirs, commonCodexDirs()...)
	return dedupeDirs(dirs)
}

func windowsCodexDirs() []string {
	dirs := []string{}
	if path := strings.TrimSpace(os.Getenv("APPDATA")); path != "" {
		dirs = append(dirs, filepath.Join(path, "npm"))
	}
	if path := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); path != "" {
		dirs = append(dirs, filepath.Join(path, "Volta", "bin"))
	}
	if path := strings.TrimSpace(os.Getenv("NVM_SYMLINK")); path != "" {
		dirs = append(dirs, path)
	}
	if path := strings.TrimSpace(os.Getenv("ProgramFiles")); path != "" {
		dirs = append(dirs, filepath.Join(path, "nodejs"))
	}
	if path := strings.TrimSpace(os.Getenv("ProgramFiles(x86)")); path != "" {
		dirs = append(dirs, filepath.Join(path, "nodejs"))
	}
	return dedupeDirs(dirs)
}

func windowsNodeDirs() []string {
	dirs := []string{}
	if path := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); path != "" {
		dirs = append(dirs, filepath.Join(path, "Volta", "bin"))
	}
	if path := strings.TrimSpace(os.Getenv("NVM_SYMLINK")); path != "" {
		dirs = append(dirs, path)
	}
	if path := strings.TrimSpace(os.Getenv("ProgramFiles")); path != "" {
		dirs = append(dirs, filepath.Join(path, "nodejs"))
	}
	if path := strings.TrimSpace(os.Getenv("ProgramFiles(x86)")); path != "" {
		dirs = append(dirs, filepath.Join(path, "nodejs"))
	}
	return dedupeDirs(dirs)
}

func nvmVersionBinDirs(home string) []string {
	pattern := filepath.Join(home, ".nvm", "versions", "node", "*", "bin")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	return matches
}

func dedupeDirs(dirs []string) []string {
	if len(dirs) == 0 {
		return nil
	}
	out := make([]string, 0, len(dirs))
	seen := make(map[string]struct{}, len(dirs))
	for _, dir := range dirs {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			continue
		}
		key := dirKey(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func dirKey(dir string) string {
	cleaned := filepath.Clean(strings.TrimSpace(dir))
	if runtime.GOOS == "windows" {
		return strings.ToLower(cleaned)
	}
	return cleaned
}

func findNamedExecutableInDirs(dirs []string, names []string) string {
	for _, dir := range dirs {
		if path := findNamedExecutableInDir(dir, names); path != "" {
			return path
		}
	}
	return ""
}

func findNamedExecutableInDir(dir string, names []string) string {
	for _, name := range names {
		candidate := filepath.Join(dir, name)
		if isRunnableFile(candidate) {
			return candidate
		}
	}
	return ""
}

func codexCandidateNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"codex.cmd", "codex.exe", "codex.bat", "codex"}
	}
	return []string{"codex"}
}

func nodeCandidateNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"node.exe", "node"}
	}
	return []string{"node"}
}

func isRunnableFile(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return false
	}
	info, err := os.Stat(trimmed)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode().Perm()&0o111 != 0
}

func launcherRequiresNode(path string) (bool, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".cmd" || ext == ".bat" {
		return true, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("read codex launcher failed: %w", err)
	}
	defer file.Close()
	line, err := bufio.NewReader(file).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read codex launcher failed: %w", err)
	}
	shebang, ok := strings.CutPrefix(line, "#!")
	if !ok {
		return false, nil
	}
	normalized := strings.TrimSpace(shebang)
	return strings.HasPrefix(normalized, "/usr/bin/env node") ||
		strings.HasPrefix(normalized, "env node") ||
		strings.HasPrefix(normalized, "/usr/bin/env -S node") ||
		strings.HasPrefix(normalized, "env -S node"), nil
}

func prependPathDir(dir string, currentPath string) (string, error) {
	trimmedDir := strings.TrimSpace(dir)
	if trimmedDir == "" {
		return "", fmt.Errorf("path dir is empty")
	}
	entries := []string{trimmedDir}
	for _, entry := range filepath.SplitList(currentPath) {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" || dirKey(trimmed) == dirKey(trimmedDir) {
			continue
		}
		entries = append(entries, trimmed)
	}
	return strings.Join(entries, string(os.PathListSeparator)), nil
}
