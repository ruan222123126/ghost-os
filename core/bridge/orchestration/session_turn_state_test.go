package orchestration

import (
	"context"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type persistingGraphQLTextExecutor struct{}

func (persistingGraphQLTextExecutor) Execute(
	ctx context.Context,
	text string,
	traceID string,
) (tools.GraphQLTextExecutionResult, error) {
	sess := tools.SessionFromContext(ctx)
	if sess == nil {
		return tools.GraphQLTextExecutionResult{}, context.Canceled
	}
	checkpoint := tools.SessionCheckpointFromContext(ctx)
	if checkpoint == nil {
		return tools.GraphQLTextExecutionResult{}, context.Canceled
	}
	now := time.Now().UTC()
	sess.StorePendingGraphQLMutationIntent(session.PendingGraphQLMutationIntent{
		IntentID:      "intent-graphql-text-auto",
		Source:        "crm",
		Domain:        "people",
		PolicyName:    "update_viewer",
		RootMutation:  "updateViewer",
		Query:         strings.TrimSpace(text),
		ToolCallID:    "graphql-text-call-1",
		TraceID:       strings.TrimSpace(traceID),
		PreparedAt:    now,
		ApprovedAt:    now,
		ExecutedAt:    now,
		Status:        session.GraphQLMutationIntentExecuted,
		CommitState:   session.GraphQLMutationCommitStateExecuted,
		DeliveryKey:   "delivery-key",
		RequestHash:   "request-hash",
		Summary:       "update viewer",
		AttemptCount:  1,
		ResponseBytes: len(`{"status":"executed"}`),
	})
	if err := checkpoint.Save(sess); err != nil {
		return tools.GraphQLTextExecutionResult{}, err
	}
	return tools.GraphQLTextExecutionResult{
		Recognized: true,
		Output:     `{"status":"executed","intent_id":"intent-graphql-text-auto"}`,
	}, nil
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
				Text: `mutation { updateViewer(input: {id: "user-1"}) { ok } }`,
			},
			FinishReason: llm.FinishStop,
		}},
	}
	runAgent := agent.NewAgentWithHistory(completer, tools.NewRegistry(), agent.NewHistory("base system prompt"), 3)
	runAgent.SetStrictToolCallProtocol(true)
	runAgent.SetGraphQLTextExecutor(persistingGraphQLTextExecutor{})

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
	if len(loaded.Messages) != 4 {
		t.Fatalf("expected system + committed graphql turn messages, got %+v", loaded.Messages)
	}
	if loaded.Messages[1].Role != llm.RoleUser || loaded.Messages[1].Text != "apply update" {
		t.Fatalf("unexpected persisted user message: %+v", loaded.Messages[1])
	}
	if loaded.Messages[2].Role != llm.RoleAssistant || !strings.Contains(loaded.Messages[2].Text, "updateViewer") {
		t.Fatalf("unexpected persisted assistant graphql text: %+v", loaded.Messages[2])
	}
	if loaded.Messages[3].Role != llm.RoleUser || !strings.Contains(loaded.Messages[3].Text, "[GRAPHQL_EXECUTION_RESULT]") {
		t.Fatalf("unexpected persisted graphql feedback: %+v", loaded.Messages[3])
	}

	intents := loaded.PendingGraphQLMutationIntentsSnapshot()
	if len(intents) != 1 {
		t.Fatalf("expected persisted graphql mutation intent, got %+v", intents)
	}
	if intents[0].Status != session.GraphQLMutationIntentExecuted {
		t.Fatalf("expected executed graphql mutation intent, got %+v", intents[0])
	}
}
