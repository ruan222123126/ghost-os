// Human-response use cases exposed to the transport layer.

package orchestration

import (
	"context"
	"errors"
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
func (s *bridgeService) executeHumanResponseAction(_ context.Context, params humanResponseParams, traceID string) (ServiceResult, error) {
	store, err := s.requireSessionStore()
	if err != nil {
		return ServiceResult{}, err
	}

	validated, err := validateHumanResponseParams(params)
	if err != nil {
		return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}

	logAction(traceID, busActionHumanResponse, "running", nil)
	sess, err := store.Load(validated.sessionID)
	if err != nil {
		logAction(traceID, busActionHumanResponse, "error", err)
		return ServiceResult{}, wrapServiceError(mapSessionStorageErrorKind(err), err)
	}

	accepted, err := applyHumanResponse(sess, validated)
	if err != nil {
		logAction(traceID, busActionHumanResponse, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorNotFound, err)
	}

	if err := store.Save(sess); err != nil {
		logAction(traceID, busActionHumanResponse, "error", err)
		return ServiceResult{}, wrapServiceError(mapSessionStorageErrorKind(err), err)
	}

	logAction(traceID, busActionHumanResponse, "success", nil)
	return serviceResultSuccess(humanResponseAck{
		SessionID:  validated.sessionID,
		QuestionID: validated.questionID,
		Accepted:   accepted,
	}), nil
}

func validateHumanResponseParams(params humanResponseParams) (validatedHumanResponse, error) {
	sessionID, err := requireSessionID(params.SessionID)
	if err != nil {
		return validatedHumanResponse{}, err
	}
	questionID := strings.TrimSpace(params.QuestionID)
	if questionID == "" {
		return validatedHumanResponse{}, errors.New("question_id is required")
	}
	answer := strings.TrimSpace(params.Answer)
	if !params.Cancelled && answer == "" {
		return validatedHumanResponse{}, errors.New("answer is required")
	}
	return validatedHumanResponse{
		sessionID:  sessionID,
		questionID: questionID,
		answer:     answer,
		cancelled:  params.Cancelled,
	}, nil
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
func (s *bridgeService) executeHumanAnswerAndResumeAction(ctx context.Context, params humanResponseParams, traceID string) (ServiceResult, error) {
	sessionID, err := requireSessionID(params.SessionID)
	if err != nil {
		return ServiceResult{}, err
	}

	if inflightErr := s.ensureSessionNotInflight(sessionID); inflightErr != nil {
		return ServiceResult{}, inflightErr
	}
	if activeErr := s.ensureSessionActive(sessionID); activeErr != nil {
		return ServiceResult{}, activeErr
	}

	params.SessionID = sessionID
	if _, err := s.executeHumanResponseAction(ctx, params, traceID); err != nil {
		return ServiceResult{}, err
	}
	if params.Cancelled {
		result := cancelledHumanTurn(sessionID)
		payload, err := newAgentResponsePayload(result.message, result.sessionID, result.sessionEnd, agentResponseMeta{})
		if err != nil {
			return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
		}
		s.publishAssistantSessionPush(traceID, result)
		return serviceResultSuccess(payload), nil
	}

	return s.sessionResumeRunner().Resume(ctx, sessionID, traceID)
}

// executeHumanAnswerAndResumeStreamAction 先写入人类答案，再以 SSE 方式续跑被 ask_human 暂停的回合。
func (s *bridgeService) executeHumanAnswerAndResumeStreamAction(ctx context.Context, params humanResponseParams, traceID string, sink streaming.Sink) (string, string, error) {
	sessionID, err := s.resolveHumanResumeStreamSession(ctx, params.SessionID, traceID, sink)
	if err != nil {
		return "", sessionID, err
	}

	params.SessionID = sessionID
	if err := s.applyHumanResponseForResumeStream(ctx, params, traceID, sink); err != nil {
		return "", sessionID, err
	}
	if params.Cancelled {
		result := cancelledHumanTurn(sessionID)
		if emitErr := emitDirectAgentStreamResult(ctx, sink, traceID, 0, result); emitErr != nil {
			return "", "", emitErr
		}
		s.publishAssistantSessionPush(traceID, result)
		return result.message, result.sessionID, nil
	}

	return s.sessionResumeRunner().ResumeStream(ctx, sessionID, traceID, sink)
}

func cancelledHumanTurn(sessionID string) finalizedAgentTurn {
	return finalizedAgentTurn{
		message:   cancelledHumanDialogueMessage,
		sessionID: sessionID,
		sessionEnd: &assistantSessionEndSignalPayload{
			Signal:  busAssistantSessionEndSignal,
			Message: cancelledHumanDialogueMessage,
		},
	}
}

func emitHumanResumeStreamError(
	ctx context.Context,
	sink streaming.Sink,
	traceID string,
	sessionID string,
	statusCode int,
	cause error,
) error {
	if emitErr := emitStreamErrorEvent(ctx, sink, traceID, 0, "", sessionID, statusCode, cause); emitErr != nil {
		return emitErr
	}
	return cause
}

func (s *bridgeService) resolveHumanResumeStreamSession(
	ctx context.Context,
	rawSessionID string,
	traceID string,
	sink streaming.Sink,
) (string, error) {
	sessionID, err := requireSessionID(rawSessionID)
	if err != nil {
		return "", emitHumanResumeStreamError(
			ctx,
			sink,
			traceID,
			rawSessionID,
			legacyStatusFromServiceError(err),
			err,
		)
	}
	if inflightErr := s.ensureSessionNotInflight(sessionID); inflightErr != nil {
		return sessionID, emitHumanResumeStreamError(
			ctx,
			sink,
			traceID,
			sessionID,
			legacyStatusFromServiceError(inflightErr),
			inflightErr,
		)
	}
	if activeErr := s.ensureSessionActive(sessionID); activeErr != nil {
		return sessionID, emitHumanResumeStreamError(
			ctx,
			sink,
			traceID,
			sessionID,
			legacyStatusFromServiceError(activeErr),
			activeErr,
		)
	}
	return sessionID, nil
}

func (s *bridgeService) applyHumanResponseForResumeStream(
	ctx context.Context,
	params humanResponseParams,
	traceID string,
	sink streaming.Sink,
) error {
	_, err := s.executeHumanResponseAction(ctx, params, traceID)
	if err == nil {
		return nil
	}
	return emitHumanResumeStreamError(
		ctx,
		sink,
		traceID,
		params.SessionID,
		legacyStatusFromServiceError(err),
		err,
	)
}
