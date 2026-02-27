package tools

import "context"

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, argsJSON string) (string, error)
}
