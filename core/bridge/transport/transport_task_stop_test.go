package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/session"
)

type taskStopResponsePayload struct {
	Status string                     `json:"status"`
	TaskID string                     `json:"task_id"`
	RunID  string                     `json:"run_id"`
	Run    *taskRunLogResponsePayload `json:"run,omitempty"`
}

func TestHandleTaskStopCancelsRunningTask(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	handler := newTestHandler(t, func(
		ctx context.Context,
		_ string,
		sessionID string,
		_ string,
		_ bridgeconfig.Store,
		_ *session.Store,
	) (string, string, error) {
		close(started)
		select {
		case <-release:
			return "ok", sessionID, nil
		case <-ctx.Done():
			return "", sessionID, ctx.Err()
		}
	})

	taskID := createRunnableAgentTask(t, handler)
	runRecorder := serveRequest(handler, http.MethodPost, "/api/tasks/"+taskID+"/run?start_only=1", "", nil)
	if runRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected start-only status: %d", runRecorder.Code)
	}

	select {
	case <-started:
	case <-ctxTimeout():
		t.Fatal("timed out waiting for task run to start")
	}

	logs := listTaskRunLogs(t, handler, taskID)
	if len(logs) != 1 {
		t.Fatalf("expected one running log, got %#v", logs)
	}
	stopRecorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/tasks/"+taskID+"/stop",
		`{"run_id":"`+logs[0].RunID+`"}`,
		nil,
	)
	if stopRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected stop status: %d body=%s", stopRecorder.Code, stopRecorder.Body.String())
	}

	body := decodeResponseBody(t, stopRecorder)
	data, err := json.Marshal(body.Payload)
	if err != nil {
		t.Fatalf("marshal stop payload: %v", err)
	}
	var payload taskStopResponsePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode stop payload: %v", err)
	}
	if payload.Status != "stopped" || payload.TaskID != taskID || payload.Run == nil {
		t.Fatalf("unexpected stop payload: %#v", payload)
	}
	if payload.Run.Status != "cancelled" {
		t.Fatalf("unexpected final run payload: %#v", payload.Run)
	}
	if payload.Run.Error != "" {
		t.Fatalf("cancelled run should not report error: %#v", payload.Run)
	}
}

func TestHandleTaskStopReturnsNotRunningForUnknownRun(t *testing.T) {
	handler := newTestHandler(t, nil)
	taskID := createRunnableAgentTask(t, handler)

	stopRecorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/tasks/"+taskID+"/stop",
		`{"run_id":"missing-run"}`,
		nil,
	)
	if stopRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected stop status: %d body=%s", stopRecorder.Code, stopRecorder.Body.String())
	}

	body := decodeResponseBody(t, stopRecorder)
	data, err := json.Marshal(body.Payload)
	if err != nil {
		t.Fatalf("marshal stop payload: %v", err)
	}
	var payload taskStopResponsePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode stop payload: %v", err)
	}
	if payload.Status != "not_running" || payload.TaskID != taskID || payload.Run != nil {
		t.Fatalf("unexpected stop payload: %#v", payload)
	}
}
