package app

import (
	"context"
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
