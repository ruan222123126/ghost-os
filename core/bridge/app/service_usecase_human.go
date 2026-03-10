// Human-response use cases exposed to the transport layer.

package app

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"ghost-os/bridge/streaming"
)

const cancelledHumanDialogueMessage = "Conversation cancelled by user."

// executeHumanResponseAction 接收 HUMAN_RESPONSE，将答案写回会话并解除 pending 状态。
func (s *bridgeService) executeHumanResponseAction(_ context.Context, params humanResponseParams, traceID string) (any, int, error) {
	store, code, err := s.requireSessionStore()
	if err != nil {
		return nil, code, err
	}

	sessionID, code, err := requireSessionID(params.SessionID)
	if err != nil {
		return nil, code, err
	}

	questionID := strings.TrimSpace(params.QuestionID)
	if questionID == "" {
		return nil, http.StatusBadRequest, errors.New("question_id is required")
	}

	cancelled := params.Cancelled
	answer := strings.TrimSpace(params.Answer)
	if !cancelled && answer == "" {
		return nil, http.StatusBadRequest, errors.New("answer is required")
	}

	logAction(traceID, busActionHumanResponse, "running", nil)
	sess, err := store.Load(sessionID)
	if err != nil {
		statusCode := mapSessionStorageError(err)
		logAction(traceID, busActionHumanResponse, "error", err)
		return nil, statusCode, err
	}

	accepted := true
	if cancelled {
		if _, ok := sess.RemovePendingQuestion(questionID); !ok {
			err = errors.New("question not found in pending questions")
			logAction(traceID, busActionHumanResponse, "error", err)
			return nil, http.StatusNotFound, err
		}
		sess.MarkEnded(sess.UpdatedAt)
		accepted = false
	} else if !sess.SetHumanAnswer(questionID, answer) {
		err = errors.New("question not found in pending questions")
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
		SessionID:  sessionID,
		QuestionID: questionID,
		Accepted:   accepted,
	}, http.StatusOK, nil
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
		payload, err := newAgentResponsePayload(cancelledHumanDialogueMessage, sessionID, signal)
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

	return s.resumeAgentAction(ctx, sessionID, traceID)
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

	return s.resumeAgentStreamAction(ctx, sessionID, traceID, sink)
}

// resumeAgentAction 走 ask_human 专用续跑链路，不复用公开 AGENT_SEND 的空消息语义。
func (s *bridgeService) resumeAgentAction(ctx context.Context, sessionID string, traceID string) (any, int, error) {
	response, resumedSessionID, err := s.agentRunner.RunTurn(ctx, "", sessionID, traceID)
	if err != nil {
		awaitingErr, normalizedErr, statusCode, _ := classifyAgentTurnError(err)
		if awaitingErr != nil {
			s.publishAwaitingHumanSessionPush(traceID, resumedSessionID, awaitingErr)
			return newAwaitingHumanResponse(resumedSessionID, awaitingErr), http.StatusAccepted, nil
		}
		return nil, statusCode, normalizedErr
	}

	result, code, err := s.finalizeAgentTurn(response, resumedSessionID)
	if err != nil {
		return nil, code, err
	}

	payload, payloadErr := newAgentResponsePayload(result.message, result.sessionID, result.sessionEnd)
	if payloadErr != nil {
		return nil, http.StatusInternalServerError, payloadErr
	}
	s.publishAssistantSessionPush(traceID, result)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) resumeAgentStreamAction(ctx context.Context, sessionID string, traceID string, sink streaming.Sink) (string, string, error) {
	trackedSink := newEventTurnTracker(newSessionStreamBroadcastSink(sink, s.sessionPush))
	response, resumedSessionID, err := s.agentRunner.RunTurnStream(ctx, "", sessionID, traceID, trackedSink)
	if err != nil {
		awaitingErr, normalizedErr, _, cancelled := classifyAgentTurnError(err)
		if awaitingErr != nil {
			s.publishAwaitingHumanSessionPush(traceID, resumedSessionID, awaitingErr)
			return "", resumedSessionID, err
		}
		if cancelled {
			return "", resumedSessionID, normalizedErr
		}
		return "", resumedSessionID, normalizedErr
	}

	result, code, err := s.finalizeAgentTurn(response, resumedSessionID)
	if err != nil {
		stepID, stepErr := streaming.AssistantStepID(trackedSink.finalAssistantTurn())
		if stepErr != nil {
			return "", "", stepErr
		}
		if emitErr := emitStreamErrorEvent(ctx, trackedSink, traceID, trackedSink.finalAssistantTurn(), stepID, resumedSessionID, code, err); emitErr != nil {
			return "", "", emitErr
		}
		return "", "", err
	}

	s.publishAssistantSessionPush(traceID, result)
	return result.message, result.sessionID, nil
}
