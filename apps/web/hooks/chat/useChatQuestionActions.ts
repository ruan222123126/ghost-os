import { useCallback } from 'react';
import { createClientTraceId } from '@/lib/api/trace';
import { buildUserMessage } from '@/lib/chatMessages';
import { isAbortError, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { PendingQuestionMessage } from '@/lib/types';
import type { ChatStateControls, ChatStreamRunResult } from './types';

interface UseChatQuestionActionsOptions {
  appendCommittedMessages: ChatStateControls['appendCommittedMessages'];
  appendErrorMessage: ChatStateControls['appendErrorMessage'];
  clearChatError: ChatStateControls['clearChatError'];
  clearStreamingState: ChatStateControls['clearStreamingState'];
  pendingQuestions: PendingQuestionMessage[];
  getCurrentSessionId: () => string;
  getStopPending: ChatStateControls['getStopPending'];
  hasPendingQuestionInSession: ChatStateControls['hasPendingQuestionInSession'];
  markBackgroundCompleted: ChatStateControls['markBackgroundCompleted'];
  runHumanStream: (run: {
    answer: string;
    cancelled?: boolean;
    questionId: string;
    sessionId: string;
    signal?: AbortSignal;
    traceId: string;
  }) => Promise<ChatStreamRunResult>;
  removePendingQuestion: ChatStateControls['removePendingQuestion'];
  resolveActiveRunSessionId: ChatStateControls['resolveActiveRunSessionId'];
  setActiveRun: ChatStateControls['setActiveRun'];
  setChatError: ChatStateControls['setChatError'];
  setLoading: ChatStateControls['setLoading'];
  setStopPending: ChatStateControls['setStopPending'];
}

export function useChatQuestionActions(options: UseChatQuestionActionsOptions) {
  const { copy } = useWebLocale();
  const {
    appendCommittedMessages,
    appendErrorMessage,
    clearChatError,
    clearStreamingState,
    pendingQuestions,
    getCurrentSessionId,
    getStopPending,
    hasPendingQuestionInSession,
    markBackgroundCompleted,
    runHumanStream,
    removePendingQuestion,
    resolveActiveRunSessionId,
    setActiveRun,
    setChatError,
    setLoading,
    setStopPending,
  } = options;

  const answerQuestion = useCallback(async (questionId: string, answer: string) => {
    const trimmedQuestionId = questionId.trim();
    const trimmedAnswer = answer.trim();
    if (!trimmedQuestionId) {
      return;
    }
    const pending = findPendingQuestion(pendingQuestions, trimmedQuestionId);
    if (!trimmedAnswer) {
      setChatError(pending?.sessionId ?? '', copy.chat.answerCannotBeEmpty);
      return;
    }

    if (!pending) {
      return;
    }

    clearChatError(pending.sessionId);
    clearStreamingState(pending.sessionId);
    setStopPending(pending.sessionId, false);
    setLoading(pending.sessionId, true);
    const traceId = createClientTraceId('human-response');
    const abortController = new AbortController();
    setActiveRun(pending.sessionId, { abortController, sessionId: pending.sessionId, traceId });
    try {
      removePendingQuestion(pending.sessionId, trimmedQuestionId);
      appendCommittedMessages(pending.sessionId, [buildUserMessage(trimmedAnswer, {
        id: `local:answer:${traceId}:${trimmedQuestionId}`,
      })]);
      const result = await runHumanStream({
        answer: trimmedAnswer,
        questionId: pending.questionId,
        sessionId: pending.sessionId,
        signal: abortController.signal,
        traceId,
      });
      markBackgroundCompletionIfNeeded({
        currentSessionId: getCurrentSessionId(),
        hasPendingQuestion: hasPendingQuestionInSession(result.sessionId),
        initialSessionId: pending.sessionId,
        markBackgroundCompleted,
        result,
      });
    } catch (error) {
      const targetSessionId = resolveActiveRunSessionId(traceId, pending.sessionId);
      if (!shouldSuppressQuestionStreamError(error, getStopPending(targetSessionId), abortController.signal.aborted)) {
        appendErrorMessage(targetSessionId, toErrorMessage(error, copy.system.genericRequestFailed));
      }
    } finally {
      const targetSessionId = resolveActiveRunSessionId(traceId, pending.sessionId);
      setLoading(targetSessionId, false);
      setActiveRun(targetSessionId, null);
      setStopPending(targetSessionId, false);
    }
  }, [
    copy.chat.answerCannotBeEmpty,
    copy.system.genericRequestFailed,
    appendCommittedMessages,
    appendErrorMessage,
    clearChatError,
    clearStreamingState,
    getCurrentSessionId,
    getStopPending,
    hasPendingQuestionInSession,
    markBackgroundCompleted,
    pendingQuestions,
    removePendingQuestion,
    resolveActiveRunSessionId,
    runHumanStream,
    setActiveRun,
    setChatError,
    setLoading,
    setStopPending,
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

    clearChatError(pending.sessionId);
    clearStreamingState(pending.sessionId);
    setStopPending(pending.sessionId, false);
    setLoading(pending.sessionId, true);
    const traceId = createClientTraceId('human-response');
    const abortController = new AbortController();
    setActiveRun(pending.sessionId, { abortController, sessionId: pending.sessionId, traceId });
    try {
      removePendingQuestion(pending.sessionId, trimmedQuestionId);
      const result = await runHumanStream({
        answer: '',
        cancelled: true,
        questionId: pending.questionId,
        sessionId: pending.sessionId,
        signal: abortController.signal,
        traceId,
      });
      markBackgroundCompletionIfNeeded({
        currentSessionId: getCurrentSessionId(),
        hasPendingQuestion: hasPendingQuestionInSession(result.sessionId),
        initialSessionId: pending.sessionId,
        markBackgroundCompleted,
        result,
      });
    } catch (error) {
      const targetSessionId = resolveActiveRunSessionId(traceId, pending.sessionId);
      if (!shouldSuppressQuestionStreamError(error, getStopPending(targetSessionId), abortController.signal.aborted)) {
        appendErrorMessage(targetSessionId, toErrorMessage(error, copy.system.genericRequestFailed));
      }
    } finally {
      const targetSessionId = resolveActiveRunSessionId(traceId, pending.sessionId);
      setLoading(targetSessionId, false);
      setActiveRun(targetSessionId, null);
      setStopPending(targetSessionId, false);
    }
  }, [
    copy.system.genericRequestFailed,
    appendErrorMessage,
    clearChatError,
    clearStreamingState,
    getCurrentSessionId,
    getStopPending,
    hasPendingQuestionInSession,
    markBackgroundCompleted,
    pendingQuestions,
    removePendingQuestion,
    resolveActiveRunSessionId,
    runHumanStream,
    setActiveRun,
    setLoading,
    setStopPending,
  ]);

  return {
    answerQuestion,
    cancelQuestion,
  };
}

function markBackgroundCompletionIfNeeded(input: {
  currentSessionId: string;
  hasPendingQuestion: boolean;
  initialSessionId: string;
  markBackgroundCompleted: ChatStateControls['markBackgroundCompleted'];
  result: ChatStreamRunResult;
}): void {
  const completedSessionId = input.result.sessionId.trim();
  if (
    !completedSessionId
    || input.result.terminalType !== 'done'
    || input.hasPendingQuestion
    || input.currentSessionId.trim() === completedSessionId
    || input.initialSessionId.trim() === ''
  ) {
    return;
  }
  input.markBackgroundCompleted(completedSessionId);
}

function shouldSuppressQuestionStreamError(error: unknown, stopPending: boolean, streamAborted: boolean): boolean {
  if (isAbortError(error)) {
    return true;
  }

  const message = toErrorMessage(error);
  if (!stopPending && !streamAborted) {
    return false;
  }

  return message === 'agent stream closed before terminal event'
    || message === 'agent run cancelled';
}

function findPendingQuestion(
  questions: PendingQuestionMessage[],
  questionId: string,
): PendingQuestionMessage | undefined {
  return questions.find((question) => question.questionId === questionId);
}
