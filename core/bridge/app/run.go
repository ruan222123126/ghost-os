package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

const defaultUserMessage = "Hello, what can you do?"

// Run 负责 CLI 子命令分发，保持最小入口逻辑。
func Run(ctx context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return runAgent(ctx, defaultUserMessage)
	}

	switch args[0] {
	case "ping":
		return runPing()
	case "serve":
		port := 8080
		if len(args) > 1 {
			parsed, err := strconv.Atoi(strings.TrimSpace(args[1]))
			if err != nil || parsed <= 0 || parsed > 65535 {
				return "", fmt.Errorf("invalid port %q, expected 1-65535", args[1])
			}
			port = parsed
		}
		return runServer(ctx, port)
	case "agent":
		message := defaultUserMessage
		if len(args) > 1 {
			message = strings.Join(args[1:], " ")
		}
		return runAgent(ctx, message)
	default:
		return runAgent(ctx, strings.Join(args, " "))
	}
}
