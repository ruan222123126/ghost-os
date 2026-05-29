package transport

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	bridgeorchestration "ghost-os/bridge/orchestration"
)

type sessionSourcesPayload struct {
	Assignments      map[string]sessionSourceAssignmentPayload `json:"assignments"`
	HiddenSessionIDs []string                                  `json:"hidden_session_ids"`
}

type sessionSourceAssignmentPayload struct {
	Kind      string `json:"kind"`
	OwnerID   string `json:"owner_id"`
	OwnerName string `json:"owner_name"`
}

func TestHandleSessionSourcesReturnsAggregatedAssignments(t *testing.T) {
	handler := newTestHandler(t, nil)
	store := requireTaskStoreForTest(t)
	task := bridgeorchestration.ScheduledTask{
		ID:              "loop-task",
		Message:         "Loop task",
		AgentMode:       "relay",
		TaskKind:        "agent_message",
		ScheduleType:    "interval",
		IntervalSeconds: 60,
		Enabled:         true,
	}
	if err := store.SaveTask(&task); err != nil {
		t.Fatalf("save task: %v", err)
	}
	if err := store.AppendRunLog(sessionSourceRunLog(task.ID, "loop-session")); err != nil {
		t.Fatalf("append run log: %v", err)
	}

	recorder := serveRequest(handler, http.MethodGet, "/api/sessions/sources", "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("session sources status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	payload := decodeSessionSourcesPayload(t, decodeResponseBody(t, recorder).Payload)
	assignment := payload.Assignments["loop-session"]
	if assignment.Kind != "loop" || assignment.OwnerID != task.ID || assignment.OwnerName != task.Message {
		t.Fatalf("unexpected source assignment: %#v", assignment)
	}
	if len(payload.HiddenSessionIDs) != 0 {
		t.Fatalf("unexpected hidden sessions: %#v", payload.HiddenSessionIDs)
	}
}

func requireTaskStoreForTest(t *testing.T) *bridgeorchestration.TaskStore {
	t.Helper()
	store, err := bridgeorchestration.NewTaskStore(os.Getenv("GHOST_TASKS_PATH"))
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}
	return store
}

func sessionSourceRunLog(taskID string, sessionID string) bridgeorchestration.TaskRunLog {
	now := time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC)
	return bridgeorchestration.TaskRunLog{
		TaskID:          taskID,
		RunID:           "run-1",
		TraceID:         "trace-1",
		TaskKind:        "agent_message",
		ScheduledAt:     now,
		Status:          "success",
		SessionIDOutput: sessionID,
	}
}

func decodeSessionSourcesPayload(t *testing.T, payload any) sessionSourcesPayload {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal session sources payload: %v", err)
	}
	var decoded sessionSourcesPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode session sources payload: %v", err)
	}
	return decoded
}
