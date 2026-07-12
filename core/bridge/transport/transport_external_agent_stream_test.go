package transport

import (
	"context"
	"net/http"
	"testing"

	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/streaming"
)

func TestHandleExternalAgentStreamReturnsSSEEvents(t *testing.T) {
	fake := &externalAgentStreamUsecase{}
	handler := newExternalAgentStreamTestHandler(fake)

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/external-agent/stream",
		`{"provider":"codex","message":"hello","permission_mode":"safe-yolo","mode":"default"}`,
		map[string]string{"Content-Type": "application/json", "X-Trace-ID": "trace-external-stream"},
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("unexpected content type: got %q want %q", got, "text/event-stream")
	}
	if !fake.forceStart {
		t.Fatal("expected external stream to force start")
	}
	if fake.request.Provider != "codex" || fake.request.Message != "hello" {
		t.Fatalf("unexpected external request: %+v", fake.request)
	}
	if fake.request.PermissionMode != "safe-yolo" {
		t.Fatalf("unexpected permission mode: %+v", fake.request)
	}
	if fake.request.Mode != "default" {
		t.Fatalf("unexpected codex mode: %+v", fake.request)
	}

	events := decodeSSEEvents(t, recorder)
	if len(events) != 2 {
		t.Fatalf("unexpected event count: got %d want %d events=%+v", len(events), 2, events)
	}
	if events[0].Type != streaming.EventRunStarted || events[1].Type != streaming.EventDone {
		t.Fatalf("unexpected event types: %+v", events)
	}
}

func TestHandleExternalAgentStreamMethodNotAllowed(t *testing.T) {
	handler := newExternalAgentStreamTestHandler(&externalAgentStreamUsecase{})
	recorder := serveRequest(handler, http.MethodGet, "/api/external-agent/stream", "", nil)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}

func newExternalAgentStreamTestHandler(agent AgentUsecase) http.Handler {
	tr := &transport{
		usecases: transportUsecases{
			agent: agent,
		},
		maxBodyBytes: bridgeorchestration.DefaultMaxRequestBodyBytes,
	}
	mux := http.NewServeMux()
	mountTransportRoutes(mux, tr)
	return mux
}

type externalAgentStreamUsecase struct {
	request    bridgeorchestration.ExternalAgentRequest
	traceID    string
	forceStart bool
}

func (u *externalAgentStreamUsecase) Send(
	context.Context,
	bridgeorchestration.AgentParams,
	string,
) (bridgeorchestration.ServiceResult, error) {
	return bridgeorchestration.ServiceResult{}, nil
}

func (u *externalAgentStreamUsecase) Answer(
	context.Context,
	bridgeorchestration.HumanResponseParams,
	string,
) (bridgeorchestration.ServiceResult, error) {
	return bridgeorchestration.ServiceResult{}, nil
}

func (u *externalAgentStreamUsecase) AnswerStream(
	context.Context,
	bridgeorchestration.HumanResponseParams,
	string,
	bridgeorchestration.StreamSink,
) (string, string, error) {
	return "", "", nil
}

func (u *externalAgentStreamUsecase) PrepareStream(
	context.Context,
	bridgeorchestration.AgentParams,
	string,
) (bridgeorchestration.PreparedAgentStream, bridgeorchestration.ServiceResult, error) {
	return bridgeorchestration.PreparedAgentStream{}, bridgeorchestration.ServiceResult{}, nil
}

func (u *externalAgentStreamUsecase) PrepareExternalStream(
	params bridgeorchestration.ExternalAgentRequest,
	traceID string,
	forceStart bool,
) (bridgeorchestration.PreparedAgentStream, bridgeorchestration.ServiceResult, error) {
	u.request = params
	u.traceID = traceID
	u.forceStart = forceStart
	prepared := bridgeorchestration.NewPreparedAgentStream(
		func(ctx context.Context, sink bridgeorchestration.StreamSink) (string, string, error) {
			started, err := streaming.NewEvent(traceID, "session-external", 1, "", streaming.EventRunStarted, map[string]any{
				"session_id": "session-external",
				"provider":   "codex",
			})
			if err != nil {
				return "", "", err
			}
			if _, err := sink.Emit(ctx, started); err != nil {
				return "", "", err
			}
			done, err := streaming.NewEvent(traceID, "session-external", 1, "", streaming.EventDone, map[string]any{
				"session_id": "session-external",
			})
			if err != nil {
				return "", "", err
			}
			if _, err := sink.Emit(ctx, done); err != nil {
				return "", "", err
			}
			return "ok", "session-external", nil
		},
	)
	return prepared, bridgeorchestration.ServiceResult{}, nil
}
