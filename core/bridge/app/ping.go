package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// pingRequest 遵循 core/shared/schema.json 的最小请求结构。
type pingRequest struct {
	Action  string         `json:"action"`
	Params  map[string]any `json:"params"`
	TraceID string         `json:"trace_id"`
}

// pingResponse 对应 native 进程返回的统一响应结构。
type pingResponse struct {
	Status  string         `json:"status"`
	Payload map[string]any `json:"payload"`
	Error   string         `json:"error"`
}

// runPing 启动 native 二进制并完成一次 PING/PONG 往返。
func runPing() (string, error) {
	nativeBin, err := locateNativeBinary()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(nativeBin)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	request := pingRequest{
		Action:  "PING",
		Params:  map[string]any{},
		TraceID: "bridge-ping-1",
	}

	if err := json.NewEncoder(stdin).Encode(request); err != nil {
		_ = cmd.Wait()
		return "", err
	}
	if err := stdin.Close(); err != nil {
		_ = cmd.Wait()
		return "", err
	}

	var response pingResponse
	if err := json.NewDecoder(stdout).Decode(&response); err != nil {
		_ = cmd.Wait()
		return "", err
	}
	if err := cmd.Wait(); err != nil {
		return "", err
	}

	if response.Status != "success" {
		return "", fmt.Errorf("native error: %s", response.Error)
	}

	message, _ := response.Payload["message"].(string)
	return message, nil
}

// locateNativeBinary 按候选路径查找 native 可执行文件。
func locateNativeBinary() (string, error) {
	candidates := []string{
		"../../drivers/native/target/debug/native",
		"drivers/native/target/debug/native",
		"../../drivers/native/target/debug/native.exe",
		"drivers/native/target/debug/native.exe",
	}

	for _, candidate := range candidates {
		path := filepath.Clean(candidate)
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return path, nil
		}
	}

	return "", errors.New("native binary not found, run `cargo build` in drivers/native first")
}
