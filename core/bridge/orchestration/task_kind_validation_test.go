package orchestration

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestValidateTaskDefinitionRejectsUnsupportedTaskKind(t *testing.T) {
	task := ScheduledTask{
		TaskKind: "unexpected_kind",
		Message:  "hello",
	}

	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), `unsupported task_kind "unexpected_kind"`) {
		t.Fatalf("expected unsupported task_kind error, got %v", err)
	}
}

func TestTaskCreateRejectsUnsupportedTaskKind(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	_, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        "unexpected_kind",
		Message:         "hello",
		IntervalSeconds: 60,
	}, "trace-invalid-kind")
	if err == nil {
		t.Fatal("expected task create to fail")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), `unsupported task_kind "unexpected_kind"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskExecutorAdapterRejectsUnsupportedTaskKind(t *testing.T) {
	result := taskExecutorAdapter{}.Execute(context.Background(), ScheduledTask{
		TaskKind: "unexpected_kind",
		Message:  "hello",
	}, "trace-invalid-kind")
	if result.Status != taskRunStatusError {
		t.Fatalf("unexpected execution status: got %q want %q", result.Status, taskRunStatusError)
	}
	if result.Error != `unsupported task_kind "unexpected_kind"` {
		t.Fatalf("unexpected execution error: %q", result.Error)
	}
}
