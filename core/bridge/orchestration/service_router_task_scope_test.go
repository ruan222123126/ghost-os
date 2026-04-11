package orchestration

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestNormalizeTaskListScope(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		wantScope string
		wantErr   bool
	}{
		{name: "default_empty", input: "", wantScope: taskListScopeUser},
		{name: "user_scope", input: "user", wantScope: taskListScopeUser},
		{name: "trimmed_user_scope", input: " user ", wantScope: taskListScopeUser},
		{name: "system_scope", input: "system", wantScope: taskListScopeSystem},
		{name: "invalid_scope", input: "admin", wantErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeTaskListScope(tc.input)
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

func TestTaskListActionRejectsInvalidScope(t *testing.T) {
	service := newBridgeServiceState(nil, nil)
	registerTaskActions(service)

	handler, ok := service.actionHandler(busActionTaskList)
	if !ok {
		t.Fatalf("task list action handler is not registered")
	}

	_, err := handler(context.Background(), json.RawMessage(`{"scope":"invalid"}`), "trace-invalid-task-scope")
	if code := legacyStatusFromServiceError(err); code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got=%d want=%d", code, http.StatusBadRequest)
	}
	if err == nil || !strings.Contains(err.Error(), "invalid task scope") {
		t.Fatalf("unexpected error: %v", err)
	}
}
