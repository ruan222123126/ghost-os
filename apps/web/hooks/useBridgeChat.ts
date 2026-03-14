import { useCallback, useRef, useState } from 'react';
import { getSession } from '@/lib/api/sessions/api';
import { sendHumanResponse, sendMessage, stopAgent } from '@/lib/api/agent/api';
import { createClientTraceId } from '@/lib/api/trace';
import {
  buildErrorMessage,
  buildUserMessage,
  findPendingQuestion,
  hasPendingQuestion,
  mapAgentReplyToChatMessages,
  mapSessionMessagesToChat,
  removePendingQuestion,
  replacePendingQuestionWithUserAnswer,
} from '@/lib/chatMessages';
import { toErrorMessage } from '@/lib/errors';
import type { AgentSendResponse, ChatMessage } from '@/lib/types';

const AGENT_RUN_CANCELLED_MESSAGE = 'agent run cancelled';

interface ActiveAgentRun {
  sessionId: string;
  traceId: string;
}

interface UseBridgeChatResult {
  messages: ChatMessage[];
  loading: boolean;
  historyLoading: boolean;
  chatError: string;
  hasPendingQuestion: boolean;
  canStop: boolean;
  sendChatMessage: (message: string) => Promise<void>;
  stopCurrentRun: () => Promise<void>;
  answerQuestion: (questionId: string, answer: string) => Promise<void>;
  cancelQuestion: (questionId: string) => Promise<void>;
  loadSessionHistory: (sessionId: string) => Promise<void>;
  clearMessages: () => void;
}

interface UseBridgeChatOptions {
  currentSessionId: string;
  onSessionResolved?: (sessionId: string) => void;
}

