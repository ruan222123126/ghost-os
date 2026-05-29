package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/session"
)

type taskRunLogResponsePayload struct {
	RunID           string `json:"run_id"`
	Status          string `json:"status"`
	SessionIDOutput string `json:"session_id_output"`
}

func TestHandleTaskRunMakesRunningLogVisibleImmediately(t *testing.T) {
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
	done := make(chan int, 1)
	go func() {
		recorder := serveRequest(handler, http.MethodPost, "/api/tasks/"+taskID+"/run", "", nil)
		done <- recorder.Code
	}()

	select {
	case <-started:
	case <-ctxTimeout():
		t.Fatal("timed out waiting for task run to start")
	}
	logs := listTaskRunLogs(t, handler, taskID)
	if len(logs) != 1 {
		t.Fatalf("expected one running log, got %#v", logs)
	}
	if logs[0].Status != "running" || logs[0].SessionIDOutput == "" {
		t.Fatalf("unexpected running log: %#v", logs[0])
	}

	close(release)
	if code := <-done; code != http.StatusOK {
		t.Fatalf("unexpected run response status: %d", code)
	}
}

func createRunnableAgentTask(t *testing.T, handler http.Handler) string {
	t.Helper()

	recorder := serveRequest(handler, http.MethodPost, "/api/tasks", `{
		"task_kind":"agent_message",
		"message":"run visible",
		"interval_seconds":60
	}`, nil)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create task status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	return decodeTaskResponsePayload(t, recorder).ID
}

func listTaskRunLogs(
	t *testing.T,
	handler http.Handler,
	taskID string,
) []taskRunLogResponsePayload {
	t.Helper()

	recorder := serveRequest(handler, http.MethodGet, "/api/tasks/"+taskID+"/logs", "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list logs status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	body := decodeResponseBody(t, recorder)
	data, err := json.Marshal(body.Payload)
	if err != nil {
		t.Fatalf("marshal logs payload: %v", err)
	}
	var logs []taskRunLogResponsePayload
	if err := json.Unmarshal(data, &logs); err != nil {
		t.Fatalf("decode logs payload: %v", err)
	}
	return logs
}

func ctxTimeout() <-chan time.Time {
	return time.After(2 * time.Second)
}
