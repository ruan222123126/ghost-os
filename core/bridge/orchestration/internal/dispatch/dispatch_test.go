package dispatch

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"ghost-os/bridge/orchestration/internal/app/tasks"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

func TestNormalizeTaskListScope(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		wantScope string
		wantErr   bool
	}{
		{name: "default_empty", input: "", wantScope: tasks.ScopeUser},
		{name: "user_scope", input: "user", wantScope: tasks.ScopeUser},
		{name: "trimmed_user_scope", input: " user ", wantScope: tasks.ScopeUser},
		{name: "system_scope", input: "system", wantScope: tasks.ScopeSystem},
		{name: "orchestration_scope", input: "orchestration", wantScope: tasks.ScopeOrchestration},
		{name: "invalid_scope", input: "admin", wantErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeTaskListScope(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for input=%q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input=%q: %v", tc.input, err)
			}
			if got != tc.wantScope {
				t.Fatalf("unexpected scope for input=%q: got=%q want=%q", tc.input, got, tc.wantScope)
			}
		})
	}
}

func TestRegisterTypedRejectsInvalidTaskScope(t *testing.T) {
	router := NewRouter(1)
	RegisterTyped[api.TaskListParams](router, bus.ActionTaskList, func(_ context.Context, params api.TaskListParams, _ string) (bus.ServiceResult, error) {
		scope, err := NormalizeTaskListScope(params.Scope)
		if err != nil {
			return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
		}
		return bus.ResultSuccess(scope), nil
	})

	_, err := router.Dispatch(context.Background(), bus.ActionTaskList, json.RawMessage(`{"scope":"invalid"}`), "trace-invalid-task-scope")
	if bus.ErrorKindOf(err) != bus.ServiceErrorInvalidInput {
		t.Fatalf("unexpected error kind: got=%s err=%v", bus.ErrorKindOf(err), err)
	}
	if err == nil || !strings.Contains(err.Error(), "invalid task scope") {
		t.Fatalf("unexpected error: %v", err)
	}
}
