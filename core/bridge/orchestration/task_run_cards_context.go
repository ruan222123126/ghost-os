package orchestration

import "context"

type taskRunCardContextKey struct{}

func withTaskRunCardRecorder(
	ctx context.Context,
	recorder *taskRunCardRecorder,
) context.Context {
	if recorder == nil {
		return ctx
	}
	return context.WithValue(ctx, taskRunCardContextKey{}, recorder)
}

func taskRunCardRecorderFromContext(
	ctx context.Context,
) *taskRunCardRecorder {
	recorder, _ := ctx.Value(taskRunCardContextKey{}).(*taskRunCardRecorder)
	return recorder
}
