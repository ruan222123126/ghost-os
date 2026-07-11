package externalagent

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

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
