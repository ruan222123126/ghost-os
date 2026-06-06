package agentturn

import (
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

func (s Service) log(traceID string, action string, status string, err error) {
	if s.Logger != nil {
		s.Logger.Log(traceID, action, status, err)
	}
}

func (s Service) publishAssistant(traceID string, turn FinalizedTurn) {
	if s.Publisher != nil {
		s.Publisher.PublishAssistant(traceID, turn)
	}
}

func (s Service) publishAwaiting(traceID string, sessionID string, awaitingErr *agent.ErrAwaitingHuman) {
	if s.Publisher != nil {
		s.Publisher.PublishAwaitingHuman(traceID, sessionID, awaitingErr)
	}
}

func (s Service) classify(err error) (*agent.ErrAwaitingHuman, bus.ServiceErrorKind, bool, error) {
	if s.Classifier == nil {
		return nil, bus.ServiceErrorInternal, false, err
	}
	return s.Classifier.Classify(err)
}

func NewAwaitingHumanResponse(
	sessionID string,
	awaitingErr *agent.ErrAwaitingHuman,
) api.AskHumanAwaitingResponse {
	response := api.AskHumanAwaitingResponse{
		Status:        "awaiting_human",
		SessionID:     strings.TrimSpace(sessionID),
		QuestionID:    awaitingErr.QuestionID,
		Prompt:        awaitingErr.Prompt,
		SelectionMode: strings.TrimSpace(awaitingErr.SelectionMode),
	}
	if len(awaitingErr.Options) == 0 {
		return response
	}
	response.Options = make([]api.AskHumanOption, 0, len(awaitingErr.Options))
	for _, option := range awaitingErr.Options {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			continue
		}
		response.Options = append(response.Options, api.AskHumanOption{
			Label:       label,
			AllowCustom: option.AllowCustom,
		})
	}
	return response
}
