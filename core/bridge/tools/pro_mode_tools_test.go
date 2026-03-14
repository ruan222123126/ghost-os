package tools

import (
	"context"
	"testing"
)

func TestProUpdateRecordToolExecuteAndInterpret(t *testing.T) {
	tool := NewProUpdateRecordTool()

	output, err := tool.Execute(context.Background(), []byte(`{"did":"inspected config","remaining":"apply patch and run tests"}`), "trace-pro")
	if err != nil {
		t.Fatalf("execute pro_update_record: %v", err)
	}

	meta := InterpretExecuteResult(tool, output)
	if meta.Iteration == nil {
		t.Fatal("expected iteration handoff signal")
	}
	if meta.Iteration.Completed {
		t.Fatal("record tool should not mark completion")
	}
	if meta.Iteration.Did != "inspected config" {
		t.Fatalf("unexpected did: %q", meta.Iteration.Did)
	}
	if meta.Iteration.Remaining != "apply patch and run tests" {
		t.Fatalf("unexpected remaining: %q", meta.Iteration.Remaining)
	}
}

func TestProCompleteToolExecuteAndInterpret(t *testing.T) {
	tool := NewProCompleteTool()

	output, err := tool.Execute(context.Background(), []byte(`{"did":"patched config and ran tests","remaining":"none","final_message":"done","final_change_log":"updated config and verified tests"}`), "trace-pro")
	if err != nil {
		t.Fatalf("execute pro_complete: %v", err)
	}

	meta := InterpretExecuteResult(tool, output)
	if meta.Iteration == nil {
		t.Fatal("expected iteration handoff signal")
	}
	if !meta.Iteration.Completed {
		t.Fatal("complete tool should mark completion")
	}
	if meta.Iteration.FinalMessage != "done" {
		t.Fatalf("unexpected final message: %q", meta.Iteration.FinalMessage)
	}
	if meta.Iteration.FinalChangeLog != "updated config and verified tests" {
		t.Fatalf("unexpected final change log: %q", meta.Iteration.FinalChangeLog)
	}
}
