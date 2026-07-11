package execution

import (
	"context"
	"fmt"
)

const persistentHandshakeTraceID = "bridge-exec-persistent-handshake"

func (c *PersistentNativeClient) nextPersistentRequest(
	action string,
	params map[string]any,
	traceID string,
) request {
	req := newRequest(action, params, traceID)
	req.RequestID = c.nextRequestIDLocked()
	return req
}

func (c *PersistentNativeClient) exchangePersistentRequestLocked(
	ctx context.Context,
	req request,
	writeLabel string,
	mismatchLabel string,
) (map[string]any, error) {
	if err := writeFrame(c.stdin, req); err != nil {
		_ = c.stopProcessLocked(true)
		return nil, fmt.Errorf("%s: %w", writeLabel, err)
	}

	resp, err := c.readFrameWithContextLocked(ctx)
	if err != nil {
		return nil, err
	}
	return c.validatePersistentResponseLocked(req, resp, mismatchLabel)
}

func (c *PersistentNativeClient) validatePersistentResponseLocked(
	req request,
	resp *response,
	mismatchLabel string,
) (map[string]any, error) {
	if resp.RequestID != req.RequestID {
		_ = c.stopProcessLocked(true)
		return nil, fmt.Errorf("%s: got %q want %q", mismatchLabel, resp.RequestID, req.RequestID)
	}

	payload, err := resp.intoResult(req)
	if err != nil {
		return nil, err
	}
	c.verified = true
	return payload, nil
}
