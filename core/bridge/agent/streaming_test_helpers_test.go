package agent

import (
	"testing"

	"ghost-os/bridge/streaming"
)

func mustAgentAssistantStepID(t *testing.T, turn int) string {
	t.Helper()

	stepID, err := streaming.AssistantStepID(turn)
	if err != nil {
		t.Fatalf("AssistantStepID returned error: %v", err)
	}
	return stepID
}

func mustAgentToolStepID(t *testing.T, turn int, toolIndex int) string {
	t.Helper()

	stepID, err := streaming.ToolStepID(turn, toolIndex)
	if err != nil {
		t.Fatalf("ToolStepID returned error: %v", err)
	}
	return stepID
}
