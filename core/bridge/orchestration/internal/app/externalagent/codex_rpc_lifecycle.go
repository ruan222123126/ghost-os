package externalagent

import (
	"context"
	"strings"
)

func sandboxPolicy(sandbox string) map[string]any {
	switch sandbox {
	case "read-only":
		return map[string]any{"type": "readOnly"}
	case "danger-full-access":
		return map[string]any{"type": "dangerFullAccess"}
	default:
		return map[string]any{"type": "workspaceWrite"}
	}
}

func (c *appServerClient) InterruptTurn(ctx context.Context, threadID string, turnID string) error {
	if strings.TrimSpace(threadID) == "" || strings.TrimSpace(turnID) == "" {
		return nil
	}
	_, err := c.request(ctx, "turn/interrupt", map[string]string{
		"threadId": strings.TrimSpace(threadID),
		"turnId":   strings.TrimSpace(turnID),
	})
	return err
}

func (c *appServerClient) Close() error {
	c.mu.Lock()
	cmd := c.cmd
	stdin := c.stdin
	c.cmd = nil
	c.stdin = nil
	c.connected = false
	pending := c.pending
	c.pending = make(map[int]chan rpcResponse)
	c.mu.Unlock()

	for _, ch := range pending {
		close(ch)
	}
	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return nil
}
