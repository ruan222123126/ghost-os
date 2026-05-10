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
	if prepared.UserInput.Text != "" || len(prepared.UserInput.Content) != 1 || prepared.UserInput.Content[0].Image == nil {
		t.Fatalf("unexpected prepared input: %+v", prepared.UserInput)
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
	if prepared.Mode != agentModePlan {
		t.Fatalf("unexpected mode: got %q want %q", prepared.Mode, agentModePlan)
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
