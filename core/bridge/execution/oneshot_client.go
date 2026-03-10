package execution

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
)

// NativeClient 通过本地 native 二进制完成 one-shot execution bus 调用。
type NativeClient struct{}

func NewNativeClient() Client {
	return NativeClient{}
}

func (NativeClient) Call(ctx context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
	nativeBin, err := locateNativeBinary()
	if err != nil {
		return nil, err
	}

	return callNativeOnceWithCommand(exec.CommandContext(ctx, nativeBin), newRequest(action, params, traceID))
}

func callNativeOnceWithCommand(cmd *exec.Cmd, req request) (map[string]any, error) {
	cmd.Stderr = os.Stderr

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
	if err := json.NewDecoder(stdout).Decode(&resp); err != nil {
		_ = cmd.Wait()
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return resp.intoResult()
}
