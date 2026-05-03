package agent

import "time"

const (
	defaultCompletionRetryCount    = 1
	defaultCompletionRetryInterval = 200 * time.Millisecond
)

type CompletionRetryPolicy struct {
	retryCount    int
	retryInterval time.Duration
}

func NewCompletionRetryPolicy(
	retryCount int,
	retryInterval time.Duration,
) CompletionRetryPolicy {
	if retryCount < 0 {
		panic("completion retry count must be >= 0")
	}
	if retryInterval < 0 {
		panic("completion retry interval must be >= 0")
	}
	return CompletionRetryPolicy{
		retryCount:    retryCount,
		retryInterval: retryInterval,
	}
}

func DefaultCompletionRetryPolicy() CompletionRetryPolicy {
	return NewCompletionRetryPolicy(
		defaultCompletionRetryCount,
		defaultCompletionRetryInterval,
	)
}

func (p CompletionRetryPolicy) maxAttempts() int {
	return p.retryCount + 1
}

func (p CompletionRetryPolicy) retryIntervalOrZero() time.Duration {
	return p.retryInterval
}
