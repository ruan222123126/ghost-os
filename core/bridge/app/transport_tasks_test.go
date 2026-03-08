package app

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"ghost-os/bridge/session"
)

func TestHandleTasksCRUD(t *testing.T) {
	handler := newTestHandler(t, nil)

	createResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/tasks",
		`{"message":"hello from task","interval_seconds":60}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: got %d want %d body=%s", createResp.Code, http.StatusCreated, createResp.Body.String())
	}
	createBody := decodeResponseBody(t, createResp)
	created, ok := createBody.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected create payload type: %T", createBody.Payload)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created task id should not be empty")
	}
	if created["schedule_type"] != taskScheduleTypeInterval {
		t.Fatalf("unexpected schedule_type: got %v want %q", created["schedule_type"], taskScheduleTypeInterval)
	}
	if created["next_run_at"] == "" {
		t.Fatal("next_run_at should not be empty")
	}

	listResp := serveRequest(handler, http.MethodGet, "/api/tasks", "", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected list status: got %d want %d", listResp.Code, http.StatusOK)
	}
	listBody := decodeResponseBody(t, listResp)
	items, ok := listBody.Payload.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected task list payload: %#v", listBody.Payload)
	}

	getResp := serveRequest(handler, http.MethodGet, "/api/tasks/"+id, "", nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("unexpected get status: got %d want %d", getResp.Code, http.StatusOK)
	}

	deleteResp := serveRequest(handler, http.MethodDelete, "/api/tasks/"+id, "", nil)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("unexpected delete status: got %d want %d", deleteResp.Code, http.StatusOK)
	}

	getAfterDelete := serveRequest(handler, http.MethodGet, "/api/tasks/"+id, "", nil)
	if getAfterDelete.Code != http.StatusNotFound {
		t.Fatalf("unexpected get-after-delete status: got %d want %d", getAfterDelete.Code, http.StatusNotFound)
	}
}

func TestHandleTaskCreateValidation(t *testing.T) {
	handler := newTestHandler(t, nil)

	missingSchedule := serveRequest(
		handler,
		http.MethodPost,
		"/api/tasks",
		`{"message":"hello"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if missingSchedule.Code != http.StatusBadRequest {
		t.Fatalf("unexpected missing schedule status: got %d want %d", missingSchedule.Code, http.StatusBadRequest)
	}

	bothSchedule := serveRequest(
		handler,
		http.MethodPost,
		"/api/tasks",
		`{"message":"hello","interval_seconds":60,"cron_expr":"*/5 * * * *"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if bothSchedule.Code != http.StatusBadRequest {
		t.Fatalf("unexpected dual schedule status: got %d want %d", bothSchedule.Code, http.StatusBadRequest)
	}

	missingSession := serveRequest(
		handler,
		http.MethodPost,
		"/api/tasks",
		`{"message":"hello","session_id":"session-missing","interval_seconds":60}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if missingSession.Code != http.StatusNotFound {
		t.Fatalf("unexpected missing session status: got %d want %d body=%s", missingSession.Code, http.StatusNotFound, missingSession.Body.String())
	}
	cronResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/tasks",
		`{"message":"hello","cron_expr":"*/5 * * * *"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if cronResp.Code != http.StatusCreated {
		t.Fatalf("unexpected cron create status: got %d want %d body=%s", cronResp.Code, http.StatusCreated, cronResp.Body.String())
	}
}

func TestBusTaskCreateAndList(t *testing.T) {
	handler := newTestHandler(t, nil)
	createResp := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"TASK_CREATE","params":{"message":"via-bus","interval_seconds":60},"trace_id":"trace-task-create"}`, nil)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: got %d want %d body=%s", createResp.Code, http.StatusCreated, createResp.Body.String())
	}
	listResp := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"TASK_LIST","params":{},"trace_id":"trace-task-list"}`, nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected list status: got %d want %d body=%s", listResp.Code, http.StatusOK, listResp.Body.String())
	}
	body := decodeResponseBody(t, listResp)
	items, ok := body.Payload.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected bus task list payload: %#v", body.Payload)
	}
}

func TestHandleTaskUpdateDisableAndRunLogs(t *testing.T) {
	handler := newTestHandler(t, func(_ context.Context, message string, requestSessionID string, _ string, _ *ConfigStore, _ *session.Store) (string, string, error) {
		if requestSessionID == "" {
			requestSessionID = "session-run-now"
		}
		return "ran: " + message, requestSessionID, nil
	})

	createResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/tasks",
		`{"message":"hello from task","interval_seconds":60}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: got %d want %d body=%s", createResp.Code, http.StatusCreated, createResp.Body.String())
	}
	id, _ := decodeResponseBody(t, createResp).Payload.(map[string]any)["id"].(string)

	patchResp := serveRequest(
		handler,
		http.MethodPatch,
		"/api/tasks/"+id,
		`{"enabled":false,"message":"paused task"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if patchResp.Code != http.StatusOK {
		t.Fatalf("unexpected patch status: got %d want %d body=%s", patchResp.Code, http.StatusOK, patchResp.Body.String())
	}
	patched := decodeResponseBody(t, patchResp)
	updated, ok := patched.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected patch payload: %#v", patched.Payload)
	}
	if enabled, _ := updated["enabled"].(bool); enabled {
		t.Fatalf("expected task to be disabled, payload=%#v", updated)
	}
	if nextRunAt, _ := updated["next_run_at"].(string); nextRunAt != "" && nextRunAt != "0001-01-01T00:00:00Z" {
		t.Fatalf("expected disabled task to have no effective next_run_at, got %q", nextRunAt)
	}

	runResp := serveRequest(handler, http.MethodPost, "/api/tasks/"+id+"/run", "", nil)
	if runResp.Code != http.StatusOK {
		t.Fatalf("unexpected run-now status: got %d want %d body=%s", runResp.Code, http.StatusOK, runResp.Body.String())
	}
	runPayload, ok := decodeResponseBody(t, runResp).Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected run-now payload: %#v", decodeResponseBody(t, runResp).Payload)
	}
	runData, _ := runPayload["run"].(map[string]any)
	if runData["status"] != taskRunStatusSuccess {
		t.Fatalf("unexpected run-now status payload: %#v", runData)
	}

	logsResp := serveRequest(handler, http.MethodGet, "/api/tasks/"+id+"/logs?limit=1", "", nil)
	if logsResp.Code != http.StatusOK {
		t.Fatalf("unexpected logs status: got %d want %d body=%s", logsResp.Code, http.StatusOK, logsResp.Body.String())
	}
	items, ok := decodeResponseBody(t, logsResp).Payload.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected logs payload: %#v", decodeResponseBody(t, logsResp).Payload)
	}
	entry, _ := items[0].(map[string]any)
	if entry["status"] != taskRunStatusSuccess {
		t.Fatalf("unexpected logs entry: %#v", entry)
	}
}

func TestBusTaskUpdateAndLogs(t *testing.T) {
	handler := newTestHandler(t, nil)
	createResp := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"TASK_CREATE","params":{"message":"via-bus","interval_seconds":60},"trace_id":"trace-task-create"}`, nil)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: got %d want %d body=%s", createResp.Code, http.StatusCreated, createResp.Body.String())
	}
	created, _ := decodeResponseBody(t, createResp).Payload.(map[string]any)
	id, _ := created["id"].(string)

	updateResp := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"TASK_UPDATE","params":{"id":"`+id+`","enabled":false},"trace_id":"trace-task-update"}`, nil)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("unexpected update status: got %d want %d body=%s", updateResp.Code, http.StatusOK, updateResp.Body.String())
	}
	logsResp := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"TASK_LOGS","params":{"id":"`+id+`","limit":5},"trace_id":"trace-task-logs"}`, nil)
	if logsResp.Code != http.StatusOK {
		t.Fatalf("unexpected logs status: got %d want %d body=%s", logsResp.Code, http.StatusOK, logsResp.Body.String())
	}
}

