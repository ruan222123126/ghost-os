package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"ghost-os/bridge/app"
)

// main 作为 bridge 进程入口：执行 app.Run 并输出结果。
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	output, err := app.Run(ctx, os.Args[1:])
	if err != nil {
		fatal(err)
	}
	if output != "" {
		fmt.Println(output)
	}
}

// fatal 统一向 stderr 输出错误并返回非零退出码。
func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
