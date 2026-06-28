package externalagent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func codexEnv(pathOverride string) []string {
	env := os.Environ()
	hasRustLog := false
	hasPath := false
	for i, item := range env {
		if matchesEnvKey(item, "RUST_LOG") {
			hasRustLog = true
			if !strings.Contains(item, "codex_core::rollout::list=") {
				env[i] = envAssignment("RUST_LOG", envValue(item)+",codex_core::rollout::list=off")
			}
			continue
		}
		if pathOverride != "" && matchesEnvKey(item, "PATH") {
			hasPath = true
			env[i] = envAssignment(envKey(item), pathOverride)
		}
	}
	if !hasRustLog {
		env = append(env, "RUST_LOG=codex_core::rollout::list=off")
	}
	if pathOverride != "" && !hasPath {
		env = append(env, envAssignment("PATH", pathOverride))
	}
	return env
}

func matchesEnvKey(item string, key string) bool {
	return strings.EqualFold(envKey(item), key)
}

func envKey(item string) string {
	key, _, _ := strings.Cut(item, "=")
	return key
}

func envValue(item string) string {
	_, value, _ := strings.Cut(item, "=")
	return value
}

func envAssignment(key string, value string) string {
	return key + "=" + value
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
