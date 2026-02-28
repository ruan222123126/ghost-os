package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Client 定义 bridge 调用 execution layer 的最小接口。
type Client interface {
	Call(ctx context.Context, action string, params map[string]any, traceID string) (map[string]any, error)
}

type request struct {
	Action  string         `json:"action"`
	Params  map[string]any `json:"params"`
	TraceID string         `json:"trace_id"`
}

type response struct {
	Status  string         `json:"status"`
	Payload map[string]any `json:"payload"`
	Error   string         `json:"error"`
}

// NativeClient 通过本地 native 二进制完成 execution bus 调用。
type NativeClient struct{}

func NewNativeClient() Client {
	return NativeClient{}
}

func (NativeClient) Call(ctx context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
	nativeBin, err := locateNativeBinary()
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, nativeBin)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	if traceID == "" {
		traceID = fmt.Sprintf("bridge-exec-%d", time.Now().UnixNano())
	}

	request := request{
		Action:  action,
		Params:  params,
		TraceID: traceID,
	}
	if request.Params == nil {
		request.Params = map[string]any{}
	}

	if err := json.NewEncoder(stdin).Encode(request); err != nil {
		_ = cmd.Wait()
		return nil, err
	}
	if err := stdin.Close(); err != nil {
		_ = cmd.Wait()
		return nil, err
	}

	var response response
	if err := json.NewDecoder(stdout).Decode(&response); err != nil {
		_ = cmd.Wait()
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	if response.Status != "success" {
		if response.Error == "" {
			response.Error = "execution returned error"
		}
		return nil, errors.New(response.Error)
	}
	if response.Payload == nil {
		response.Payload = map[string]any{}
	}
	return response.Payload, nil
}

func locateNativeBinary() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("GHOST_NATIVE_BIN")); configured != "" {
		path, err := normalizeBinaryPath(configured)
		if err != nil {
			return "", fmt.Errorf("invalid GHOST_NATIVE_BIN: %w", err)
		}
		if !binaryExists(path) {
			return "", fmt.Errorf("GHOST_NATIVE_BIN points to missing binary: %s", path)
		}
		return path, nil
	}

	roots := []string{"."}
	if executablePath, err := os.Executable(); err == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(executablePath); resolveErr == nil {
			executablePath = resolved
		}
		roots = append(roots, filepath.Dir(executablePath))
	}

	path, err := locateNativeBinaryInRoots(nativeBinaryCandidates(), roots)
	if err == nil {
		return path, nil
	}

	return "", errors.New("native binary not found, set GHOST_NATIVE_BIN or run `cargo build` in drivers/native first")
}

func nativeBinaryCandidates() []string {
	return []string{
		"../../drivers/native/target/debug/native",
		"drivers/native/target/debug/native",
		"../../drivers/native/target/debug/native.exe",
		"drivers/native/target/debug/native.exe",
	}
}

func locateNativeBinaryInRoots(candidates []string, roots []string) (string, error) {
	seen := make(map[string]struct{}, len(candidates)*len(roots))
	for _, root := range roots {
		base := strings.TrimSpace(root)
		if base == "" {
			base = "."
		}

		for _, candidate := range candidates {
			if strings.TrimSpace(candidate) == "" {
				continue
			}

			path := candidate
			if !filepath.IsAbs(candidate) {
				path = filepath.Join(base, candidate)
			}

			resolved, err := normalizeBinaryPath(path)
			if err != nil {
				continue
			}
			if _, exists := seen[resolved]; exists {
				continue
			}
			seen[resolved] = struct{}{}

			if binaryExists(resolved) {
				return resolved, nil
			}
		}
	}

	return "", errors.New("native binary not found in roots")
}

func normalizeBinaryPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("path is empty")
	}

	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}

	return filepath.Clean(abs), nil
}

func binaryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
