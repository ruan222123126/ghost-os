import { useCallback, useRef, useState } from 'react';
import { buildErrorMessage } from '@/lib/chatMessages';
import type { ChatMessage } from '@/lib/types';
import type { ActiveAgentRun, ChatStateControls } from './types';

export function useChatState(): ChatStateControls {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [loading, setLoading] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [chatError, setChatError] = useState('');
  const [activeRun, setActiveRunState] = useState<ActiveAgentRun | null>(null);
  const [stopPending, setStopPendingState] = useState(false);
  const activeRunRef = useRef<ActiveAgentRun | null>(null);
  const stopPendingRef = useRef(false);

  const setActiveRun = useCallback((value: ActiveAgentRun | null) => {
    activeRunRef.current = value;
    setActiveRunState(value);
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

  const clearChatError = useCallback(() => {
    setChatError('');
  }, []);

  const replaceWithErrorMessage = useCallback((messageText: string) => {
    setMessages([buildErrorMessage(messageText)]);
  }, []);

  const appendErrorMessage = useCallback((messageText: string) => {
    setChatError(messageText);
    appendMessages([buildErrorMessage(messageText)]);
  }, [appendMessages]);

  const clearMessages = useCallback(() => {
    clearChatError();
    setMessages([]);
  }, [clearChatError]);

  return {
    messages,
    loading,
    historyLoading,
    chatError,
    activeRun,
    stopPending,
    activeRunRef,
    stopPendingRef,
    setMessages,
    setLoading,
    setHistoryLoading,
    setChatError,
    setActiveRun,
    setStopPending,
    appendMessages,
    clearChatError,
    replaceWithErrorMessage,
    appendErrorMessage,
    clearMessages,
  };
}
