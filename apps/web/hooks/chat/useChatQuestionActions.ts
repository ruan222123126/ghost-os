import { useCallback } from 'react';
import { createClientTraceId } from '@/lib/api/trace';
import { buildUserMessage } from '@/lib/chatMessages';
import { isAbortError, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { PendingQuestionMessage } from '@/lib/types';
import type { ChatStateControls } from './types';

interface UseChatQuestionActionsOptions {
  appendCommittedMessages: ChatStateControls['appendCommittedMessages'];
  appendErrorMessage: ChatStateControls['appendErrorMessage'];
  clearChatError: ChatStateControls['clearChatError'];
  clearStreamingState: ChatStateControls['clearStreamingState'];
  pendingQuestions: PendingQuestionMessage[];
  runHumanStream: (run: {
    answer: string;
    cancelled?: boolean;
    questionId: string;
    sessionId: string;
    signal?: AbortSignal;
    traceId: string;
  }) => Promise<void>;
  removePendingQuestion: ChatStateControls['removePendingQuestion'];
  setActiveRun: ChatStateControls['setActiveRun'];
  setChatError: ChatStateControls['setChatError'];
  setLoading: ChatStateControls['setLoading'];
  setStopPending: ChatStateControls['setStopPending'];
  stopPendingRef: ChatStateControls['stopPendingRef'];
}

export function useChatQuestionActions(options: UseChatQuestionActionsOptions) {
  const { copy } = useWebLocale();
  const {
    appendCommittedMessages,
    appendErrorMessage,
    clearChatError,
    clearStreamingState,
    pendingQuestions,
    runHumanStream,
    removePendingQuestion,
    setActiveRun,
    setChatError,
    setLoading,
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
      setChatError(copy.chat.answerCannotBeEmpty);
      return;
    }

    const pending = findPendingQuestion(pendingQuestions, trimmedQuestionId);
    if (!pending) {
      return;
    }

    clearChatError();
    clearStreamingState();
    setStopPending(false);
    setLoading(true);
    const traceId = createClientTraceId('human-response');
    const abortController = new AbortController();
    setActiveRun({ abortController, sessionId: pending.sessionId, traceId });
    try {
      removePendingQuestion(trimmedQuestionId);
      appendCommittedMessages([buildUserMessage(trimmedAnswer, {
        id: `local:answer:${traceId}:${trimmedQuestionId}`,
      })]);
      await runHumanStream({
        answer: trimmedAnswer,
        questionId: pending.questionId,
        sessionId: pending.sessionId,
        signal: abortController.signal,
        traceId,
      });
    } catch (error) {
      if (!shouldSuppressQuestionStreamError(error, stopPendingRef.current)) {
        appendErrorMessage(toErrorMessage(error, copy.system.genericRequestFailed));
      }
    } finally {
      setLoading(false);
      setActiveRun(null);
      setStopPending(false);
    }
  }, [
    copy.chat.answerCannotBeEmpty,
    copy.system.genericRequestFailed,
    appendCommittedMessages,
    appendErrorMessage,
    clearChatError,
    clearStreamingState,
    pendingQuestions,
    removePendingQuestion,
    runHumanStream,
    setActiveRun,
    setChatError,
    setLoading,
    setStopPending,
    stopPendingRef,
  ]);

  const cancelQuestion = useCallback(async (questionId: string) => {
    const trimmedQuestionId = questionId.trim();
    if (!trimmedQuestionId) {
      return;
    }

    const pending = findPendingQuestion(pendingQuestions, trimmedQuestionId);
    if (!pending) {
      return;
    }

    clearChatError();
    clearStreamingState();
    setStopPending(false);
    setLoading(true);
    const traceId = createClientTraceId('human-response');
    const abortController = new AbortController();
    setActiveRun({ abortController, sessionId: pending.sessionId, traceId });
    try {
      removePendingQuestion(trimmedQuestionId);
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
        appendErrorMessage(toErrorMessage(error, copy.system.genericRequestFailed));
      }
    } finally {
      setLoading(false);
      setActiveRun(null);
      setStopPending(false);
    }
  }, [
    copy.system.genericRequestFailed,
    appendErrorMessage,
    clearChatError,
    clearStreamingState,
    pendingQuestions,
    removePendingQuestion,
    runHumanStream,
    setActiveRun,
    setLoading,
    setStopPending,
    stopPendingRef,
  ]);

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

function findPendingQuestion(
  questions: PendingQuestionMessage[],
  questionId: string,
): PendingQuestionMessage | undefined {
  return questions.find((question) => question.questionId === questionId);
}
