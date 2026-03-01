package tools

import (
	"context"
	"strings"

	"ghost-os/bridge/session"
)

type sessionContextKey struct{}
type toolCallIDContextKey struct{}

// WithSession 把当前会话注入 tool 执行上下文。
func WithSession(ctx context.Context, sess *session.Session) context.Context {
	if sess == nil {
		return ctx
	}
	return context.WithValue(ctx, sessionContextKey{}, sess)
}

// SessionFromContext 读取 tool 执行时绑定的会话。
func SessionFromContext(ctx context.Context) *session.Session {
	if ctx == nil {
		return nil
	}
	sess, _ := ctx.Value(sessionContextKey{}).(*session.Session)
	return sess
}

// WithToolCallID 注入当前工具调用 ID，便于工具写回可追踪状态。
func WithToolCallID(ctx context.Context, toolCallID string) context.Context {
	trimmed := strings.TrimSpace(toolCallID)
	if trimmed == "" {
		return ctx
	}
	return context.WithValue(ctx, toolCallIDContextKey{}, trimmed)
}

// ToolCallIDFromContext 返回当前工具调用 ID。
func ToolCallIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(toolCallIDContextKey{}).(string)
	return strings.TrimSpace(value)
}
