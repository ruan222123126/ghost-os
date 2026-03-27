import { useCallback } from 'react';
import { createClientTraceId } from '@/lib/api/trace';
import {
  findPendingQuestion,
  removePendingQuestion,
  replacePendingQuestionWithUserAnswer,
} from '@/lib/chatMessages';
import { isAbortError, toErrorMessage } from '@/lib/errors';
import type { ChatStateControls } from './types';

interface UseChatQuestionActionsOptions {
  appendErrorMessage: ChatStateControls['appendErrorMessage'];
  clearChatError: ChatStateControls['clearChatError'];
  runHumanStream: (run: {
    answer: string;
    cancelled?: boolean;
    questionId: string;
    sessionId: string;
    signal?: AbortSignal;
    traceId: string;
  }) => Promise<void>;
  messages: ChatStateControls['messages'];
  setActiveRun: ChatStateControls['setActiveRun'];
  setChatError: ChatStateControls['setChatError'];
  setLoading: ChatStateControls['setLoading'];
  setStopPending: ChatStateControls['setStopPending'];
  setMessages: ChatStateControls['setMessages'];
  stopPendingRef: ChatStateControls['stopPendingRef'];
}

export function useChatQuestionActions(options: UseChatQuestionActionsOptions) {
  const {
    appendErrorMessage,
    clearChatError,
    runHumanStream,
    messages,
    setActiveRun,
    setChatError,
    setLoading,
    setMessages,
    setStopPending,
    stopPendingRef,
  } = options;

  const answerQuestion = useCallback(async (questionId: string, answer: string) => {
    const trimmedQuestionId = questionId.trim();
    const trimmedAnswer = answer.trim();
    if (!trimmedQuestionId) {
      return;
    }
    if (!trimmedAnswer) {
      setChatError('Answer cannot be empty');
      return;
    }

    const pending = findPendingQuestion(messages, trimmedQuestionId);
    if (!pending) {
      return;
    }

    clearChatError();
    setStopPending(false);
    setLoading(true);
    const traceId = createClientTraceId('human-response');
    const abortController = new AbortController();
    setActiveRun({ abortController, sessionId: pending.sessionId, traceId });
    try {
      setMessages((previous) => replacePendingQuestionWithUserAnswer(previous, trimmedQuestionId, trimmedAnswer));
      await runHumanStream({
        answer: trimmedAnswer,
        questionId: pending.questionId,
        sessionId: pending.sessionId,
        signal: abortController.signal,
        traceId,
      });
    } catch (error) {
      if (!shouldSuppressQuestionStreamError(error, stopPendingRef.current)) {
        appendErrorMessage(toErrorMessage(error));
      }
    } finally {
      setLoading(false);
      setActiveRun(null);
      setStopPending(false);
    }
  }, [appendErrorMessage, clearChatError, messages, runHumanStream, setActiveRun, setChatError, setLoading, setMessages, setStopPending, stopPendingRef]);

  const cancelQuestion = useCallback(async (questionId: string) => {
    const trimmedQuestionId = questionId.trim();
    if (!trimmedQuestionId) {
      return;
    }

    const pending = findPendingQuestion(messages, trimmedQuestionId);
    if (!pending) {
      return;
    }

    clearChatError();
    setStopPending(false);
    setLoading(true);
    const traceId = createClientTraceId('human-response');
    const abortController = new AbortController();
    setActiveRun({ abortController, sessionId: pending.sessionId, traceId });
    try {
      setMessages((previous) => removePendingQuestion(previous, trimmedQuestionId));
      await runHumanStream({
        answer: '',
        cancelled: true,
        questionId: pending.questionId,
        sessionId: pending.sessionId,
        signal: abortController.signal,
        traceId,
      });
    } catch (error) {
      if (!shouldSuppressQuestionStreamError(error, stopPendingRef.current)) {
        appendErrorMessage(toErrorMessage(error));
      }
    } finally {
      setLoading(false);
      setActiveRun(null);
      setStopPending(false);
    }
  }, [appendErrorMessage, clearChatError, messages, runHumanStream, setActiveRun, setLoading, setMessages, setStopPending, stopPendingRef]);

  return {
    answerQuestion,
    cancelQuestion,
  };
}

function shouldSuppressQuestionStreamError(error: unknown, stopPending: boolean): boolean {
  if (!stopPending) {
    return false;
  }

  return isAbortError(error) || toErrorMessage(error) === 'agent stream closed before terminal event';
}
