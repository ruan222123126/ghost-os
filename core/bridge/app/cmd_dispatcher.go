package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	bridgetransport "ghost-os/bridge/transport"
)

const (
	defaultServePort = 8080
	cliUsage         = "usage:\n  ping\n  serve [port]\n  agent <message>"
)

type commandDispatcher struct {
	runPing  func(context.Context) (string, error)
	runServe func(context.Context, int) (string, error)
	runAgent func(context.Context, string) (string, error)
}

type usageError struct {
	detail string
}

func (e usageError) Error() string {
	if e.detail == "" {
		return cliUsage
	}
	return fmt.Sprintf("%s\n%s", e.detail, cliUsage)
}

func newUsageError(detail string) error {
	return usageError{detail: detail}
}

// newCommandDispatcher 装配 CLI 入口依赖，保持 Run 本身最小化。
func newCommandDispatcher() commandDispatcher {
	return commandDispatcher{
		runPing:  runPing,
		runServe: bridgetransport.RunServer,
		runAgent: runAgent,
	}
}

// dispatch 负责 CLI 子命令分发，专注命令解析与路由。
func (d commandDispatcher) dispatch(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", newUsageError("missing subcommand")
	}

	switch args[0] {
	case "ping":
		if len(args) != 1 {
			return "", newUsageError("ping does not accept arguments")
		}
		return d.runPing(ctx)
	case "serve":
		port, err := d.parseServePort(args[1:])
		if err != nil {
			return "", err
		}
		return d.runServe(ctx, port)
	case "agent":
		message, err := d.parseAgentMessage(args[1:])
		if err != nil {
			return "", err
		}
		return d.runAgent(ctx, message)
	default:
		return "", newUsageError(fmt.Sprintf("unknown subcommand %q", args[0]))
	}
}

// parseServePort 解析 serve 子命令端口，仅允许一个可选端口参数。
func (commandDispatcher) parseServePort(args []string) (int, error) {
	if len(args) > 1 {
		return 0, newUsageError("serve accepts at most one port argument")
	}
	if len(args) == 0 {
		return defaultServePort, nil
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(args[0]))
	if err != nil || parsed <= 0 || parsed > 65535 {
		return 0, fmt.Errorf("invalid port %q, expected 1-65535", args[0])
	}
	return parsed, nil
}

// parseAgentMessage 将命令行剩余参数拼成一条非空消息。
func (commandDispatcher) parseAgentMessage(args []string) (string, error) {
	message := strings.TrimSpace(strings.Join(args, " "))
	if message == "" {
		return "", newUsageError("agent requires a non-empty message")
	}
	return message, nil
}
