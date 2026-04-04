package guiagent

import "testing"

func TestParseDecisionRejectsMissingCallUserPrompt(t *testing.T) {
	_, err := ParseDecision(`{"thought":"need help","action":{"type":"call_user"}}`)
	if err == nil {
		t.Fatal("expected parse error")
	}
	runErr, ok := err.(*RunError)
	if !ok || runErr.Code != ErrorModelOutputParse {
		t.Fatalf("unexpected error: %+v", err)
	}
}

func TestParseDecisionRejectsInvalidBox(t *testing.T) {
	_, err := ParseDecision(`{"thought":"click","action":{"type":"click","target":{"box":[0.9,0.9,0.1,1.0]}}}`)
	if err == nil {
		t.Fatal("expected parse error")
	}
	runErr, ok := err.(*RunError)
	if !ok || runErr.Code != ErrorInvalidTargetBox {
		t.Fatalf("unexpected error: %+v", err)
	}
}
