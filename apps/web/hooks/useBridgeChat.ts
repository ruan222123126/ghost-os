import { useCallback, useState } from 'react';
import { getSession, sendHumanResponse, sendMessage } from '@/lib/api';
import { toErrorMessage } from '@/lib/errors';
import type {
  AgentSendAwaitingHumanResponse,
  AgentSendResponse,
  ChatMessage,
  PendingQuestionMessage,
  SessionMessage,
} from '@/lib/types';

function nextID() {
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

interface UseBridgeChatResult {
  messages: ChatMessage[];
  loading: boolean;
  historyLoading: boolean;
  chatError: string;
  hasPendingQuestion: boolean;
  sendChatMessage: (message: string) => Promise<void>;
  answerQuestion: (questionId: string, answer: string) => Promise<void>;
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
      // Web 聊天流统一只渲染 user/assistant 两类气泡，system/tool 作为 assistant 文本呈现。
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

function isAwaitingHumanResponse(response: AgentSendResponse): response is AgentSendAwaitingHumanResponse {
  return response.status === 'awaiting_human';
}

function buildPendingQuestionMessage(response: AgentSendAwaitingHumanResponse): PendingQuestionMessage {
  return {
    id: nextID(),
    kind: 'pending_question',
    content: response.prompt,
    questionId: response.question_id,
    sessionId: response.session_id,
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
    const trimmed = message.trim();
    // 新会话下空输入没有意义；已有 session 时允许空消息用于“继续执行”。
    if (!trimmed && !currentSessionId.trim()) {
      return;
    }

    setChatError('');
    if (trimmed) {
      appendMessage({
        id: nextID(),
        kind: 'user',
        content: trimmed,
      });
    }
    setLoading(true);

    try {
      const reply = await sendMessage(trimmed, currentSessionId || undefined);
      if (reply.session_id && reply.session_id !== currentSessionId) {
        onSessionResolved?.(reply.session_id);
      }

      if (isAwaitingHumanResponse(reply)) {
        // 将 ask_human 回合渲染为待回答卡片，避免丢失 question_id/session_id。
        appendMessage(buildPendingQuestionMessage(reply));
      } else {
        appendMessage({
          id: nextID(),
          kind: 'assistant',
          content: reply.message,
        });
      }
    } catch (error) {
      const messageText = toErrorMessage(error);
      setChatError(messageText);
      appendMessage(buildErrorMessage(messageText));
    } finally {
      setLoading(false);
    }
  }, [appendMessage, currentSessionId, onSessionResolved]);

  const answerQuestion = useCallback(async (questionId: string, answer: string) => {
    const trimmedQuestionID = questionId.trim();
    const trimmedAnswer = answer.trim();
    if (!trimmedQuestionID || !trimmedAnswer) {
      return;
    }

    const pending = messages.find(
      (message): message is PendingQuestionMessage =>
        message.kind === 'pending_question' && message.questionId === trimmedQuestionID
    );
    if (!pending) {
      return;
    }

    setChatError('');
    setLoading(true);
    try {
      await sendHumanResponse(pending.sessionId, pending.questionId, trimmedAnswer);
      setMessages((previous) =>
        previous.flatMap((message) => {
          if (message.kind === 'pending_question' && message.questionId === trimmedQuestionID) {
            // 题卡被回答后替换为用户消息，维持对话时间线连续性。
            return [
              {
                id: nextID(),
                kind: 'user' as const,
                content: trimmedAnswer,
              },
            ];
          }
          return [message];
        })
      );

      // 发送空消息触发 bridge 继续运行暂停中的 agent 回合。
      const reply = await sendMessage('', pending.sessionId);
      if (reply.session_id && reply.session_id !== currentSessionId) {
        onSessionResolved?.(reply.session_id);
      }

      if (isAwaitingHumanResponse(reply)) {
        appendMessage(buildPendingQuestionMessage(reply));
      } else {
        appendMessage({
          id: nextID(),
          kind: 'assistant',
          content: reply.message,
        });
      }
    } catch (error) {
      const messageText = toErrorMessage(error);
      setChatError(messageText);
      appendMessage(buildErrorMessage(messageText));
    } finally {
      setLoading(false);
    }
  }, [appendMessage, currentSessionId, messages, onSessionResolved]);

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
    hasPendingQuestion: messages.some((message) => message.kind === 'pending_question'),
    sendChatMessage,
    answerQuestion,
    loadSessionHistory,
    clearMessages,
  };
}