func TestTaskSchedulerUsesProvidedSessionAndLogsRun(t *testing.T) {
	_, service, sessionStore := newTestHandlerWithService(t, func(_ context.Context, _ string, requestSessionID string, _ string, _ *ConfigStore, _ *session.Store) (string, string, error) {
		return "done", requestSessionID, nil
	}, nil)

	sess := session.NewSession("system")
	sess.ID = "session-task-1"
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	payload, code, err := service.executeTaskCreateAction(taskCreateParams{
		Message:         "run task",
		SessionID:       sess.ID,
		IntervalSeconds: 1,
	}, "trace-task")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create task: code=%d err=%v", code, err)
	}
	task := payload.(taskPayload)
	logs := waitForTaskLogs(t, service.taskStore, task.ID, 1, 3*time.Second)
	logEntry := findTaskLogByStatus(t, logs, taskRunStatusSuccess)
	if logEntry.SessionIDInput != sess.ID {
		t.Fatalf("unexpected session_id_input: got %q want %q", logEntry.SessionIDInput, sess.ID)
	}
	if logEntry.SessionIDOutput != sess.ID {
		t.Fatalf("unexpected session_id_output: got %q want %q", logEntry.SessionIDOutput, sess.ID)
	}
	if logEntry.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected log status: got %q want %q", logEntry.Status, taskRunStatusSuccess)
	}
	stored, err := service.taskStore.LoadTask(task.ID)
	if err != nil {
		t.Fatalf("load task: %v", err)
	}
	if stored.LastRunAt.IsZero() {
		t.Fatal("last_run_at should be set after execution")
	}
	service.taskScheduler.Stop()
}

