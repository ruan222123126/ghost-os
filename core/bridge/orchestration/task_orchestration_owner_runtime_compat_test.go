package orchestration

import (
	"strings"
	"testing"

	"ghost-os/bridge/llm"
)

func TestOrchestrationOwnerRepairsEmptyResponseIntoDispatch(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := installOwnerDispatchRuntime(
		service,
		ownerAssistantResponse(""),
		dispatchToolResponse("dispatch-1", `{"action":"end_group"}`),
	)

	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 1))
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("expected repaired owner run to succeed, got %#v", run.Run)
	}
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	dispatchResults := groupOutput["dispatch_results"].([]any)
	if len(dispatchResults) != 1 || dispatchResults[0].(map[string]any)["action"] != "end_group" {
		t.Fatalf("expected repaired end_group dispatch, got %#v", dispatchResults)
	}
	if len(completer.requests) != 2 {
		t.Fatalf("expected owner repair round, got %#v", completer.requests)
	}
	repairRequest := completer.requests[1]
	if repairRequest.ToolChoice != "" {
		t.Fatalf("expected repair request to keep free tool choice, got %#v", repairRequest)
	}
	lastMessage := repairRequest.Messages[len(repairRequest.Messages)-1].Text
	if !strings.Contains(lastMessage, "(empty response)") || !strings.Contains(lastMessage, "did not advance the owner-led orchestration yet") {
		t.Fatalf("unexpected owner repair prompt: %q", lastMessage)
	}
}

func TestOrchestrationOwnerFailureRetainsOwnerSessionIDAfterRepairAttempt(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := installOwnerDispatchRuntime(
		service,
		ownerAssistantResponse(""),
		ownerAssistantResponse("我还是不调用工具。"),
	)

	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 1))
	if run.Run.Status != taskRunStatusError {
		t.Fatalf("expected owner run to fail after invalid repair, got %#v", run.Run)
	}
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	if strings.TrimSpace(groupOutput["owner_session_id"].(string)) == "" {
		t.Fatalf("expected failed owner run to retain owner_session_id, got %#v", groupOutput)
	}
	if len(completer.requests) != 2 || completer.requests[1].ToolChoice != "" {
		t.Fatalf("expected repair request before failure, got %#v", completer.requests)
	}
}

func ownerAssistantResponse(text string) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		Message: llm.Message{
			Role: llm.RoleAssistant,
			Text: text,
		},
		FinishReason: llm.FinishStop,
	}
}
