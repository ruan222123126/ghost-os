import { useCallback } from 'react';
import { createClientTraceId } from '@/lib/api/trace';
import { buildUserMessage } from '@/lib/chatMessages';
import { isAbortError, isAgentRunCancellationMessage, toErrorMessage } from '@/lib/errors';
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

interface QuestionResponseRunnerOptions extends UseChatQuestionActionsOptions {
  genericErrorMessage: string;
}

interface QuestionResponseRequest {
  answer: string;
  appendAnswerMessage: boolean;
  cancelled?: boolean;
  pending: PendingQuestionMessage;
  trimmedQuestionId: string;
}

interface AnswerQuestionOptions {
  answerCannotBeEmpty: string;
  pendingQuestions: PendingQuestionMessage[];
  runQuestionResponse: (request: QuestionResponseRequest) => Promise<void>;
  setChatError: ChatStateControls['setChatError'];
}

interface CancelQuestionOptions {
  pendingQuestions: PendingQuestionMessage[];
  runQuestionResponse: (request: QuestionResponseRequest) => Promise<void>;
}

interface QuestionResponseExecutionOptions extends Omit<QuestionResponseRunnerOptions, 'pendingQuestions' | 'setChatError'> {
  request: QuestionResponseRequest;
}

interface QuestionResponseErrorOptions {
  abortController: AbortController;
  error: unknown;
  options: QuestionResponseExecutionOptions;
  traceId: string;
}

export function useChatQuestionActions(options: UseChatQuestionActionsOptions) {
  const { copy } = useWebLocale();
  const runQuestionResponse = useQuestionResponseRunner({
    ...options,
    genericErrorMessage: copy.system.genericRequestFailed,
  });
  const answerQuestion = useAnswerQuestion({
    answerCannotBeEmpty: copy.chat.answerCannotBeEmpty,
    pendingQuestions: options.pendingQuestions,
    runQuestionResponse,
    setChatError: options.setChatError,
  });
  const cancelQuestion = useCancelQuestion({
    pendingQuestions: options.pendingQuestions,
    runQuestionResponse,
  });

  return {
    answerQuestion,
    cancelQuestion,
  };
}

function useAnswerQuestion(options: AnswerQuestionOptions) {
  const { answerCannotBeEmpty, pendingQuestions, runQuestionResponse, setChatError } = options;

  return useCallback(async (questionId: string, answer: string) => {
    const trimmedQuestionId = questionId.trim();
    const trimmedAnswer = answer.trim();
    if (!trimmedQuestionId) {
      return;
    }
    const pending = findPendingQuestion(pendingQuestions, trimmedQuestionId);
    if (!trimmedAnswer) {
      setChatError(pending?.sessionId ?? '', answerCannotBeEmpty);
      return;
    }
    if (!pending) {
      return;
    }

    await runQuestionResponse({
      answer: trimmedAnswer,
      appendAnswerMessage: true,
      pending,
      trimmedQuestionId,
    });
  }, [answerCannotBeEmpty, pendingQuestions, runQuestionResponse, setChatError]);
}

function useCancelQuestion(options: CancelQuestionOptions) {
  const { pendingQuestions, runQuestionResponse } = options;

  return useCallback(async (questionId: string) => {
    const trimmedQuestionId = questionId.trim();
    if (!trimmedQuestionId) {
      return;
    }

    const pending = findPendingQuestion(pendingQuestions, trimmedQuestionId);
    if (!pending) {
      return;
    }

    await runQuestionResponse({
      answer: '',
      appendAnswerMessage: false,
      cancelled: true,
      pending,
      trimmedQuestionId,
    });
  }, [pendingQuestions, runQuestionResponse]);
}

function useQuestionResponseRunner(options: QuestionResponseRunnerOptions) {
  return useCallback(async (request: QuestionResponseRequest) => {
    await executeQuestionResponse({
      ...options,
      request,
    });
  }, [options]);
}

async function executeQuestionResponse(options: QuestionResponseExecutionOptions): Promise<void> {
  const { request } = options;
  const { answer, appendAnswerMessage, cancelled, pending, trimmedQuestionId } = request;
  options.clearChatError(pending.sessionId);
  options.clearStreamingState(pending.sessionId);
  options.setStopPending(pending.sessionId, false);
  options.setLoading(pending.sessionId, true);
  const traceId = createClientTraceId('human-response');
  const abortController = new AbortController();
  options.setActiveRun(pending.sessionId, { abortController, sessionId: pending.sessionId, traceId });
  try {
    options.removePendingQuestion(pending.sessionId, trimmedQuestionId);
    appendAnswerMessageIfNeeded(options, traceId);
    const result = await options.runHumanStream({
      answer,
      cancelled,
      questionId: pending.questionId,
      sessionId: pending.sessionId,
      signal: abortController.signal,
      traceId,
    });
    markBackgroundCompletionIfNeeded({
      currentSessionId: options.getCurrentSessionId(),
      hasPendingQuestion: options.hasPendingQuestionInSession(result.sessionId),
      initialSessionId: pending.sessionId,
      markBackgroundCompleted: options.markBackgroundCompleted,
      result,
    });
  } catch (error) {
    handleQuestionResponseError({ abortController, error, options, traceId });
  } finally {
    finishQuestionResponse(options, traceId);
  }
}

function appendAnswerMessageIfNeeded(options: QuestionResponseExecutionOptions, traceId: string): void {
  const { request } = options;
  if (!request.appendAnswerMessage) {
    return;
  }

  options.appendCommittedMessages(request.pending.sessionId, [buildUserMessage(request.answer, {
    id: `local:answer:${traceId}:${request.trimmedQuestionId}`,
  })]);
}

function handleQuestionResponseError(input: QuestionResponseErrorOptions): void {
  const { abortController, error, options, traceId } = input;
  const targetSessionId = options.resolveActiveRunSessionId(traceId, options.request.pending.sessionId);
  if (!shouldSuppressQuestionStreamError(error, options.getStopPending(targetSessionId), abortController.signal.aborted)) {
    options.appendErrorMessage(targetSessionId, toErrorMessage(error, options.genericErrorMessage));
  }
}

function finishQuestionResponse(options: QuestionResponseExecutionOptions, traceId: string): void {
  const targetSessionId = options.resolveActiveRunSessionId(traceId, options.request.pending.sessionId);
  options.setLoading(targetSessionId, false);
  options.setActiveRun(targetSessionId, null);
  options.setStopPending(targetSessionId, false);
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

  return isAgentRunCancellationMessage(message);
}

function findPendingQuestion(
  questions: PendingQuestionMessage[],
  questionId: string,
): PendingQuestionMessage | undefined {
  return questions.find((question) => question.questionId === questionId);
}
