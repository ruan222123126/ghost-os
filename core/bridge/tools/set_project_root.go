package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const setProjectRootToolName = "set_project_root"

var defaultProjectRootMediaDirs = []string{"Files", "Apps", "App"}

// ProjectRootStore 描述持久化 project_root 的最小接口。
type ProjectRootStore interface {
	SetProjectRoot(path string) error
}

type SetProjectRootTool struct {
	store             ProjectRootStore
	execution         ExecutionClient
	allowedReadPaths  []string
	allowedWritePaths []string
}

type setProjectRootArgs struct {
	Path string `json:"path"`
}

type setProjectRootResult struct {
	NewRoot              string `json:"new_root"`
	Persisted            bool   `json:"persisted"`
	EffectiveImmediately bool   `json:"effective_immediately"`
}

// NewSetProjectRootTool 创建 set_project_root 工具。
func NewSetProjectRootTool(store ProjectRootStore, execution ExecutionClient, readPaths, writePaths []string) Tool {
	return &SetProjectRootTool{
		store:             store,
		execution:         execution,
		allowedReadPaths:  append([]string(nil), readPaths...),
		allowedWritePaths: append([]string(nil), writePaths...),
	}
}

func (SetProjectRootTool) Name() string {
	return setProjectRootToolName
}

func (SetProjectRootTool) Description() string {
	return "Set and persist the project root directory used by execution tools. The path must exist, be a directory, and be inside the allowed read/write roots."
}

func (SetProjectRootTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"path":{"type":"string","description":"New project root directory path."}
		},
		"required":["path"],
		"additionalProperties":false
	}`)
}

func (t *SetProjectRootTool) Execute(_ context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	if t == nil || t.store == nil {
		return "", fmt.Errorf("config store is not configured")
	}
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}
	args, err := decodeSetProjectRootArgs(argsJSON)
	if err != nil {
		return "", err
	}
	root, err := resolveProjectRootPath(args.Path)
	if err != nil {
		return "", err
	}
	allowed, err := t.allowedRoots()
	if err != nil {
		return "", err
	}
	if !pathWithinAnyRoot(root, allowed) {
		return "", fmt.Errorf("project root is outside allowed paths")
	}
	if err := t.store.SetProjectRoot(root); err != nil {
		return "", err
	}
	if err := t.applyWorkingDir(root); err != nil {
		return "", err
	}
	return marshalSetProjectRootResult(root)
}

func decodeSetProjectRootArgs(argsJSON json.RawMessage) (setProjectRootArgs, error) {
	var args setProjectRootArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return setProjectRootArgs{}, fmt.Errorf("decode args: %w", err)
	}
	if strings.TrimSpace(args.Path) == "" {
		return setProjectRootArgs{}, errors.New("path is required")
	}
	return args, nil
}

func resolveProjectRootPath(raw string) (string, error) {
	absPath, err := resolveAbsolutePath(raw)
	if err != nil {
		return "", err
	}
	if err := ensureDirExists(absPath); err != nil {
		return "", err
	}
	return absPath, nil
}

func resolveAbsolutePath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("path is empty")
	}
	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home directory: %w", err)
		}
		if trimmed == "~" {
			trimmed = homeDir
		} else {
			trimmed = filepath.Join(homeDir, strings.TrimPrefix(trimmed, "~/"))
		}
	}
	cleaned := filepath.Clean(trimmed)
	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}
	return absPath, nil
}

func ensureDirExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path does not exist: %w", err)
	}
	if !info.IsDir() {
		return errors.New("path is not a directory")
	}
	return nil
}

func (t *SetProjectRootTool) allowedRoots() ([]string, error) {
	if len(t.allowedReadPaths) == 0 && len(t.allowedWritePaths) == 0 {
		return defaultAllowedRoots()
	}
	combined := append([]string(nil), t.allowedReadPaths...)
	combined = append(combined, t.allowedWritePaths...)
	return normalizeAllowedRoots(combined)
}

func defaultAllowedRoots() ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve cwd: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir: %w", err)
	}
	roots := []string{cwd, home}
	user := filepath.Base(home)
	mediaRoot := filepath.Join(string(filepath.Separator), "media", user)
	for _, name := range defaultProjectRootMediaDirs {
		candidate := filepath.Join(mediaRoot, name)
		if isDir(candidate) {
			roots = append(roots, candidate)
		}
	}
	return uniquePaths(roots), nil
}

func normalizeAllowedRoots(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, errors.New("allowed paths are empty")
	}
	roots := make([]string, 0, len(raw))
	for _, value := range raw {
		resolved, err := resolveAbsolutePath(value)
		if err != nil {
			return nil, fmt.Errorf("resolve allowed path %q: %w", value, err)
		}
		if err := ensureDirExists(resolved); err != nil {
			return nil, fmt.Errorf("allowed path %q: %w", resolved, err)
		}
		roots = append(roots, resolved)
	}
	return uniquePaths(roots), nil
}

func pathWithinAnyRoot(target string, roots []string) bool {
	for _, root := range roots {
		if isWithinRoot(target, root) {
			return true
		}
	}
	return false
}

func isWithinRoot(target, root string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func uniquePaths(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (t *SetProjectRootTool) applyWorkingDir(dir string) error {
	setter, ok := t.execution.(WorkingDirSetter)
	if !ok {
		return errors.New("execution client does not support working directory updates")
	}
	return setter.SetWorkingDir(dir)
}

func marshalSetProjectRootResult(root string) (string, error) {
	result := setProjectRootResult{
		NewRoot:              root,
		Persisted:            true,
		EffectiveImmediately: true,
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("encode result: %w", err)
	}
	return string(raw), nil
}
