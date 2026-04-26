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
import type { ChatRuntimeAction } from '@/lib/chatRuntime/actions';
import type {
  PendingQuestionMessage,
  StreamingAssistantSegment,
  StreamingToolState,
} from '@/lib/types';
import type { ChatStateStore } from './chatStateReducer';

export function buildChatStateView(state: ChatStateStore): {
  streamingAssistantSegments: StreamingAssistantSegment[];
  streamingThinkingText: string;
  streamingTools: StreamingToolState[];
  pendingQuestions: PendingQuestionMessage[];
} {
  return {
    streamingAssistantSegments: state.streamingAssistantState.order
      .map((segmentId) => state.streamingAssistantState.segmentsById[segmentId])
      .filter((segment): segment is StreamingAssistantSegment => segment !== undefined),
    streamingThinkingText: state.streamingThinkingText,
    streamingTools: state.streamingToolState.order
      .map((toolId) => state.streamingToolState.toolsById[toolId])
      .filter((tool): tool is StreamingToolState => tool !== undefined),
    pendingQuestions: state.pendingQuestionState.order
      .map((questionId) => state.pendingQuestionState.questionsById[questionId])
      .filter((question): question is PendingQuestionMessage => question !== undefined),
  };
}

export function applyRuntimeActionsToState(
  state: ChatStateStore,
  actions: ChatRuntimeAction[],
): ChatStateStore {
  let nextState = state;
  for (const runtimeAction of actions) {
    nextState = applyRuntimeAction(nextState, runtimeAction);
  }
  return nextState;
}

function applyRuntimeAction(state: ChatStateStore, action: ChatRuntimeAction): ChatStateStore {
  switch (action.type) {
    case 'append_streaming_assistant_text':
      return appendStreamingAssistantTextState(state, action.text);
    case 'clear_streaming_assistant_text':
      return clearStreamingAssistantTextState(state);
    case 'append_streaming_thinking_text':
      return appendStreamingThinkingTextState(state, action.text);
    case 'clear_streaming_thinking_text':
      return clearStreamingThinkingTextState(state);
    case 'append_committed_messages':
      return appendCommittedMessagesState(state, action.messages);
    case 'upsert_streaming_tool':
      return upsertStreamingToolRuntimeState(state, action.tool);
    case 'clear_streaming_tools':
      return clearStreamingToolsState(state);
    case 'upsert_pending_question':
      return upsertPendingQuestionRuntimeState(state, action.question);
    case 'remove_pending_question':
      return removePendingQuestionRuntimeState(state, action.questionId);
    case 'clear_pending_questions':
      return clearPendingQuestionsState(state);
    default:
      return state;
  }
}

function appendStreamingAssistantTextState(state: ChatStateStore, text: string): ChatStateStore {
  if (!text) {
    return state;
  }

  const result = appendStreamingAssistantState(state.streamingAssistantState, text);
  let nextOrder = state.streamingItemOrder;
  if (result.createdSegmentId) {
    nextOrder = appendUniqueOrderKey(nextOrder, assistantOrderKey(result.createdSegmentId));
  }
  return {
    ...state,
    streamingAssistantState: result.state,
    streamingItemOrder: nextOrder,
  };
}

function clearStreamingAssistantTextState(state: ChatStateStore): ChatStateStore {
  return {
    ...state,
    streamingAssistantState: clearStreamingAssistantState(),
    streamingItemOrder: removeOrderKeyByPrefix(state.streamingItemOrder, STREAMING_ASSISTANT_ORDER_PREFIX),
  };
}

function appendStreamingThinkingTextState(state: ChatStateStore, text: string): ChatStateStore {
  if (!text) {
    return state;
  }
  return {
    ...state,
    streamingThinkingText: `${state.streamingThinkingText}${text}`,
  };
}

function clearStreamingThinkingTextState(state: ChatStateStore): ChatStateStore {
  if (!state.streamingThinkingText) {
    return state;
  }
  return {
    ...state,
    streamingThinkingText: '',
  };
}

function appendCommittedMessagesState(
  state: ChatStateStore,
  nextMessages: ChatStateStore['committedMessages'],
): ChatStateStore {
  if (nextMessages.length === 0) {
    return state;
  }
  return {
    ...state,
    committedMessages: [...state.committedMessages, ...nextMessages],
  };
}

function upsertStreamingToolRuntimeState(state: ChatStateStore, tool: StreamingToolState): ChatStateStore {
  const toolId = tool.id.trim();
  return {
    ...state,
    streamingAssistantState: markStreamingAssistantBoundary(state.streamingAssistantState),
    streamingItemOrder: toolId
      ? appendUniqueOrderKey(state.streamingItemOrder, toolOrderKey(toolId))
      : state.streamingItemOrder,
    streamingToolState: upsertStreamingToolState(state.streamingToolState, tool),
  };
}

function clearStreamingToolsState(state: ChatStateStore): ChatStateStore {
  return {
    ...state,
    streamingToolState: clearStreamingToolState(),
    streamingItemOrder: removeOrderKeyByPrefix(state.streamingItemOrder, STREAMING_TOOL_ORDER_PREFIX),
  };
}

function upsertPendingQuestionRuntimeState(
  state: ChatStateStore,
  question: PendingQuestionMessage,
): ChatStateStore {
  const questionId = question.questionId.trim();
  return {
    ...state,
    streamingAssistantState: markStreamingAssistantBoundary(state.streamingAssistantState),
    streamingItemOrder: questionId
      ? appendUniqueOrderKey(state.streamingItemOrder, questionOrderKey(questionId))
      : state.streamingItemOrder,
    pendingQuestionState: upsertPendingQuestionState(state.pendingQuestionState, question),
  };
}

function removePendingQuestionRuntimeState(state: ChatStateStore, questionId: string): ChatStateStore {
  const trimmedQuestionId = questionId.trim();
  return {
    ...state,
    streamingItemOrder: trimmedQuestionId
      ? removeOrderKey(state.streamingItemOrder, questionOrderKey(trimmedQuestionId))
      : state.streamingItemOrder,
    pendingQuestionState: removePendingQuestionState(state.pendingQuestionState, questionId),
  };
}

function clearPendingQuestionsState(state: ChatStateStore): ChatStateStore {
  return {
    ...state,
    pendingQuestionState: clearPendingQuestionState(),
    streamingItemOrder: removeOrderKeyByPrefix(state.streamingItemOrder, STREAMING_QUESTION_ORDER_PREFIX),
  };
}

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
