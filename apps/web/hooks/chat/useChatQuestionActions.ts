import { useCallback } from 'react';
import { sendHumanResponse } from '@/lib/api/agent/api';
import {
  findPendingQuestion,
  removePendingQuestion,
  replacePendingQuestionWithUserAnswer,
} from '@/lib/chatMessages';
import { toErrorMessage } from '@/lib/errors';
import type { ChatStateControls } from './types';

interface UseChatQuestionActionsOptions {
  appendErrorMessage: ChatStateControls['appendErrorMessage'];
  clearChatError: ChatStateControls['clearChatError'];
  handleReply: (reply: Awaited<ReturnType<typeof sendHumanResponse>>) => Promise<void>;
  messages: ChatStateControls['messages'];
  setChatError: ChatStateControls['setChatError'];
  setLoading: ChatStateControls['setLoading'];
  setMessages: ChatStateControls['setMessages'];
}

export function useChatQuestionActions(options: UseChatQuestionActionsOptions) {
  const { appendErrorMessage, clearChatError, handleReply, messages, setChatError, setLoading, setMessages } = options;

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
    setLoading(true);
    try {
      const reply = await sendHumanResponse(pending.sessionId, pending.questionId, trimmedAnswer);
      setMessages((previous) => replacePendingQuestionWithUserAnswer(previous, trimmedQuestionId, trimmedAnswer));
      await handleReply(reply);
    } catch (error) {
      appendErrorMessage(toErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [appendErrorMessage, clearChatError, handleReply, messages, setChatError, setLoading, setMessages]);

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
    setLoading(true);
    try {
      const reply = await sendHumanResponse(pending.sessionId, pending.questionId, '', true);
      setMessages((previous) => removePendingQuestion(previous, trimmedQuestionId));
      await handleReply(reply);
    } catch (error) {
      appendErrorMessage(toErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [appendErrorMessage, clearChatError, handleReply, messages, setLoading, setMessages]);

  return {
    answerQuestion,
    cancelQuestion,
  };
}
