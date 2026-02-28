package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

const defaultUserMessage = "Hello, what can you do?"

type commandDispatcher struct{}

func newCommandDispatcher() commandDispatcher {
	return commandDispatcher{}
}

// dispatch 负责 CLI 子命令分发，专注命令解析与路由。
func (d commandDispatcher) dispatch(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return runAgent(ctx, defaultUserMessage)
	}

	switch args[0] {
	case "ping":
		return runPing()
	case "serve":
		port, err := d.parseServePort(args[1:])
		if err != nil {
			return "", err
		}
		return runServer(ctx, port)
	case "agent":
		return runAgent(ctx, d.parseAgentMessage(args[1:]))
	default:
		return runAgent(ctx, strings.Join(args, " "))
	}
}

func (commandDispatcher) parseServePort(args []string) (int, error) {
	if len(args) == 0 {
		return 8080, nil
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(args[0]))
	if err != nil || parsed <= 0 || parsed > 65535 {
		return 0, fmt.Errorf("invalid port %q, expected 1-65535", args[0])
	}
	return parsed, nil
}

func (commandDispatcher) parseAgentMessage(args []string) string {
	if len(args) == 0 {
		return defaultUserMessage
	}
	return strings.Join(args, " ")
}
