package orchestration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type persistingToolTurnEchoTool struct{}

type persistingGraphQLQueryTool struct{}

func (persistingToolTurnEchoTool) Name() string {
	return "echo"
}

func (persistingGraphQLQueryTool) Name() string {
	return "web_search"
}

func (persistingToolTurnEchoTool) Description() string {
	return "persisting echo"
}

func (persistingGraphQLQueryTool) Description() string {
	return "persisting graphql query"
}

func (persistingToolTurnEchoTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (persistingGraphQLQueryTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`)
}

func (persistingGraphQLQueryTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{ReadOnly: true}
}

func (persistingToolTurnEchoTool) Execute(
	ctx context.Context,
	_ json.RawMessage,
	_ string,
) (string, error) {
	sess := tools.SessionFromContext(ctx)
	if sess == nil {
		return "", context.Canceled
	}
	checkpoint := tools.SessionCheckpointFromContext(ctx)
	if checkpoint == nil {
		return "", context.Canceled
	}
	sess.EnsureDynamicToolLoaded("echo", "tool-turn-test")
	if err := checkpoint.Save(sess); err != nil {
		return "", err
	}
	return "tool-ok", nil
}

func (persistingGraphQLQueryTool) Execute(
	ctx context.Context,
	_ json.RawMessage,
	_ string,
) (string, error) {
	sess := tools.SessionFromContext(ctx)
	if sess == nil {
		return "", context.Canceled
	}
	checkpoint := tools.SessionCheckpointFromContext(ctx)
	if checkpoint == nil {
		return "", context.Canceled
	}
	sess.EnsureDynamicToolLoaded("web_search", "graphql-tool-test")
	if err := checkpoint.Save(sess); err != nil {
		return "", err
	}
	return `{"items":[{"title":"OpenAI"}]}`, nil
}

func TestSessionTurnStatePersistsCommittedGraphQLTextTurnOnLaterError(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	sess := session.NewSession("base system prompt")
	execCtx := tools.WithSession(context.Background(), sess)
	execCtx = tools.WithSessionCheckpoint(execCtx, sessionStore)

	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message: llm.Message{
				Role: llm.RoleAssistant,
				Text: `query { web_search(query: "OpenAI") }`,
			},
			FinishReason: llm.FinishStop,
		}},
	}
	registry := tools.NewRegistry()
	registry.Register(persistingGraphQLQueryTool{})
	runAgent := agent.NewAgentWithHistory(completer, registry, agent.NewHistory("base system prompt"), 3)
	runAgent.SetStrictToolCallProtocol(true)
	runAgent.AddAssistantTextHandler(agent.NewGraphQLTextTurnHandler(tools.NewGraphQLTextExecutor(registry)))

	turn := &sessionTurnState{
		sessionStore: sessionStore,
		persistence:  newSessionTurnCommitter(sessionStore, nil),
		sess:         sess,
		agent:        runAgent,
		execCtx:      execCtx,
		traceID:      "trace-graphql-text-transaction",
	}

	response, runErr := runAgent.RunWithTraceID(execCtx, "apply update", turn.traceID)
	if runErr == nil {
		t.Fatal("expected completion failure after graphql text execution")
	}

	_, persistedSessionID, err := turn.complete(response, runErr, nil)
	if err == nil {
		t.Fatal("expected turn completion to return the run error")
	}
	if persistedSessionID != sess.ID {
		t.Fatalf("unexpected persisted session id: got %q want %q", persistedSessionID, sess.ID)
	}

	loaded, loadErr := sessionStore.Load(sess.ID)
	if loadErr != nil {
		t.Fatalf("load session: %v", loadErr)
	}
	if len(loaded.Messages) != 5 {
		t.Fatalf("expected system + committed graphql turn messages, got %+v", loaded.Messages)
	}
	if loaded.Messages[1].Role != llm.RoleUser || loaded.Messages[1].Text != "apply update" {
		t.Fatalf("unexpected persisted user message: %+v", loaded.Messages[1])
	}
	if loaded.Messages[2].Role != llm.RoleAssistant || !strings.Contains(loaded.Messages[2].Text, "web_search") {
		t.Fatalf("unexpected persisted assistant graphql text: %+v", loaded.Messages[2])
	}
	if loaded.Messages[3].Role != llm.RoleTool {
		t.Fatalf("unexpected persisted graphql tool result: %+v", loaded.Messages[3])
	}
	if loaded.Messages[4].Role != llm.RoleInternal || !strings.Contains(loaded.Messages[4].Text, "[GRAPHQL_TOOL_RESULT]") {
		t.Fatalf("unexpected persisted graphql feedback: %+v", loaded.Messages[4])
	}
	loads := loaded.DynamicToolLoadsSnapshot()
	if len(loads) != 1 || loads[0].ToolName != "web_search" {
		t.Fatalf("expected persisted graphql tool side effect, got %+v", loads)
	}
}

func TestSessionTurnStatePersistsCommittedToolTurnOnLaterError(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	sess := session.NewSession("base system prompt")
	execCtx := tools.WithSession(context.Background(), sess)
	execCtx = tools.WithSessionCheckpoint(execCtx, sessionStore)

	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message: llm.Message{
				Role: llm.RoleAssistant,
				ToolCalls: []llm.ToolCall{{
					ID:        "call-echo-1",
					Name:      "echo",
					Arguments: []byte(`{"input":"hi"}`),
				}},
			},
			FinishReason: llm.FinishToolCalls,
		}},
	}
	registry := tools.NewRegistry()
	registry.Register(persistingToolTurnEchoTool{})
	runAgent := agent.NewAgentWithHistory(completer, registry, agent.NewHistory("base system prompt"), 3)

	turn := &sessionTurnState{
		sessionStore: sessionStore,
		persistence:  newSessionTurnCommitter(sessionStore, nil),
		sess:         sess,
		agent:        runAgent,
		execCtx:      execCtx,
		traceID:      "trace-tool-turn-transaction",
	}

	response, runErr := runAgent.RunWithTraceID(execCtx, "hello", turn.traceID)
	if runErr == nil {
		t.Fatal("expected completion failure after tool turn")
	}

	_, persistedSessionID, err := turn.complete(response, runErr, nil)
	if err == nil {
		t.Fatal("expected turn completion to return the run error")
	}
	if persistedSessionID != sess.ID {
		t.Fatalf("unexpected persisted session id: got %q want %q", persistedSessionID, sess.ID)
	}

	loaded, loadErr := sessionStore.Load(sess.ID)
	if loadErr != nil {
		t.Fatalf("load session: %v", loadErr)
	}
	if len(loaded.Messages) != 4 {
		t.Fatalf("expected system + committed tool turn messages, got %+v", loaded.Messages)
	}
	if loaded.Messages[1].Role != llm.RoleUser || loaded.Messages[1].Text != "hello" {
		t.Fatalf("unexpected persisted user message: %+v", loaded.Messages[1])
	}
	if loaded.Messages[2].Role != llm.RoleAssistant || len(loaded.Messages[2].ToolCalls) != 1 {
		t.Fatalf("unexpected persisted assistant tool call: %+v", loaded.Messages[2])
	}
	if loaded.Messages[3].Role != llm.RoleTool || loaded.Messages[3].ToolCallID != "call-echo-1" {
		t.Fatalf("unexpected persisted tool result: %+v", loaded.Messages[3])
	}
	if !strings.Contains(loaded.Messages[3].Text, `"status":"success"`) || !strings.Contains(loaded.Messages[3].Text, `"tool":"echo"`) {
		t.Fatalf("unexpected tool result payload: %q", loaded.Messages[3].Text)
	}

	loads := loaded.DynamicToolLoadsSnapshot()
	if len(loads) != 1 || loads[0].ToolName != "echo" {
		t.Fatalf("expected persisted tool-side effect, got %+v", loads)
	}
}
