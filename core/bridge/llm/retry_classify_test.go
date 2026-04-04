package llm

import (
	"context"
	"fmt"
	"testing"
)

func TestIsTransientCompletionErrorStatusCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "status 429 retryable",
			err:  &completionStatusError{statusCode: 429},
			want: true,
		},
		{
			name: "status 503 retryable",
			err:  &completionStatusError{statusCode: 503},
			want: true,
		},
		{
			name: "status 400 not retryable",
			err:  &completionStatusError{statusCode: 400},
			want: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTransientCompletionError(tc.err); got != tc.want {
				t.Fatalf("unexpected result: got %t want %t", got, tc.want)
			}
		})
	}
}

func TestIsTransientCompletionErrorContextCancelled(t *testing.T) {
	if IsTransientCompletionError(context.Canceled) {
		t.Fatal("context cancellation should not be retryable")
	}
	if IsTransientCompletionError(context.DeadlineExceeded) {
		t.Fatal("context deadline exceeded should not be retryable")
	}
}

func TestIsTransientCompletionErrorNetworkError(t *testing.T) {
	netErr := transientCompletionNetError{message: "timeout"}
	if !IsTransientCompletionError(netErr) {
		t.Fatal("network timeout error should be retryable")
	}
	if !IsTransientCompletionError(fmt.Errorf("wrapped: %w", netErr)) {
		t.Fatal("wrapped network timeout error should be retryable")
	}
}

type transientCompletionNetError struct {
	message string
}

func (e transientCompletionNetError) Error() string {
	return e.message
}

func (transientCompletionNetError) Timeout() bool {
	return true
}

func (transientCompletionNetError) Temporary() bool {
	return true
}
