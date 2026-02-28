import { useCallback, useState } from 'react';
import { getSession, sendMessage } from '@/lib/api';
import { toErrorMessage } from '@/lib/errors';
import type { ChatMessage, SessionMessage } from '@/lib/types';

function nextID() {
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

interface UseBridgeChatResult {
  messages: ChatMessage[];
  loading: boolean;
  historyLoading: boolean;
  chatError: string;
  sendChatMessage: (message: string) => Promise<void>;
  loadSessionHistory: (sessionId: string) => Promise<void>;
  clearMessages: () => void;
}

interface UseBridgeChatOptions {
  currentSessionId: string;
  onSessionResolved?: (sessionId: string) => void;
}

function mapSessionRoleToChatMessage(message: SessionMessage): ChatMessage | null {
  switch (message.role) {
    case 'user':
      return {
        id: nextID(),
        kind: 'user',
        content: message.text,
      };
    case 'assistant':
      return {
        id: nextID(),
        kind: 'assistant',
        content: message.text,
      };
    case 'system':
      return {
        id: nextID(),
        kind: 'assistant',
        content: message.text ? `[system] ${message.text}` : '[system]',
      };
    case 'tool':
      return {
        id: nextID(),
        kind: 'assistant',
        content: message.text ? `[tool] ${message.text}` : '[tool]',
      };
    default:
      return null;
  }
}

function mapSessionMessagesToChat(messages: SessionMessage[]): ChatMessage[] {
  const output: ChatMessage[] = [];
  for (const message of messages) {
    const mapped = mapSessionRoleToChatMessage(message);
    if (!mapped) {
      continue;
    }
    output.push(mapped);
  }
  return output;
}

function buildErrorMessage(messageText: string): ChatMessage {
  return {
    id: nextID(),
    kind: 'error',
    content: messageText,
    error: messageText,
  };
}

export function useBridgeChat(options: UseBridgeChatOptions): UseBridgeChatResult {
  const { currentSessionId, onSessionResolved } = options;
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [loading, setLoading] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [chatError, setChatError] = useState('');

  const appendMessage = useCallback((message: ChatMessage) => {
    setMessages((previous) => [...previous, message]);
  }, []);

  const replaceWithErrorMessage = useCallback((messageText: string) => {
    setMessages([buildErrorMessage(messageText)]);
  }, []);

  const sendChatMessage = useCallback(async (message: string) => {
    setChatError('');
    appendMessage({
      id: nextID(),
      kind: 'user',
      content: message,
    });
    setLoading(true);

    try {
      const reply = await sendMessage(message, currentSessionId || undefined);
      appendMessage({
        id: nextID(),
        kind: 'assistant',
        content: reply.message,
      });
      if (reply.session_id && reply.session_id !== currentSessionId) {
        onSessionResolved?.(reply.session_id);
      }
    } catch (error) {
      const messageText = toErrorMessage(error);
      setChatError(messageText);
      appendMessage(buildErrorMessage(messageText));
    } finally {
      setLoading(false);
    }
  }, [appendMessage, currentSessionId, onSessionResolved]);

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
    sendChatMessage,
    loadSessionHistory,
    clearMessages,
  };
}
