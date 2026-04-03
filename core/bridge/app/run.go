package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
)

// Run 委托给命令分发器，保持 bridge 进程入口最小化。
func Run(ctx context.Context, args []string) (string, error) {
	dispatcher := newCommandDispatcher()
	if !isServeCommand(args) {
		return dispatcher.dispatch(ctx, args)
	}

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

	wrapped := fmt.Errorf("serve dispatch failed: %w", err)
	log.Printf("startup checkpoint stage=dispatch status=error error=%v", wrapped)
	return "", wrapped
}

func isServeCommand(args []string) bool {
	return len(args) > 0 && strings.TrimSpace(args[0]) == "serve"
}
