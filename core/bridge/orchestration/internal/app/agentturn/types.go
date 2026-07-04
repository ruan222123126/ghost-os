package agentturn

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	"ghost-os/bridge/orchestration/internal/domain/runtimeopts"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

const (
	ModeDefault = ""
)

type RequestRuntimeOptions = runtimeopts.RequestOptions

type PreparedRequest struct {
	UserInput        llm.Message
	Message          string
	SessionID        string
	RequestRuntime   *RequestRuntimeOptions
	RuntimeOverrides *bridgeTasks.TaskRuntimeOverrides
}

type FinalizedTurn struct {
	Message    string
	SessionID  string
	SessionEnd *api.AssistantSessionEndSignalPayload
}

type SessionGuards interface {
	EnsureSessionNotInflight(sessionID string) error
	EnsureSessionActive(sessionID string) error
}

type Runner interface {
	RunTurn(ctx context.Context, req PreparedRequest, traceID string) (string, string, error)
	RunTurnStream(ctx context.Context, req PreparedRequest, traceID string, sink streaming.Sink) (string, string, error)
}

type Finalizer interface {
	Finalize(response string, sessionID string) (FinalizedTurn, error)
	NewResponsePayload(turn FinalizedTurn) (api.AgentResponse, error)
}

type Publisher interface {
	PublishAssistant(traceID string, turn FinalizedTurn)
	PublishAwaitingHuman(traceID string, sessionID string, awaitingErr *agent.ErrAwaitingHuman)
}

type ErrorClassifier interface {
	Classify(err error) (*agent.ErrAwaitingHuman, bus.ServiceErrorKind, bool, error)
}

type Logger interface {
	Log(traceID string, action string, status string, err error)
}

type RunStopper interface {
	CancelAndWaitBySessionID(ctx context.Context, sessionID string) (StopHandle, error)
	CancelAndWaitByTraceID(ctx context.Context, traceID string) (StopHandle, error)
}

type StopHandle struct {
	SessionID string
}

type Service struct {
	Guards     SessionGuards
	Runner     Runner
	Finalizer  Finalizer
	Publisher  Publisher
	Classifier ErrorClassifier
	Logger     Logger
	Stopper    RunStopper
}

func NewResponsePayload(
	message string,
	sessionID string,
	sessionEnd *api.AssistantSessionEndSignalPayload,
) (api.AgentResponse, error) {
	payload := api.AgentResponse{
		Message:      strings.TrimSpace(message),
		SessionID:    strings.TrimSpace(sessionID),
		SessionEnded: sessionEnd != nil,
		SessionEnd:   sessionEnd,
	}
	if err := ValidateResponsePayload(payload); err != nil {
		return api.AgentResponse{}, err
	}
	return payload, nil
}

func ValidateResponsePayload(payload api.AgentResponse) error {
	if err := validateResponseRequiredFields(payload); err != nil {
		return err
	}
	if err := validateResponseSessionEnd(payload); err != nil {
		return err
	}
	return validateResponseMode(payload)
}

func validateResponseRequiredFields(payload api.AgentResponse) error {
	if strings.TrimSpace(payload.Message) == "" {
		return errors.New("agent response message is empty")
	}
	if strings.TrimSpace(payload.SessionID) == "" {
		return errors.New("agent response session_id is empty")
	}
	return nil
}

func validateResponseSessionEnd(payload api.AgentResponse) error {
	if payload.SessionEnded && payload.SessionEnd == nil {
		return errors.New("agent response session_end is required when session_ended=true")
	}
	if !payload.SessionEnded && payload.SessionEnd != nil {
		return errors.New("agent response session_end must be empty when session_ended=false")
	}
	if payload.SessionEnd == nil {
		return nil
	}
	if strings.TrimSpace(payload.SessionEnd.Signal) != bus.AssistantSessionEndSignal {
		return fmt.Errorf("agent response session_end.signal must be %q", bus.AssistantSessionEndSignal)
	}
	if strings.TrimSpace(payload.SessionEnd.Message) == "" {
		return errors.New("agent response session_end.message is empty")
	}
	if strings.TrimSpace(payload.SessionEnd.Message) != strings.TrimSpace(payload.Message) {
		return errors.New("agent response session_end.message must equal message")
	}
	return nil
}

func validateResponseMode(payload api.AgentResponse) error {
	if strings.TrimSpace(payload.Mode) != "" {
		return errors.New("agent response mode is not supported")
	}
	return nil
}
