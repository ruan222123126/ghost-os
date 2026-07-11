package llm

import (
	"context"
	"errors"
	"net"
	"net/http"
)

// IsTransientCompletionError 判断 completion 失败是否属于可重试瞬时错误。
func IsTransientCompletionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var statusErr *completionStatusError
	if errors.As(err, &statusErr) {
		return statusErr.statusCode == http.StatusTooManyRequests || statusErr.statusCode >= http.StatusInternalServerError
	}

	var netErr net.Error
	return errors.As(err, &netErr)
}
