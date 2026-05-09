package mode

import (
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/session"
)

func TestParseProRequest(t *testing.T) {
	request, matched, err := ParseProRequest("pro fix config", 20)
	if err != nil {
		t.Fatalf("parse pro mode request: %v", err)
	}
	if !matched {
		t.Fatal("expected pro prefix to match")
	}
	if request.Mode != Pro || request.MaxIterations != 20 || request.OriginalTask != "fix config" {
		t.Fatalf("unexpected request: %+v", request)
	}

	request, matched, err = ParseProRequest("pro 7 finish task", 20)
	if err != nil {
		t.Fatalf("parse explicit pro iterations: %v", err)
	}
	if !matched {
		t.Fatal("expected explicit pro prefix to match")
	}
	if request.Mode != Pro || request.MaxIterations != 7 || request.OriginalTask != "finish task" {
		t.Fatalf("unexpected explicit pro request: %+v", request)
	}

	_, matched, err = ParseProRequest("prox finish task", 20)
	if err == nil || !strings.Contains(err.Error(), "prox mode has been removed") {
		t.Fatalf("expected removed prox error, got matched=%t err=%v", matched, err)
	}
	if !matched {
		t.Fatal("removed prox prefix should be handled explicitly")
	}
}

func TestBuildUserPromptIncludesHistory(t *testing.T) {
	request := ProRequest{Mode: Pro, OriginalTask: "fix build", MaxIterations: 3}
	records := []session.IterationRecord{{
		Iteration:  1,
		Did:        "checked logs",
		Remaining:  "patch parser",
		RecordedAt: time.Now().UTC(),
	}}

	prompt := BuildUserPrompt(request, records, 2)
	if !strings.Contains(prompt, "did: checked logs") {
		t.Fatalf("history not present in prompt: %q", prompt)
	}
	if !strings.Contains(prompt, "Iteration: 2") {
		t.Fatalf("iteration line missing: %q", prompt)
	}
}

func TestResultStatus(t *testing.T) {
	if got := ResultStatus(StopCompleted); got != StatusCompleted {
		t.Fatalf("unexpected status for completed: %q", got)
	}
	if got := ResultStatus(StopMaxLimit); got != StatusIncomplete {
		t.Fatalf("unexpected status for max limit: %q", got)
	}
}
