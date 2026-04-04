package orchestration

import "testing"

func TestPrepareAgentTurnRequestAcceptsImageOnlyInput(t *testing.T) {
	prepared, code, err := prepareAgentTurnRequest(agentParams{
		Images: []sessionImageContent{{
			URL:      "data:image/png;base64,ZmFrZS1pbWFnZQ==",
			MimeType: "image/png",
		}},
	})
	if err != nil {
		t.Fatalf("prepareAgentTurnRequest returned error: %v", err)
	}
	if code != 200 {
		t.Fatalf("unexpected code: got %d want %d", code, 200)
	}
	if prepared.userInput.Text != "" || len(prepared.userInput.Content) != 1 || prepared.userInput.Content[0].Image == nil {
		t.Fatalf("unexpected prepared input: %+v", prepared.userInput)
	}
}

func TestPrepareAgentTurnRequestRejectsImageWithoutSource(t *testing.T) {
	_, _, err := prepareAgentTurnRequest(agentParams{
		Images: []sessionImageContent{{MimeType: "image/png"}},
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if err.Error() != "images[0] requires path or url" {
		t.Fatalf("unexpected error: %v", err)
	}
}
