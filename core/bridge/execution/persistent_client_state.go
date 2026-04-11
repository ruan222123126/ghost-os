package execution

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func (c *PersistentNativeClient) callLocked(
	ctx context.Context,
	action string,
	params map[string]any,
	traceID string,
) (map[string]any, error) {
	if c.closed {
		return nil, errors.New("persistent native client is closed")
	}

	req := c.nextPersistentRequest(action, params, traceID)
	if err := c.ensurePersistentReadyLocked(ctx); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	payload, err := c.exchangePersistentRequestLocked(
		ctx,
		req,
		"write native frame failed",
		"unexpected native response request_id",
	)
	if err != nil {
		return nil, err
	}

	c.reapExitedProcessLocked()
	return payload, nil
}

func (c *PersistentNativeClient) ensurePersistentReadyLocked(ctx context.Context) error {
	if err := c.ensureStartedLocked(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.verified {
		return nil
	}

	if err := c.verifyPersistentProtocolLocked(ctx); err != nil {
		return err
	}
	if c.cmd != nil {
		return nil
	}
	return c.ensureStartedLocked()
}

func (c *PersistentNativeClient) readFrameWithContextLocked(ctx context.Context) (*response, error) {
	type result struct {
		response response
		err      error
	}

	resultCh := make(chan result, 1)
	go func(reader *bufio.Reader) {
		var resp response
		err := readFrame(reader, &resp)
		resultCh <- result{response: resp, err: err}
	}(c.stdoutReader)

	select {
	case <-ctx.Done():
		stopErr := c.stopProcessLocked(true)
		if stopErr != nil {
			return nil, fmt.Errorf("%w (native shutdown: %v)", ctx.Err(), stopErr)
		}
		return nil, ctx.Err()
	case res := <-resultCh:
		if res.err != nil {
			_ = c.stopProcessLocked(true)
			return nil, fmt.Errorf("read native frame failed: %w", res.err)
		}
		return &res.response, nil
	}
}

func (c *PersistentNativeClient) verifyPersistentProtocolLocked(ctx context.Context) error {
	timeout := c.handshakeTimeout
	if timeout <= 0 {
		timeout = defaultPersistentHandshakeTimeout
	}
	handshakeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req := c.nextPersistentRequest("PING", map[string]any{}, persistentHandshakeTraceID)
	if _, err := c.exchangePersistentRequestLocked(
		handshakeCtx,
		req,
		"write persistent handshake failed",
		"unexpected native handshake request_id",
	); err != nil {
		return fmt.Errorf("persistent handshake failed: %w", err)
	}

	c.reapExitedProcessLocked()
	return nil
}

func persistentHandshakeTimeoutFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv(persistentHandshakeTimeoutEnv))
	if raw == "" {
		return defaultPersistentHandshakeTimeout
	}
	if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
		return parsed
	}
	if ms, err := strconv.Atoi(raw); err == nil && ms > 0 {
		return time.Duration(ms) * time.Millisecond
	}
	return defaultPersistentHandshakeTimeout
}
