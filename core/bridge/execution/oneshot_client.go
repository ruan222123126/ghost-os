package execution

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const nativeStrictDecoderEnv = "GHOST_NATIVE_STRICT_DECODER"

// NativeClient 通过本地 native 二进制完成 one-shot execution bus 调用。
type NativeClient struct {
	locator           nativeBinaryLocator
	allowedReadPaths  []string
	allowedWritePaths []string
	workingDir        string
}

func NewNativeClient() Client {
	return newNativeClientWithLocator(nativeBinaryLocator{})
}

func (c *NativeClient) SupportsPersistentSessions() bool {
	return false
}

func newNativeClientWithLocator(locator nativeBinaryLocator) *NativeClient {
	return &NativeClient{locator: locator}
}

func (c *NativeClient) Call(ctx context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
	nativeBin, err := locateNativeBinary(c.locator)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, nativeBin)
	c.applyWorkingDir(cmd)
	return callNativeOnceWithCommand(
		cmd,
		newRequest(action, params, traceID),
		c.allowedReadPaths,
		c.allowedWritePaths,
	)
}

func (c *NativeClient) SetWorkingDir(dir string) error {
	if c == nil {
		return nil
	}
	c.workingDir = strings.TrimSpace(dir)
	return nil
}

func (c *NativeClient) applyWorkingDir(cmd *exec.Cmd) {
	if c == nil || cmd == nil {
		return
	}
	if dir := strings.TrimSpace(c.workingDir); dir != "" {
		cmd.Dir = dir
	}
}

func callNativeOnceWithCommand(
	cmd *exec.Cmd,
	req request,
	allowedReadPaths []string,
	allowedWritePaths []string,
) (map[string]any, error) {
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), cmd.Env...)
	cmd.Env = append(cmd.Env, buildNativeAllowedPathEnv(allowedReadPaths, allowedWritePaths)...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, err
	}

	if err := json.NewEncoder(stdin).Encode(req); err != nil {
		_ = cmd.Wait()
		return nil, err
	}
	if err := stdin.Close(); err != nil {
		_ = cmd.Wait()
		return nil, err
	}

	var resp response
	decoder := json.NewDecoder(stdout)
	if strictDecoderEnabled() {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(&resp); err != nil {
		_ = cmd.Wait()
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return resp.intoResult(req)
}

func strictDecoderEnabled() bool {
	raw := strings.TrimSpace(os.Getenv(nativeStrictDecoderEnv))
	if raw == "" {
		return false
	}
	enabled, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}
	return enabled
}
