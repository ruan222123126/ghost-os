package orchestration

import "testing"

func TestPrepareAgentTurnRequestAcceptsImageOnlyInput(t *testing.T) {
	prepared, err := prepareAgentTurnRequest(agentParams{
		Images: []sessionImageContent{{
			URL:      "data:image/png;base64,ZmFrZS1pbWFnZQ==",
			MimeType: "image/png",
		}},
	})
	if err != nil {
		t.Fatalf("prepareAgentTurnRequest returned error: %v", err)
	}
	if prepared.userInput.Text != "" || len(prepared.userInput.Content) != 1 || prepared.userInput.Content[0].Image == nil {
		t.Fatalf("unexpected prepared input: %+v", prepared.userInput)
	}
}

func TestPrepareAgentTurnRequestRejectsImageWithoutSource(t *testing.T) {
	_, err := prepareAgentTurnRequest(agentParams{
		Images: []sessionImageContent{{MimeType: "image/png"}},
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if err.Error() != "images[0] requires path or url" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareAgentTurnRequestNormalizesPlanMode(t *testing.T) {
	prepared, err := prepareAgentTurnRequest(agentParams{
		Mode:    " PLAN ",
		Message: "plan this task",
	})
	if err != nil {
		t.Fatalf("prepareAgentTurnRequest returned error: %v", err)
	}
	if prepared.mode != agentModePlan {
		t.Fatalf("unexpected mode: got %q want %q", prepared.mode, agentModePlan)
	}
}

func TestPrepareAgentTurnRequestRejectsUnknownMode(t *testing.T) {
	_, err := prepareAgentTurnRequest(agentParams{
		Mode:    "execute",
		Message: "hello",
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if err.Error() != `unsupported agent mode: "execute"` {
		t.Fatalf("unexpected error: %v", err)
	}
}
