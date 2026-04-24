package app

import (
	"context"
	"errors"
	"log"
	"strings"
)

var newRunCommandDispatcher = newCommandDispatcher

// Run 委托给命令分发器，保持 bridge 进程入口最小化。
func Run(ctx context.Context, args []string) (string, error) {
	dispatcher := newRunCommandDispatcher()
	if !IsServeSubcommand(args) {
		return dispatcher.dispatch(ctx, args)
	}

	// startup checkpoint 仅用于 serve 启动路径，便于定位长生命周期服务初始化阶段。
	log.Print("startup checkpoint stage=dispatch status=begin")
	output, err := dispatcher.dispatch(ctx, args)
	if err == nil {
		log.Print("startup checkpoint stage=dispatch status=ready")
		return output, nil
	}

	var usageErr usageError
	if errors.As(err, &usageErr) {
		return "", err
	}

	log.Printf("startup checkpoint stage=dispatch status=error error=%v", err)
	return "", err
}

// IsServeSubcommand 返回参数是否表示 serve 子命令。
func IsServeSubcommand(args []string) bool {
	return len(args) > 0 && strings.TrimSpace(args[0]) == "serve"
}
