// Human-response use cases exposed to the transport layer.

package orchestration

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

const cancelledHumanDialogueMessage = "Conversation cancelled by user."

type validatedHumanResponse struct {
	sessionID  string
	questionID string
	answer     string
	cancelled  bool
}

// executeHumanResponseAction 接收 HUMAN_RESPONSE，将答案写回会话并解除 pending 状态。
func (s *bridgeService) executeHumanResponseAction(_ context.Context, params humanResponseParams, traceID string) (any, int, error) {
	store, code, err := s.requireSessionStore()
	if err != nil {
		return nil, code, err
	}

	validated, code, err := validateHumanResponseParams(params)
	if err != nil {
		return nil, code, err
	}

	logAction(traceID, busActionHumanResponse, "running", nil)
	sess, err := store.Load(validated.sessionID)
	if err != nil {
		statusCode := mapSessionStorageError(err)
		logAction(traceID, busActionHumanResponse, "error", err)
		return nil, statusCode, err
	}

	accepted, err := applyHumanResponse(sess, validated)
	if err != nil {
		logAction(traceID, busActionHumanResponse, "error", err)
		return nil, http.StatusNotFound, err
	}

	if err := store.Save(sess); err != nil {
		statusCode := mapSessionStorageError(err)
		logAction(traceID, busActionHumanResponse, "error", err)
		return nil, statusCode, err
	}

	logAction(traceID, busActionHumanResponse, "success", nil)
	return humanResponseAck{
		SessionID:  validated.sessionID,
		QuestionID: validated.questionID,
		Accepted:   accepted,
	}, http.StatusOK, nil
}

func validateHumanResponseParams(params humanResponseParams) (validatedHumanResponse, int, error) {
	sessionID, code, err := requireSessionID(params.SessionID)
	if err != nil {
		return validatedHumanResponse{}, code, err
	}
	questionID := strings.TrimSpace(params.QuestionID)
	if questionID == "" {
		return validatedHumanResponse{}, http.StatusBadRequest, errors.New("question_id is required")
	}
	answer := strings.TrimSpace(params.Answer)
	if !params.Cancelled && answer == "" {
		return validatedHumanResponse{}, http.StatusBadRequest, errors.New("answer is required")
	}
	return validatedHumanResponse{
		sessionID:  sessionID,
		questionID: questionID,
		answer:     answer,
		cancelled:  params.Cancelled,
	}, http.StatusOK, nil
}

func applyHumanResponse(sess *session.Session, request validatedHumanResponse) (bool, error) {
	if request.cancelled {
		if _, ok := sess.RemovePendingQuestion(request.questionID); !ok {
			return false, errors.New("question not found in pending questions")
		}
		sess.MarkEnded(sess.UpdatedAt)
		return false, nil
	}
	if !sess.SetHumanAnswer(request.questionID, request.answer) {
		return false, errors.New("question not found in pending questions")
	}
	return true, nil
}

// executeHumanAnswerAndResumeAction 先写入人类答案，再继续执行被 ask_human 暂停的回合。
func (s *bridgeService) executeHumanAnswerAndResumeAction(ctx context.Context, params humanResponseParams, traceID string) (any, int, error) {
	sessionID, code, err := requireSessionID(params.SessionID)
	if err != nil {
		return nil, code, err
	}

	if code, inflightErr := s.ensureSessionNotInflight(sessionID); inflightErr != nil {
		return nil, code, inflightErr
	}
	if code, activeErr := s.ensureSessionActive(sessionID); activeErr != nil {
		return nil, code, activeErr
	}

	params.SessionID = sessionID
	if _, code, err := s.executeHumanResponseAction(ctx, params, traceID); err != nil {
		return nil, code, err
	}
	if params.Cancelled {
		signal := &assistantSessionEndSignalPayload{
			Signal:  busAssistantSessionEndSignal,
			Message: cancelledHumanDialogueMessage,
		}
		payload, err := newAgentResponsePayload(cancelledHumanDialogueMessage, sessionID, signal, agentResponseMeta{})
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
		s.publishAssistantSessionPush(traceID, finalizedAgentTurn{
			message:    cancelledHumanDialogueMessage,
			sessionID:  sessionID,
			sessionEnd: signal,
		})
		return payload, http.StatusOK, nil
	}

	return s.sessionResumeRunner().Resume(ctx, sessionID, traceID)
}

// executeHumanAnswerAndResumeStreamAction 先写入人类答案，再以 SSE 方式续跑被 ask_human 暂停的回合。
func (s *bridgeService) executeHumanAnswerAndResumeStreamAction(ctx context.Context, params humanResponseParams, traceID string, sink streaming.Sink) (string, string, error) {
	sessionID, code, err := requireSessionID(params.SessionID)
	if err != nil {
		if emitErr := emitStreamErrorEvent(ctx, sink, traceID, 0, "", params.SessionID, code, err); emitErr != nil {
			return "", "", emitErr
		}
		return "", "", err
	}

	if code, inflightErr := s.ensureSessionNotInflight(sessionID); inflightErr != nil {
		if emitErr := emitStreamErrorEvent(ctx, sink, traceID, 0, "", sessionID, code, inflightErr); emitErr != nil {
			return "", "", emitErr
		}
		return "", sessionID, inflightErr
	}
	if code, activeErr := s.ensureSessionActive(sessionID); activeErr != nil {
		if emitErr := emitStreamErrorEvent(ctx, sink, traceID, 0, "", sessionID, code, activeErr); emitErr != nil {
			return "", "", emitErr
		}
		return "", sessionID, activeErr
	}

	params.SessionID = sessionID
	if _, code, err := s.executeHumanResponseAction(ctx, params, traceID); err != nil {
		if emitErr := emitStreamErrorEvent(ctx, sink, traceID, 0, "", sessionID, code, err); emitErr != nil {
			return "", "", emitErr
		}
		return "", sessionID, err
	}
	if params.Cancelled {
		signal := &assistantSessionEndSignalPayload{
			Signal:  busAssistantSessionEndSignal,
			Message: cancelledHumanDialogueMessage,
		}
		result := finalizedAgentTurn{
			message:    cancelledHumanDialogueMessage,
			sessionID:  sessionID,
			sessionEnd: signal,
		}
		if emitErr := emitDirectAgentStreamResult(ctx, sink, traceID, 0, result); emitErr != nil {
			return "", "", emitErr
		}
		s.publishAssistantSessionPush(traceID, result)
		return result.message, result.sessionID, nil
	}

	return s.sessionResumeRunner().ResumeStream(ctx, sessionID, traceID, sink)
}