export function useBridgeChat(options: UseBridgeChatOptions): UseBridgeChatResult {
  const { currentSessionId, onSessionResolved } = options;
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [loading, setLoading] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [chatError, setChatError] = useState('');
  const [activeRun, setActiveRunState] = useState<ActiveAgentRun | null>(null);
  const [stopPending, setStopPendingState] = useState(false);
  const activeRunRef = useRef<ActiveAgentRun | null>(null);
  const stopPendingRef = useRef(false);

  const setActiveRun = useCallback((nextRun: ActiveAgentRun | null) => {
    activeRunRef.current = nextRun;
    setActiveRunState(nextRun);
  }, []);

  const setStopPending = useCallback((value: boolean) => {
    stopPendingRef.current = value;
    setStopPendingState(value);
  }, []);

  const appendMessages = useCallback((nextMessages: ChatMessage[]) => {
    if (nextMessages.length === 0) {
      return;
    }
    setMessages((previous) => [...previous, ...nextMessages]);
  }, []);

  const replaceWithErrorMessage = useCallback((messageText: string) => {
    setMessages([buildErrorMessage(messageText)]);
  }, []);

  const appendErrorMessage = useCallback((messageText: string) => {
    setChatError(messageText);
    appendMessages([buildErrorMessage(messageText)]);
  }, [appendMessages]);

  const hydrateSessionHistory = useCallback(async (sessionId: string) => {
    const detail = await getSession(sessionId);
    setMessages(mapSessionMessagesToChat(detail.messages));
  }, []);

  const handleReply = useCallback(async (reply: AgentSendResponse) => {
    if (reply.session_id) {
      const currentRun = activeRunRef.current;
      if (currentRun && currentRun.sessionId !== reply.session_id) {
        setActiveRun({ ...currentRun, sessionId: reply.session_id });
      }
      if (reply.session_id !== currentSessionId) {
        onSessionResolved?.(reply.session_id);
      }
      await hydrateSessionHistory(reply.session_id);
      return;
    }
    appendMessages(mapAgentReplyToChatMessages(reply));
  }, [appendMessages, currentSessionId, hydrateSessionHistory, onSessionResolved, setActiveRun]);

  const appendErrorFromUnknown = useCallback((error: unknown) => {
    appendErrorMessage(toErrorMessage(error));
  }, [appendErrorMessage]);

  const sendChatMessage = useCallback(async (message: string) => {
    const trimmed = message.trim();
    const normalizedSessionId = currentSessionId.trim();
    if (!trimmed && !normalizedSessionId) {
      return;
    }

    const traceId = createClientTraceId('agent-run');

    setChatError('');
    setStopPending(false);
    setActiveRun({ sessionId: normalizedSessionId, traceId });
    if (trimmed) {
      appendMessages([buildUserMessage(trimmed)]);
    }
    setLoading(true);

    try {
      const reply = await sendMessage(trimmed, normalizedSessionId || undefined, traceId);
      await handleReply(reply);
    } catch (error) {
      const messageText = toErrorMessage(error);
      if (!(stopPendingRef.current && messageText === AGENT_RUN_CANCELLED_MESSAGE)) {
        appendErrorMessage(messageText);
      }
    } finally {
      setLoading(false);
      setActiveRun(null);
      setStopPending(false);
    }
  }, [appendErrorMessage, appendMessages, currentSessionId, handleReply, setActiveRun, setStopPending]);

  const stopCurrentRun = useCallback(async () => {
    const run = activeRunRef.current;
    if (!run || stopPendingRef.current) {
      return;
    }

    setChatError('');
    setStopPending(true);
    try {
      await stopAgent(run.sessionId || undefined, run.traceId || undefined);
    } catch (error) {
      setChatError(toErrorMessage(error));
      setStopPending(false);
    }
  }, [setStopPending]);

  const answerQuestion = useCallback(async (questionId: string, answer: string) => {
    const trimmedQuestionID = questionId.trim();
    const trimmedAnswer = answer.trim();
    if (!trimmedQuestionID) {
      return;
    }
    if (!trimmedAnswer) {
      setChatError('Answer cannot be empty');
      return;
    }

    const pending = findPendingQuestion(messages, trimmedQuestionID);
    if (!pending) {
      return;
    }

    setChatError('');
    setLoading(true);
    try {
      const reply = await sendHumanResponse(pending.sessionId, pending.questionId, trimmedAnswer);
      setMessages((previous) => replacePendingQuestionWithUserAnswer(previous, trimmedQuestionID, trimmedAnswer));
      await handleReply(reply);
    } catch (error) {
      appendErrorFromUnknown(error);
    } finally {
      setLoading(false);
    }
  }, [appendErrorFromUnknown, handleReply, messages]);

  const cancelQuestion = useCallback(async (questionId: string) => {
    const trimmedQuestionID = questionId.trim();
    if (!trimmedQuestionID) {
      return;
    }

    const pending = findPendingQuestion(messages, trimmedQuestionID);
    if (!pending) {
      return;
    }

    setChatError('');
    setLoading(true);
    try {
      const reply = await sendHumanResponse(pending.sessionId, pending.questionId, '', true);
      setMessages((previous) => removePendingQuestion(previous, trimmedQuestionID));
      await handleReply(reply);
    } catch (error) {
      appendErrorFromUnknown(error);
    } finally {
      setLoading(false);
    }
  }, [appendErrorFromUnknown, handleReply, messages]);

  const loadSessionHistory = useCallback(async (sessionId: string) => {
    const id = sessionId.trim();
    if (!id) {
      setMessages([]);
      return;
    }

    setChatError('');
    setHistoryLoading(true);
    try {
      const detail = await getSession(id);
      setMessages(mapSessionMessagesToChat(detail.messages));
    } catch (error) {
      const messageText = toErrorMessage(error);
      setChatError(messageText);
      replaceWithErrorMessage(messageText);
    } finally {
      setHistoryLoading(false);
    }
  }, [replaceWithErrorMessage]);

  const clearMessages = useCallback(() => {
    setChatError('');
    setMessages([]);
  }, []);

  return {
    messages,
    loading,
    historyLoading,
    chatError,
    hasPendingQuestion: hasPendingQuestion(messages),
    canStop: loading && activeRun !== null && !stopPending,
    sendChatMessage,
    stopCurrentRun,
    answerQuestion,
    cancelQuestion,
    loadSessionHistory,
    clearMessages,
  };
}
