package agent

import (
	"context"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
)

// RunStream 保留给测试兼容旧调用，生产路径统一走 RunMessageStreamWithTraceID。
func (a *Agent) RunStream(ctx context.Context, userMessage string, sink streaming.Sink) (string, error) {
	return a.RunMessageStreamWithTraceID(ctx, llm.Message{Role: llm.RoleUser, Text: userMessage}, "", sink)
}

// RunStreamWithTraceID 保留给测试兼容旧调用，生产路径统一走 RunMessageStreamWithTraceID。
func (a *Agent) RunStreamWithTraceID(ctx context.Context, userMessage string, traceID string, sink streaming.Sink) (string, error) {
	return a.RunMessageStreamWithTraceID(ctx, llm.Message{Role: llm.RoleUser, Text: userMessage}, traceID, sink)
}
