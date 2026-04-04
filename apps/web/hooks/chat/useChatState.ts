import { useCallback, useRef, useState } from 'react';
import { buildErrorMessage } from '@/lib/chatMessages';
import {
  appendStreamingAssistantState,
  clearPendingQuestionState,
  clearStreamingAssistantState,
  clearStreamingToolState,
  markStreamingAssistantBoundary,
  removePendingQuestionState,
  STREAMING_ASSISTANT_ORDER_PREFIX,
  STREAMING_QUESTION_ORDER_PREFIX,
  STREAMING_TOOL_ORDER_PREFIX,
  upsertPendingQuestionState,
  upsertStreamingToolState,
} from '@/lib/chatStream';
import type { ChatMessage, PendingQuestionMessage, StreamingToolState } from '@/lib/types';
import type { ActiveAgentRun, ChatStateControls } from './types';

function appendUniqueOrderKey(order: string[], key: string): string[] {
  return order.includes(key) ? order : [...order, key];
}

function removeOrderKey(order: string[], key: string): string[] {
  return order.filter((current) => current !== key);
}

function removeOrderKeyByPrefix(order: string[], prefix: string): string[] {
  return order.filter((current) => !current.startsWith(prefix));
}

function assistantOrderKey(segmentId: string): string {
  return `${STREAMING_ASSISTANT_ORDER_PREFIX}${segmentId}`;
}

function toolOrderKey(toolId: string): string {
  return `${STREAMING_TOOL_ORDER_PREFIX}${toolId}`;
}

function questionOrderKey(questionId: string): string {
  return `${STREAMING_QUESTION_ORDER_PREFIX}${questionId}`;
}

