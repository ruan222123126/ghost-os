package agentnode

import (
	"context"

	"ghost-os/bridge/llm"
	appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

type TurnPreparer interface {
	PrepareWorkflowAgentTurn(
		ctx context.Context,
		req appworkflows.AgentRequest,
		input llm.Message,
	) (PreparedTurn, error)
}

type PreparedTurn interface {
	CurrentSessionID() string
	Run(ctx context.Context, input llm.Message, sink streaming.Sink) (string, string, error)
	Close()
}

type DirectRunner interface {
	RunDirectWorkflowAgent(ctx context.Context, req appworkflows.AgentRequest, sink streaming.Sink) (string, string, error)
}

type OverrideRunner interface {
	RunWorkflowAgentWithOverrides(ctx context.Context, req appworkflows.AgentRequest, sink streaming.Sink) (string, string, error)
}

type RunCardStarter interface {
	StartWorkflowAgentCard(ctx context.Context, req appworkflows.AgentRequest, sourceSessionID string) (RunCard, error)
}

type RunCard interface {
	Sink() streaming.Sink
	Finish(ctx context.Context, result bridgeTasks.ExecutionResult) error
}

type ResultMapper interface {
	FromWorkflowAgentStream(
		ctx context.Context,
		response string,
		sessionID string,
		runErr error,
	) bridgeTasks.ExecutionResult
}

type Runner struct {
	Preparer                      TurnPreparer
	Direct                        DirectRunner
	Override                      OverrideRunner
	Cards                         RunCardStarter
	Results                       ResultMapper
	RuntimeOverridesRequireStream bool
}

func (r Runner) Execute(ctx context.Context, req appworkflows.AgentRequest) bridgeTasks.ExecutionResult {
	if r.Preparer == nil {
		return bridgeTasks.ExecutionResult{
			Status: bridgeTasks.RunStatusError,
			Error:  "workflow agent runner is not configured",
		}
	}
	if req.RuntimeOverrides != nil {
		if r.Override == nil && r.RuntimeOverridesRequireStream {
			return bridgeTasks.ExecutionResult{
				Status: bridgeTasks.RunStatusError,
				Error:  "workflow agent runtime overrides require a streaming runner",
			}
		}
		if r.Override != nil {
			return r.runOverride(ctx, req)
		}
	}
	if r.Direct != nil && req.RuntimeOverrides == nil {
		return r.runDirect(ctx, req)
	}
	return r.runPrepared(ctx, req)
}

func (r Runner) runPrepared(ctx context.Context, req appworkflows.AgentRequest) bridgeTasks.ExecutionResult {
	input := llm.Message{Role: llm.RoleUser, Text: req.Message}
	turn, err := r.Preparer.PrepareWorkflowAgentTurn(ctx, req, input)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: err.Error()}
	}
	defer turn.Close()

	card, err := r.startCard(ctx, req, turn.CurrentSessionID())
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: err.Error()}
	}
	response, sessionID, runErr := turn.Run(ctx, input, cardSink(card))
	return r.finish(ctx, card, response, sessionID, runErr)
}

func (r Runner) runDirect(ctx context.Context, req appworkflows.AgentRequest) bridgeTasks.ExecutionResult {
	card, err := r.startCard(ctx, req, "")
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: err.Error()}
	}
	response, sessionID, runErr := r.Direct.RunDirectWorkflowAgent(ctx, req, cardSink(card))
	return r.finish(ctx, card, response, sessionID, runErr)
}

func (r Runner) runOverride(ctx context.Context, req appworkflows.AgentRequest) bridgeTasks.ExecutionResult {
	card, err := r.startCard(ctx, req, "")
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: err.Error()}
	}
	response, sessionID, runErr := r.Override.RunWorkflowAgentWithOverrides(ctx, req, cardSink(card))
	return r.finish(ctx, card, response, sessionID, runErr)
}

func (r Runner) startCard(ctx context.Context, req appworkflows.AgentRequest, sourceSessionID string) (RunCard, error) {
	if r.Cards == nil {
		return nil, nil
	}
	return r.Cards.StartWorkflowAgentCard(ctx, req, sourceSessionID)
}

func (r Runner) finish(
	ctx context.Context,
	card RunCard,
	response string,
	sessionID string,
	runErr error,
) bridgeTasks.ExecutionResult {
	if r.Results == nil {
		return bridgeTasks.ExecutionResult{
			Status: bridgeTasks.RunStatusError,
			Error:  "workflow agent result mapper is not configured",
		}
	}
	result := r.Results.FromWorkflowAgentStream(ctx, response, sessionID, runErr)
	if card == nil {
		return result
	}
	if err := card.Finish(ctx, result); err != nil {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: err.Error()}
	}
	return result
}

func cardSink(card RunCard) streaming.Sink {
	if card == nil {
		return nil
	}
	return card.Sink()
}
