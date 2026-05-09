import type {
  PendingQuestionMessage,
  StreamingAssistantSegment,
  StreamingThinkingSegment,
  StreamingToolState,
} from '@/lib/types';

export const STREAMING_ASSISTANT_ORDER_PREFIX = 'assistant:';
export const STREAMING_THINKING_ORDER_PREFIX = 'thinking:';
export const STREAMING_TOOL_ORDER_PREFIX = 'tool:';
export const STREAMING_QUESTION_ORDER_PREFIX = 'question:';

const ASSISTANT_SEGMENT_ID_PREFIX = 'stream-segment:assistant:';
const THINKING_SEGMENT_ID_PREFIX = 'stream-segment:thinking:';
const INITIAL_STREAMING_SEGMENT_SEQUENCE = 1;

interface StreamingSegmentBase {
  id: string;
  content: string;
}

interface StreamingSegmentState<TSegment extends StreamingSegmentBase> {
  activeSegmentId: string;
  nextSegmentSeq: number;
  order: string[];
  segmentsById: Record<string, TSegment>;
}

export interface PendingQuestionState {
  order: string[];
  questionsById: Record<string, PendingQuestionMessage>;
}

export interface StreamingToolTableState {
  order: string[];
  toolsById: Record<string, StreamingToolState>;
}

export type StreamingAssistantState = StreamingSegmentState<StreamingAssistantSegment>;
export type StreamingThinkingState = StreamingSegmentState<StreamingThinkingSegment>;

interface AppendStreamingSegmentResult<TState> {
  createdSegmentId?: string;
  state: TState;
}

export function clearStreamingAssistantState(): StreamingAssistantState {
  return clearStreamingSegmentState<StreamingAssistantSegment>();
}

export function clearStreamingThinkingState(): StreamingThinkingState {
  return clearStreamingSegmentState<StreamingThinkingSegment>();
}

export function markStreamingAssistantBoundary(
  state: StreamingAssistantState,
): StreamingAssistantState {
  return markStreamingSegmentBoundary(state);
}

export function markStreamingThinkingBoundary(
  state: StreamingThinkingState,
): StreamingThinkingState {
  return markStreamingSegmentBoundary(state);
}

export function appendStreamingAssistantState(
  state: StreamingAssistantState,
  delta: string,
): AppendStreamingSegmentResult<StreamingAssistantState> {
  return appendStreamingSegmentState(state, delta, createAssistantSegmentID);
}

export function appendStreamingThinkingState(
  state: StreamingThinkingState,
  delta: string,
): AppendStreamingSegmentResult<StreamingThinkingState> {
  return appendStreamingSegmentState(state, delta, createThinkingSegmentID);
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

function clearStreamingSegmentState<TSegment extends StreamingSegmentBase>(): StreamingSegmentState<TSegment> {
  return {
    activeSegmentId: '',
    nextSegmentSeq: INITIAL_STREAMING_SEGMENT_SEQUENCE,
    order: [],
    segmentsById: {},
  };
}

function markStreamingSegmentBoundary<TSegment extends StreamingSegmentBase>(
  state: StreamingSegmentState<TSegment>,
): StreamingSegmentState<TSegment> {
  if (!state.activeSegmentId) {
    return state;
  }
  return {
    ...state,
    activeSegmentId: '',
  };
}

function appendStreamingSegmentState<TSegment extends StreamingSegmentBase>(
  state: StreamingSegmentState<TSegment>,
  delta: string,
  createSegmentID: (sequence: number) => string,
): AppendStreamingSegmentResult<StreamingSegmentState<TSegment>> {
  if (!delta) {
    return { state };
  }

  const segmentId = state.activeSegmentId || createSegmentID(state.nextSegmentSeq);
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
        } as TSegment,
      },
    },
  };
}

function createAssistantSegmentID(sequence: number): string {
  return createSegmentID(ASSISTANT_SEGMENT_ID_PREFIX, sequence);
}

function createThinkingSegmentID(sequence: number): string {
  return createSegmentID(THINKING_SEGMENT_ID_PREFIX, sequence);
}

function createSegmentID(prefix: string, sequence: number): string {
  return `${prefix}${sequence}`;
}