export function useChatState(): ChatStateControls {
  const [committedMessages, setCommittedMessages] = useState<ChatMessage[]>([]);
  const [streamingAssistantState, setStreamingAssistantState] = useState(clearStreamingAssistantState);
  const [streamingItemOrder, setStreamingItemOrder] = useState<string[]>([]);
  const [streamingToolState, setStreamingToolState] = useState(clearStreamingToolState);
  const [pendingQuestionState, setPendingQuestionState] = useState(clearPendingQuestionState);
  const [loading, setLoading] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [loadingOlderHistory, setLoadingOlderHistory] = useState(false);
  const [chatError, setChatError] = useState('');
  const [activeRun, setActiveRunState] = useState<ActiveAgentRun | null>(null);
  const [stopPending, setStopPendingState] = useState(false);
  const [hasOlderHistory, setHasOlderHistory] = useState(false);
  const [nextHistoryBefore, setNextHistoryBefore] = useState<number | null>(null);
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

  const appendCommittedMessages = useCallback((nextMessages: ChatMessage[]) => {
    if (nextMessages.length === 0) {
      return;
    }

    setCommittedMessages((previous) => [...previous, ...nextMessages]);
  }, []);

  const clearChatError = useCallback(() => {
    setChatError('');
  }, []);

  const clearStreamingTools = useCallback(() => {
    setStreamingToolState(clearStreamingToolState());
    setStreamingItemOrder((previous) => removeOrderKeyByPrefix(previous, STREAMING_TOOL_ORDER_PREFIX));
  }, []);

  const clearPendingQuestions = useCallback(() => {
    setPendingQuestionState(clearPendingQuestionState());
    setStreamingItemOrder((previous) => removeOrderKeyByPrefix(previous, STREAMING_QUESTION_ORDER_PREFIX));
  }, []);

  const clearStreamingAssistantText = useCallback(() => {
    setStreamingAssistantState(clearStreamingAssistantState());
    setStreamingItemOrder((previous) => removeOrderKeyByPrefix(previous, STREAMING_ASSISTANT_ORDER_PREFIX));
  }, []);

  const clearStreamingState = useCallback(() => {
    clearStreamingAssistantText();
    clearStreamingTools();
  }, [clearStreamingAssistantText, clearStreamingTools]);

  const replaceWithErrorMessage = useCallback((messageText: string) => {
    setCommittedMessages([buildErrorMessage(messageText)]);
    setStreamingAssistantState(clearStreamingAssistantState());
    setStreamingItemOrder([]);
    setStreamingToolState(clearStreamingToolState());
    setPendingQuestionState(clearPendingQuestionState());
  }, []);

  const appendErrorMessage = useCallback((messageText: string) => {
    setChatError(messageText);
    appendCommittedMessages([buildErrorMessage(messageText)]);
  }, [appendCommittedMessages]);

  const appendStreamingAssistantTextState = useCallback((text: string) => {
    if (!text) {
      return;
    }

    setStreamingAssistantState((previous) => {
      const result = appendStreamingAssistantState(previous, text);
      const createdSegmentId = result.createdSegmentId;
      if (createdSegmentId) {
        setStreamingItemOrder((currentOrder) => {
          return appendUniqueOrderKey(currentOrder, assistantOrderKey(createdSegmentId));
        });
      }
      return result.state;
    });
  }, []);

  const upsertStreamingTool = useCallback((tool: StreamingToolState) => {
    const toolId = tool.id.trim();
    setStreamingAssistantState(markStreamingAssistantBoundary);
    if (toolId) {
      setStreamingItemOrder((previous) => appendUniqueOrderKey(previous, toolOrderKey(toolId)));
    }
    setStreamingToolState((previous) => upsertStreamingToolState(previous, tool));
  }, []);

  const upsertPendingQuestion = useCallback((question: PendingQuestionMessage) => {
    const questionId = question.questionId.trim();
    setStreamingAssistantState(markStreamingAssistantBoundary);
    if (questionId) {
      setStreamingItemOrder((previous) => appendUniqueOrderKey(previous, questionOrderKey(questionId)));
    }
    setPendingQuestionState((previous) => upsertPendingQuestionState(previous, question));
  }, []);

  const removePendingQuestion = useCallback((questionId: string) => {
    const trimmedQuestionId = questionId.trim();
    if (trimmedQuestionId) {
      setStreamingItemOrder((previous) => removeOrderKey(previous, questionOrderKey(trimmedQuestionId)));
    }
    setPendingQuestionState((previous) => removePendingQuestionState(previous, questionId));
  }, []);

  const clearMessages = useCallback(() => {
    clearChatError();
    setCommittedMessages([]);
    setStreamingAssistantState(clearStreamingAssistantState());
    setStreamingItemOrder([]);
    setStreamingToolState(clearStreamingToolState());
    setPendingQuestionState(clearPendingQuestionState());
    setLoadingOlderHistory(false);
    setHasOlderHistory(false);
    setNextHistoryBefore(null);
  }, [clearChatError]);

  return {
    committedMessages,
    streamingAssistantSegments: streamingAssistantState.order
      .map((segmentId) => streamingAssistantState.segmentsById[segmentId])
      .filter((segment) => segment !== undefined),
    streamingItemOrder,
    streamingTools: streamingToolState.order
      .map((toolId) => streamingToolState.toolsById[toolId])
      .filter((tool) => tool !== undefined),
    pendingQuestions: pendingQuestionState.order
      .map((questionId) => pendingQuestionState.questionsById[questionId])
      .filter((question) => question !== undefined),
    loading,
    historyLoading,
    loadingOlderHistory,
    chatError,
    activeRun,
    stopPending,
    hasOlderHistory,
    nextHistoryBefore,
    activeRunRef,
    stopPendingRef,
    setCommittedMessages,
    setLoading,
    setHistoryLoading,
    setLoadingOlderHistory,
    setChatError,
    setActiveRun,
    setStopPending,
    setHasOlderHistory,
    setNextHistoryBefore,
    appendCommittedMessages,
    clearChatError,
    replaceWithErrorMessage,
    appendErrorMessage,
    appendStreamingAssistantText: appendStreamingAssistantTextState,
    clearStreamingAssistantText,
    clearStreamingState,
    upsertStreamingTool,
    clearStreamingTools,
    upsertPendingQuestion,
    removePendingQuestion,
    clearPendingQuestions,
    clearMessages,
  };
}
