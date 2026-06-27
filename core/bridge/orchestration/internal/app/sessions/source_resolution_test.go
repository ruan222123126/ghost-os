package sessions

import (
	"testing"
	"time"

	"ghost-os/bridge/session"
	bridgeTasks "ghost-os/bridge/tasks"
)

func TestBuildSessionSourceResolutionHidesUnfinishedRunSessions(t *testing.T) {
	resolution, err := BuildSessionSourceResolution([]SessionSourceTask{{
		ID:       "task-1",
		TaskKind: bridgeTasks.KindAgentMessage,
		RunLogs: []bridgeTasks.RunLog{
			runLogForSourceTest("task-1", bridgeTasks.KindAgentMessage, bridgeTasks.RunStatusRunning, "running-session"),
			runLogForSourceTest("task-1", bridgeTasks.KindAgentMessage, bridgeTasks.RunStatusAwaitingHuman, "awaiting-session"),
			runLogForSourceTest("task-1", bridgeTasks.KindAgentMessage, bridgeTasks.RunStatusSuccess, "done-session"),
		},
	}})
	if err != nil {
		t.Fatalf("build source resolution: %v", err)
	}

	if _, ok := resolution.Assignments["running-session"]; ok {
		t.Fatalf("running session should not be assigned: %#v", resolution.Assignments)
	}
	if _, ok := resolution.Assignments["awaiting-session"]; ok {
		t.Fatalf("awaiting-human session should not be assigned: %#v", resolution.Assignments)
	}
	if got := resolution.Assignments["done-session"].Kind; got != SessionSourceKindTask {
		t.Fatalf("done session assignment kind: got %q want %q", got, SessionSourceKindTask)
	}
	assertStringSet(t, resolution.HiddenSessionIDs, []string{"running-session", "awaiting-session"})
}

func TestBuildSessionSourceResolutionShowsErrorAndCancelledRunSessions(t *testing.T) {
	resolution, err := BuildSessionSourceResolution([]SessionSourceTask{{
		ID:       "task-1",
		TaskKind: bridgeTasks.KindAgentMessage,
		RunLogs: []bridgeTasks.RunLog{
			runLogForSourceTest("task-1", bridgeTasks.KindAgentMessage, bridgeTasks.RunStatusError, "error-session"),
			runLogForSourceTest("task-1", bridgeTasks.KindAgentMessage, bridgeTasks.RunStatusCancelled, "cancelled-session"),
		},
	}})
	if err != nil {
		t.Fatalf("build source resolution: %v", err)
	}

	if got := resolution.Assignments["error-session"].Kind; got != SessionSourceKindTask {
		t.Fatalf("error session assignment kind: got %q want %q", got, SessionSourceKindTask)
	}
	if got := resolution.Assignments["cancelled-session"].Kind; got != SessionSourceKindTask {
		t.Fatalf("cancelled session assignment kind: got %q want %q", got, SessionSourceKindTask)
	}
	if len(resolution.HiddenSessionIDs) != 0 {
		t.Fatalf("terminal sessions should not be hidden: %#v", resolution.HiddenSessionIDs)
	}
}

func TestBuildSessionSourceResolutionHidesUnfinishedWorkflowChildSessions(t *testing.T) {
	run := runLogForSourceTest("workflow-1", bridgeTasks.KindWorkflow, bridgeTasks.RunStatusRunning, "workflow-session")
	run.NodeResults = []bridgeTasks.RunNodeResult{{
		NodeID: "agent",
		Output: map[string]any{
			"session_id_output": "workflow-child-session",
		},
	}}

	resolution, err := BuildSessionSourceResolution([]SessionSourceTask{{
		ID:       "workflow-1",
		TaskKind: bridgeTasks.KindWorkflow,
		RunLogs:  []bridgeTasks.RunLog{run},
	}})
	if err != nil {
		t.Fatalf("build source resolution: %v", err)
	}

	if len(resolution.Assignments) != 0 {
		t.Fatalf("unfinished workflow should not assign sessions: %#v", resolution.Assignments)
	}
	assertStringSet(t, resolution.HiddenSessionIDs, []string{"workflow-session", "workflow-child-session"})
}

func TestServiceHiddenSessionIDSetFiltersOnlyUnfinishedTaskSessions(t *testing.T) {
	service := Service{TaskStore: sessionSourceTaskStoreStub{
		tasks: []bridgeTasks.ScheduledTask{{
			ID:       "task-1",
			TaskKind: bridgeTasks.KindAgentMessage,
		}},
		runsByTaskID: map[string][]bridgeTasks.RunLog{
			"task-1": {
				runLogForSourceTest("task-1", bridgeTasks.KindAgentMessage, bridgeTasks.RunStatusRunning, "running-session"),
				runLogForSourceTest("task-1", bridgeTasks.KindAgentMessage, bridgeTasks.RunStatusError, "error-session"),
				runLogForSourceTest("task-1", bridgeTasks.KindAgentMessage, bridgeTasks.RunStatusCancelled, "cancelled-session"),
			},
		},
	}}

	hiddenSessionIDs, err := service.hiddenSessionIDSet()
	if err != nil {
		t.Fatalf("build hidden session ids: %v", err)
	}
	filtered := filterHiddenSessionMetadata([]session.SessionMetadata{
		{ID: "running-session"},
		{ID: "error-session"},
		{ID: "cancelled-session"},
		{ID: "manual-session"},
	}, hiddenSessionIDs)

	if got := metadataIDs(filtered); !equalStrings(got, []string{"error-session", "cancelled-session", "manual-session"}) {
		t.Fatalf("filtered metadata ids: got %#v", got)
	}
}

func runLogForSourceTest(
	taskID string,
	taskKind string,
	status string,
	sessionID string,
) bridgeTasks.RunLog {
	return bridgeTasks.RunLog{
		TaskID:          taskID,
		RunID:           taskID + "-" + status,
		TraceID:         "trace-" + status,
		TaskKind:        taskKind,
		ScheduledAt:     time.Date(2026, 6, 27, 8, 0, 0, 0, time.UTC),
		Status:          status,
		SessionIDOutput: sessionID,
	}
}

type sessionSourceTaskStoreStub struct {
	tasks        []bridgeTasks.ScheduledTask
	runsByTaskID map[string][]bridgeTasks.RunLog
}

func (s sessionSourceTaskStoreStub) ListTasks() ([]bridgeTasks.ScheduledTask, error) {
	return append([]bridgeTasks.ScheduledTask(nil), s.tasks...), nil
}

func (s sessionSourceTaskStoreStub) ListRunLogs(taskID string, _ int) ([]bridgeTasks.RunLog, error) {
	return append([]bridgeTasks.RunLog(nil), s.runsByTaskID[taskID]...), nil
}

func metadataIDs(metadata []session.SessionMetadata) []string {
	ids := make([]string, 0, len(metadata))
	for _, item := range metadata {
		ids = append(ids, item.ID)
	}
	return ids
}

func assertStringSet(t *testing.T, got []string, want []string) {
	t.Helper()
	gotSet := make(map[string]struct{}, len(got))
	for _, item := range got {
		gotSet[item] = struct{}{}
	}
	if len(gotSet) != len(want) {
		t.Fatalf("unexpected string set length: got %#v want %#v", got, want)
	}
	for _, item := range want {
		if _, ok := gotSet[item]; !ok {
			t.Fatalf("string set missing %q: got %#v want %#v", item, got, want)
		}
	}
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
