package agentturn

import (
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

func (s Service) handleStandardError(
	traceID string,
	sessionID string,
	err error,
) (bus.ServiceResult, error) {
	awaitingErr, kind, _, normalizedErr := s.classify(err)
	if awaitingErr != nil {
		s.log(traceID, bus.ActionAgentSend, "awaiting_human", nil)
		s.publishAwaiting(traceID, sessionID, awaitingErr)
		return bus.ResultAccepted(newAwaitingHumanResponse(sessionID, awaitingErr)), nil
	}
	s.log(traceID, bus.ActionAgentSend, "error", normalizedErr)
	return bus.ServiceResult{}, bus.WrapError(kind, normalizedErr)
}

func (s Service) completeStandardTurn(
	traceID string,
	response string,
	sessionID string,
) (bus.ServiceResult, error) {
	result, err := s.Finalizer.Finalize(response, sessionID)
	if err != nil {
		s.log(traceID, bus.ActionAgentSend, "error", err)
		return bus.ServiceResult{}, err
	}
	payload, err := s.Finalizer.NewResponsePayload(result, ResponseMeta{})
	if err != nil {
		s.log(traceID, bus.ActionAgentSend, "error", err)
		return bus.ServiceResult{}, bus.WrapError(bus.ServiceErrorInternal, err)
	}
	s.publishAssistant(traceID, result)
	s.log(traceID, bus.ActionAgentSend, "success", nil)
	return bus.ResultSuccess(payload), nil
}

func (s Service) publishSpecial(traceID string, payload api.AgentResponse) {
	s.publishAssistant(traceID, FinalizedTurn{
		Message:    payload.Message,
		SessionID:  payload.SessionID,
		SessionEnd: payload.SessionEnd,
	})
	s.log(traceID, bus.ActionAgentSend, "success", nil)
}
