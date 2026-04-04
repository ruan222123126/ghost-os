import type {
  PendingQuestionMessage,
  StreamingAssistantSegment,
  StreamingToolState,
} from '@/lib/types';

export const STREAMING_ASSISTANT_ORDER_PREFIX = 'assistant:';
export const STREAMING_TOOL_ORDER_PREFIX = 'tool:';
export const STREAMING_QUESTION_ORDER_PREFIX = 'question:';

const ASSISTANT_SEGMENT_ID_PREFIX = 'stream-segment:assistant:';
const INITIAL_ASSISTANT_SEGMENT_SEQUENCE = 1;

export interface PendingQuestionState {
  order: string[];
  questionsById: Record<string, PendingQuestionMessage>;
}

export interface StreamingToolTableState {
  order: string[];
  toolsById: Record<string, StreamingToolState>;
}

export interface StreamingAssistantState {
  activeSegmentId: string;
  nextSegmentSeq: number;
  order: string[];
  segmentsById: Record<string, StreamingAssistantSegment>;
}

interface AppendStreamingAssistantResult {
  createdSegmentId?: string;
  state: StreamingAssistantState;
}

export function clearStreamingAssistantState(): StreamingAssistantState {
  return {
    activeSegmentId: '',
    nextSegmentSeq: INITIAL_ASSISTANT_SEGMENT_SEQUENCE,
    order: [],
    segmentsById: {},
  };
}

export function markStreamingAssistantBoundary(
  state: StreamingAssistantState,
): StreamingAssistantState {
  if (!state.activeSegmentId) {
    return state;
  }
  return {
    ...state,
    activeSegmentId: '',
  };
}

export function appendStreamingAssistantState(
  state: StreamingAssistantState,
  delta: string,
): AppendStreamingAssistantResult {
  if (!delta) {
    return { state };
  }

  const segmentId = state.activeSegmentId || createAssistantSegmentID(state.nextSegmentSeq);
  const currentSegment = state.segmentsById[segmentId];
  if (currentSegment) {
    return {
      state: {
        ...state,
        activeSegmentId: segmentId,
        segmentsById: {
          ...state.segmentsById,
          [segmentId]: {
            ...currentSegment,
            content: `${currentSegment.content}${delta}`,
          },
        },
      },
    };
  }

  return {
    createdSegmentId: segmentId,
    state: {
      ...state,
      activeSegmentId: segmentId,
      nextSegmentSeq: state.activeSegmentId ? state.nextSegmentSeq : state.nextSegmentSeq + 1,
      order: [...state.order, segmentId],
      segmentsById: {
        ...state.segmentsById,
        [segmentId]: {
          id: segmentId,
          content: delta,
        },
      },
    },
  };
}

function createAssistantSegmentID(sequence: number): string {
  return `${ASSISTANT_SEGMENT_ID_PREFIX}${sequence}`;
}

export function upsertStreamingToolState(
  state: StreamingToolTableState,
  tool: StreamingToolState,
): StreamingToolTableState {
  const toolId = tool.id.trim();
  if (!toolId) {
    return state;
  }

  const current = state.toolsById[toolId];
  return {
    order: current ? state.order : [...state.order, toolId],
    toolsById: {
      ...state.toolsById,
      [toolId]: current ? { ...current, ...tool, id: toolId } : { ...tool, id: toolId },
    },
  };
}

export function clearStreamingToolState(): StreamingToolTableState {
  return {
    order: [],
    toolsById: {},
  };
}

export function upsertPendingQuestionState(
  state: PendingQuestionState,
  question: PendingQuestionMessage,
): PendingQuestionState {
  const questionId = question.questionId.trim();
  if (!questionId) {
    return state;
  }

  const current = state.questionsById[questionId];
  return {
    order: current ? state.order : [...state.order, questionId],
    questionsById: {
      ...state.questionsById,
      [questionId]: current ? { ...current, ...question } : { ...question },
    },
  };
}

export function removePendingQuestionState(
  state: PendingQuestionState,
  questionId: string,
): PendingQuestionState {
  const trimmedQuestionId = questionId.trim();
  if (!trimmedQuestionId || !state.questionsById[trimmedQuestionId]) {
    return state;
  }

  const nextQuestionsById = { ...state.questionsById };
  delete nextQuestionsById[trimmedQuestionId];

  return {
    order: state.order.filter((currentId) => currentId !== trimmedQuestionId),
    questionsById: nextQuestionsById,
  };
}

export function clearPendingQuestionState(): PendingQuestionState {
  return {
    order: [],
    questionsById: {},
  };
}
