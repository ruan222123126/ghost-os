package transport

import "context"

func RunServer(ctx context.Context, port int) (string, error) {
	return runServer(ctx, port)
}
