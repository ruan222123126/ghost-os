package app

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"ghost-os/bridge/execution"
	bridgeruntime "ghost-os/bridge/runtime"
)

type pingRunner struct {
	newClient   func() (execution.Client, error)
	closeClient func(execution.Client) error
	nextTraceID func() string
}

var pingTraceCounter uint64

// runPing 启动 native 二进制并完成一次 PING/PONG 往返。
func runPing(ctx context.Context) (string, error) {
	return newPingRunner().run(ctx)
}

func newPingRunner() pingRunner {
	return pingRunner{
		newClient:   bridgeruntime.NewExecutionClientFromEnv,
		closeClient: bridgeruntime.CloseExecutionClient,
		nextTraceID: nextPingTraceID,
	}
}

func (r pingRunner) run(ctx context.Context) (string, error) {
	client, err := r.newClient()
	if err != nil {
		return "", err
	}
	defer func() {
		_ = r.closeClient(client)
	}()

	payload, err := client.Call(ctx, "PING", map[string]any{}, r.nextTraceID())
	if err != nil {
		return "", err
	}

	message, _ := payload["message"].(string)
	if message == "" {
		return "", fmt.Errorf("invalid PING response payload")
	}
	return message, nil
}

func nextPingTraceID() string {
	sequence := atomic.AddUint64(&pingTraceCounter, 1)
	return fmt.Sprintf("bridge-ping-%d-%d", time.Now().UnixMilli(), sequence)
}
