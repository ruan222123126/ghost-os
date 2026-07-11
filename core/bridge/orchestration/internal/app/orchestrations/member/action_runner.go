package member

import (
	"context"
	"errors"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	appagentturn "ghost-os/bridge/orchestration/internal/app/agentturn"
	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
	"ghost-os/bridge/orchestration/internal/ports"
	"ghost-os/bridge/streaming"
)

type ActionTurnPreparer interface {
	PrepareMemberActionTurn(
		ctx context.Context,
		req ports.AgentActionRequest,
		input llm.Message,
	) (PreparedActionTurn, error)
}

type PreparedActionTurn interface {
	CurrentSessionID() string
	Run(ctx context.Context, input llm.Message, sink streaming.Sink) (string, string, error)
	Close()
}

type DirectActionRunner interface {
	RunDirectMemberAction(
		ctx context.Context,
		req ports.AgentActionRequest,
		sink streaming.Sink,
	) (string, string, error)
}

type ActionRunCardStarter interface {
	StartMemberActionCard(
		ctx context.Context,
		req ports.AgentActionRequest,
		sourceSessionID string,
	) (ActionRunCard, error)
}

type ActionRunCard interface {
	Sink() streaming.Sink
	Finish(ctx context.Context, input ActionRunCardFinishInput) error
}

type ActionRunCardFinishInput struct {
	Status          string
	Preview         string
	ErrorText       string
	SourceSessionID string
}

type ActionRunner struct {
	Preparer ActionTurnPreparer
	Direct   DirectActionRunner
	Cards    ActionRunCardStarter
}

func (r ActionRunner) Run(
	ctx context.Context,
	req ports.AgentActionRequest,
) (ports.AgentActionPayload, error) {
	input := llm.Message{Role: llm.RoleUser, Text: req.Message}
	if r.Direct != nil && req.RuntimeOverrides == nil {
		return r.runDirect(ctx, req)
	}
	if r.Preparer == nil {
		return ports.AgentActionPayload{}, errors.New("orchestration member turn preparer is not configured")
	}
	turn, err := r.Preparer.PrepareMemberActionTurn(ctx, req, input)
	if err != nil {
		return ports.AgentActionPayload{}, err
	}
	defer turn.Close()

	card, err := r.startCard(ctx, req, turn.CurrentSessionID())
	if err != nil {
		return ports.AgentActionPayload{}, err
	}
	response, sessionID, runErr := turn.Run(ctx, input, card.Sink())
	return r.finish(ctx, card, response, sessionID, runErr)
}

func (r ActionRunner) runDirect(
	ctx context.Context,
	req ports.AgentActionRequest,
) (ports.AgentActionPayload, error) {
	card, err := r.startCard(ctx, req, req.SessionID)
	if err != nil {
		return ports.AgentActionPayload{}, err
	}
	response, sessionID, runErr := r.Direct.RunDirectMemberAction(ctx, req, card.Sink())
	return r.finish(ctx, card, response, sessionID, runErr)
}

func (r ActionRunner) startCard(
	ctx context.Context,
	req ports.AgentActionRequest,
	sourceSessionID string,
) (ActionRunCard, error) {
	if r.Cards == nil {
		return nil, errors.New("orchestration task run card recorder is not configured")
	}
	return r.Cards.StartMemberActionCard(ctx, req, sourceSessionID)
}

func (r ActionRunner) finish(
	ctx context.Context,
	card ActionRunCard,
	response string,
	sessionID string,
	runErr error,
) (ports.AgentActionPayload, error) {
	result := apptasks.ExecutionResultFromStreamOutcome(ctx, response, sessionID, runErr)
	if err := card.Finish(ctx, ActionRunCardFinishInput{
		Status:          result.Status,
		Preview:         result.ResponsePreview,
		ErrorText:       result.Error,
		SourceSessionID: result.SessionIDOutput,
	}); err != nil {
		return ports.AgentActionPayload{}, err
	}
	return ActionPayloadFromOutcome(response, sessionID, runErr)
}

func ActionPayloadFromOutcome(
	response string,
	sessionID string,
	runErr error,
) (ports.AgentActionPayload, error) {
	if runErr == nil {
		return ports.AgentActionPayload{
			Kind:      ports.AgentActionPayloadSuccess,
			SessionID: sessionID,
			Message:   response,
		}, nil
	}
	var awaitingErr *agent.ErrAwaitingHuman
	if errors.As(runErr, &awaitingErr) {
		return ports.AgentActionPayload{
			Kind:      ports.AgentActionPayloadAwaiting,
			SessionID: sessionID,
			Prompt:    awaitingErr.Prompt,
		}, nil
	}
	return ports.AgentActionPayload{}, appagentturn.WrapErrorWithSessionID(runErr, sessionID)
}
