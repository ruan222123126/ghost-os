package app

import (
	"context"
)

// Run 委托给命令分发器，保持 bridge 进程入口最小化。
func Run(ctx context.Context, args []string) (string, error) {
	return newCommandDispatcher().dispatch(ctx, args)
}
