package execution

import (
	"errors"
	"fmt"
	"time"
)

var errPersistentProtocolUnsupported = errors.New("persistent native protocol unsupported")

type request struct {
	Action    string         `json:"action"`
	Params    map[string]any `json:"params"`
	TraceID   string         `json:"trace_id"`
	RequestID string         `json:"request_id,omitempty"`
}

type response struct {
	Status    string         `json:"status"`
	Payload   map[string]any `json:"payload"`
	Error     string         `json:"error"`
	RequestID string         `json:"request_id,omitempty"`
}

func newRequest(action string, params map[string]any, traceID string) request {
	if traceID == "" {
		traceID = fmt.Sprintf("bridge-exec-%d", time.Now().UnixNano())
	}
	if params == nil {
		params = map[string]any{}
	}
	return request{
		Action:  action,
		Params:  params,
		TraceID: traceID,
	}
}

func (resp response) intoResult(req request) (map[string]any, error) {
	if resp.Status != "success" {
		if resp.Error == "" {
			resp.Error = "execution returned error"
		}
		requestID := resp.RequestID
		if requestID == "" {
			requestID = req.RequestID
		}
		return nil, fmt.Errorf(
			"native execution error: action=%s trace_id=%s request_id=%s: %s",
			req.Action,
			req.TraceID,
			requestID,
			resp.Error,
		)
	}
	if resp.Payload == nil {
		resp.Payload = map[string]any{}
	}
	return resp.Payload, nil
}
