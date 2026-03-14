package app

import (
	"context"
	"fmt"

	bridgeruntime "ghost-os/bridge/runtime"
)

// runPing 启动 native 二进制并完成一次 PING/PONG 往返。
func runPing() (string, error) {
	client := bridgeruntime.NewExecutionClientFromEnv()
	defer func() {
		_ = bridgeruntime.CloseExecutionClient(client)
	}()

	payload, err := client.Call(context.Background(), "PING", map[string]any{}, "bridge-ping-1")
	if err != nil {
		return "", err
	}

	message, _ := payload["message"].(string)
	if message == "" {
		return "", fmt.Errorf("invalid PING response payload")
	}
	return message, nil
}
