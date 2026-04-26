package agent

import (
	"fmt"
	"testing"

	"ghost-os/bridge/llm"
)

func TestNewAgentRunStateAppliesDefaultHistoryLimitForPreloadedHistory(t *testing.T) {
	messages := []llm.Message{{Role: llm.RoleSystem, Text: "system"}}
	totalUserMessages := DefaultMaxHistoryMessages + 20
	for i := 0; i < totalUserMessages; i++ {
		messages = append(messages, llm.Message{Role: llm.RoleUser, Text: fmt.Sprintf("m-%d", i)})
	}
	history := NewHistoryFromMessages(messages)
	agent := &Agent{
		completer: newFakeCompleter(newStopResponse("done")),
		tools:     newFakeToolCatalog(),
		history:   history,
		maxTurns:  1,
	}

	state, err := newAgentRunState(agent, nil, "trace-limit")
	if err != nil {
		t.Fatalf("newAgentRunState returned error: %v", err)
	}
	if state.history.maxMessages != DefaultMaxHistoryMessages {
		t.Fatalf("unexpected maxMessages: got %d want %d", state.history.maxMessages, DefaultMaxHistoryMessages)
	}
	if state.history.Len() != DefaultMaxHistoryMessages {
		t.Fatalf("unexpected history length: got %d want %d", state.history.Len(), DefaultMaxHistoryMessages)
	}

	got := state.history.Messages()
	if got[0].Role != llm.RoleSystem || got[0].Text != "system" {
		t.Fatalf("system message should be preserved, got %+v", got[0])
	}
	if got[1].Text != "m-21" {
		t.Fatalf("unexpected first retained user message: got %q want %q", got[1].Text, "m-21")
	}
	if got[len(got)-1].Text != "m-119" {
		t.Fatalf("unexpected last retained user message: got %q want %q", got[len(got)-1].Text, "m-119")
	}
}

func TestNewAgentRunStatePreservesCustomHistoryLimit(t *testing.T) {
	history := NewHistoryWithMaxMessages("system", 4)
	history.Append(llm.Message{Role: llm.RoleUser, Text: "u1"})
	history.Append(llm.Message{Role: llm.RoleAssistant, Text: "a1"})
	agent := &Agent{
		completer: newFakeCompleter(newStopResponse("done")),
		tools:     newFakeToolCatalog(),
		history:   history,
		maxTurns:  1,
	}

	state, err := newAgentRunState(agent, nil, "trace-custom")
	if err != nil {
		t.Fatalf("newAgentRunState returned error: %v", err)
	}
	if state.history.maxMessages != 4 {
		t.Fatalf("unexpected maxMessages: got %d want %d", state.history.maxMessages, 4)
	}

	for i := 0; i < 8; i++ {
		state.history.Append(llm.Message{Role: llm.RoleUser, Text: fmt.Sprintf("u-%d", i)})
	}
	if state.history.Len() != 4 {
		t.Fatalf("history limit not enforced after clone: got %d want %d", state.history.Len(), 4)
	}
	messages := state.history.Messages()
	if messages[0].Role != llm.RoleSystem {
		t.Fatalf("expected system message at index 0, got role=%q", messages[0].Role)
	}
}
