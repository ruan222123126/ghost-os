package sessions

import (
	"errors"
	"strings"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	"ghost-os/bridge/session"
)

var (
	ErrHumanQuestionIDRequired = errors.New("question_id is required")
	ErrHumanAnswerRequired     = errors.New("answer is required")
	ErrHumanQuestionNotFound   = errors.New("question not found in pending questions")
)

type ValidatedHumanResponse struct {
	SessionID  string
	QuestionID string
	Answer     string
	Cancelled  bool
}

func (s Service) AnswerHuman(params api.HumanResponseParams, traceID string) (api.HumanResponseAck, error) {
	store, err := s.requireSessionStore()
	if err != nil {
		return api.HumanResponseAck{}, err
	}

	validated, err := ValidateHumanResponseParams(params)
	if err != nil {
		return api.HumanResponseAck{}, err
	}

	s.log(traceID, bus.ActionHumanResponse, "running", nil)
	sess, err := store.Load(validated.SessionID)
	if err != nil {
		s.log(traceID, bus.ActionHumanResponse, "error", err)
		return api.HumanResponseAck{}, err
	}

	accepted, err := ApplyHumanResponse(sess, validated)
	if err != nil {
		s.log(traceID, bus.ActionHumanResponse, "error", err)
		return api.HumanResponseAck{}, err
	}

	if err := store.Save(sess); err != nil {
		s.log(traceID, bus.ActionHumanResponse, "error", err)
		return api.HumanResponseAck{}, err
	}

	s.log(traceID, bus.ActionHumanResponse, "success", nil)
	return api.HumanResponseAck{
		SessionID:  validated.SessionID,
		QuestionID: validated.QuestionID,
		Accepted:   accepted,
	}, nil
}

func ValidateHumanResponseParams(params api.HumanResponseParams) (ValidatedHumanResponse, error) {
	sessionID, err := RequireSessionID(params.SessionID)
	if err != nil {
		return ValidatedHumanResponse{}, err
	}
	questionID := strings.TrimSpace(params.QuestionID)
	if questionID == "" {
		return ValidatedHumanResponse{}, ErrHumanQuestionIDRequired
	}
	answer := strings.TrimSpace(params.Answer)
	if !params.Cancelled && answer == "" {
		return ValidatedHumanResponse{}, ErrHumanAnswerRequired
	}
	return ValidatedHumanResponse{
		SessionID:  sessionID,
		QuestionID: questionID,
		Answer:     answer,
		Cancelled:  params.Cancelled,
	}, nil
}

func ApplyHumanResponse(sess *session.Session, request ValidatedHumanResponse) (bool, error) {
	if request.Cancelled {
		if _, ok := sess.RemovePendingQuestion(request.QuestionID); !ok {
			return false, ErrHumanQuestionNotFound
		}
		sess.MarkEnded(sess.UpdatedAt)
		return false, nil
	}
	if !sess.SetHumanAnswer(request.QuestionID, request.Answer) {
		return false, ErrHumanQuestionNotFound
	}
	return true, nil
}