func TestTaskSchedulerLeavesSessionBlankToOpenNewConversation(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, func(_ context.Context, _ string, requestSessionID string, _ string, _ *ConfigStore, _ *session.Store) (string, string, error) {
		if requestSessionID != "" {
			t.Fatalf("expected empty session_id, got %q", requestSessionID)
		}
		return "created", "session-random-1", nil
	}, nil)

	payload, code, err := service.executeTaskCreateAction(taskCreateParams{
		Message:         "new chat",
		IntervalSeconds: 1,
	}, "trace-task")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create task: code=%d err=%v", code, err)
	}
	task := payload.(taskPayload)
	logs := waitForTaskLogs(t, service.taskStore, task.ID, 1, 3*time.Second)
	logEntry := findTaskLogByStatus(t, logs, taskRunStatusSuccess)
	if logEntry.SessionIDInput != "" {
		t.Fatalf("unexpected session_id_input: got %q want empty", logEntry.SessionIDInput)
	}
	if logEntry.SessionIDOutput != "session-random-1" {
		t.Fatalf("unexpected session_id_output: got %q want %q", logEntry.SessionIDOutput, "session-random-1")
	}
	service.taskScheduler.Stop()
}

func TestTaskSchedulerSkipsConcurrentRun(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, func(_ context.Context, _ string, _ string, _ string, _ *ConfigStore, _ *session.Store) (string, string, error) {
		time.Sleep(1500 * time.Millisecond)
		return "done", "session-task-skip", nil
	}, nil)

	payload, code, err := service.executeTaskCreateAction(taskCreateParams{
		Message:         "slow task",
		IntervalSeconds: 1,
	}, "trace-task")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create task: code=%d err=%v", code, err)
	}
	task := payload.(taskPayload)
	logs := waitForTaskLogs(t, service.taskStore, task.ID, 2, 4*time.Second)
	seenSkipped := false
	for _, item := range logs {
		if item.Status == taskRunStatusSkipped {
			seenSkipped = true
			break
		}
	}
	if !seenSkipped {
		t.Fatalf("expected skipped run log, got %#v", logs)
	}
	service.taskScheduler.Stop()
}

func TestTaskSchedulerStopWaitsForRunningTask(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan struct{})
	_, service, _ := newTestHandlerWithService(t, func(_ context.Context, _ string, _ string, _ string, _ *ConfigStore, _ *session.Store) (string, string, error) {
		select {
		case <-started:
		default:
			close(started)
		}
		time.Sleep(200 * time.Millisecond)
		close(finished)
		return "done", "session-task-stop", nil
	}, nil)

	payload, code, err := service.executeTaskCreateAction(taskCreateParams{
		Message:         "slow shutdown",
		IntervalSeconds: 1,
	}, "trace-task")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create task: code=%d err=%v", code, err)
	}
	_ = payload.(taskPayload)

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for task to start")
	}

	stopStarted := time.Now()
	service.taskScheduler.Stop()
	if time.Since(stopStarted) < 150*time.Millisecond {
		t.Fatalf("expected stop to wait for running task, duration=%s", time.Since(stopStarted))
	}
	select {
	case <-finished:
	default:
		t.Fatal("task execution should finish before Stop returns")
	}
}

func waitForTaskLogs(t *testing.T, store *TaskStore, taskID string, minCount int, timeout time.Duration) []TaskRunLog {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		logs := readTaskLogs(t, store, taskID)
		if len(logs) >= minCount {
			return logs
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d task logs for %s", minCount, taskID)
	return nil
}

func readTaskLogs(t *testing.T, store *TaskStore, taskID string) []TaskRunLog {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(store.logsDir, taskID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read task log dir: %v", err)
	}
	logs := make([]TaskRunLog, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(store.logsDir, taskID, entry.Name()))
		if err != nil {
			t.Fatalf("read task log file: %v", err)
		}
		var logEntry TaskRunLog
		if err := json.Unmarshal(data, &logEntry); err != nil {
			t.Fatalf("decode task log: %v", err)
		}
		logs = append(logs, logEntry)
	}
	sort.Slice(logs, func(i, j int) bool {
		left := logs[i].StartedAt
		if left.IsZero() {
			left = logs[i].ScheduledAt
		}
		right := logs[j].StartedAt
		if right.IsZero() {
			right = logs[j].ScheduledAt
		}
		return left.Before(right)
	})
	return logs
}

func findTaskLogByStatus(t *testing.T, logs []TaskRunLog, status string) TaskRunLog {
	t.Helper()
	for _, item := range logs {
		if item.Status == status {
			return item
		}
	}
	t.Fatalf("task log with status %q not found in %#v", status, logs)
	return TaskRunLog{}
}
