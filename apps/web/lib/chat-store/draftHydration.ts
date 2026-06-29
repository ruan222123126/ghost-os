import {
  clearPendingQuestionState,
  clearStreamingAssistantState,
  clearStreamingThinkingState,
  clearStreamingToolState,
  type PendingQuestionState,
  STREAMING_ASSISTANT_ORDER_PREFIX,
  STREAMING_THINKING_ORDER_PREFIX,
  type StreamingAssistantState,
  type StreamingThinkingState,
  type StreamingToolTableState,
} from '@/lib/chat-stream/streamState';
import { buildAssistantMessage, buildThinkingMessage } from '@/lib/chatMessages';
import { committedMessageCoversDraftMessage } from '@/lib/chatMessageEquivalence';
import type { ChatMessage, SessionTurnDraft, StreamingToolState } from '@/lib/types';

interface DraftHydratedState {
  pendingQuestionState: PendingQuestionState;
  streamingAssistantState: StreamingAssistantState;
  streamingThinkingState: StreamingThinkingState;
  streamingItemOrder: string[];
  streamingToolState: StreamingToolTableState;
}

const ACTIVE_TOOL_STATUSES = new Set(['running', 'pending', 'in_progress']);
const ERROR_TOOL_STATUS = 'error';

export function buildDraftHydratedState(
  draft: SessionTurnDraft | null | undefined,
  sessionId: string,
  committedMessages: ChatMessage[] = [],
): DraftHydratedState {
  if (!draft) {
    return {
      pendingQuestionState: clearPendingQuestionState(),
      streamingAssistantState: clearStreamingAssistantState(),
      streamingThinkingState: clearStreamingThinkingState(),
      streamingItemOrder: [],
      streamingToolState: clearStreamingToolState(),
    };
  }

  const dedupedDraft = removeCommittedDraftOverlap(draft);

  return {
    pendingQuestionState: buildPendingQuestionState(dedupedDraft, sessionId),
    streamingAssistantState: buildSegmentState(
      dedupedDraft.assistant_segments,
      dedupedDraft.item_order,
      STREAMING_ASSISTANT_ORDER_PREFIX,
      clearStreamingAssistantState(),
    ),
    streamingThinkingState: buildSegmentState(
      dedupedDraft.thinking_segments,
      dedupedDraft.item_order,
      STREAMING_THINKING_ORDER_PREFIX,
      clearStreamingThinkingState(),
    ),
    streamingItemOrder: [...dedupedDraft.item_order],
    streamingToolState: buildToolState(dedupedDraft),
  };

  function removeCommittedDraftOverlap(source: SessionTurnDraft): SessionTurnDraft {
    if (committedMessages.length === 0) {
      return source;
    }

    const draftMessages = buildDraftMessages(source);
    if (draftMessages.length === 0) {
      return source;
    }

    const overlapCount = countCoveredDraftPrefix(committedMessages, draftMessages);
    if (overlapCount === 0) {
      return source;
    }

    return trimDraftPrefix(source, draftMessages.slice(0, overlapCount).map((message) => message.id));
  }
}

function buildSegmentState<TSegment extends { id: string; content: string }>(
  segments: TSegment[],
  itemOrder: string[],
  orderPrefix: string,
  initialState: StreamingAssistantState | StreamingThinkingState,
): StreamingAssistantState | StreamingThinkingState {
  const segmentsById = Object.fromEntries(segments.map((segment) => [segment.id, { ...segment }]));
  const activeSegmentId = resolveActiveSegmentId(itemOrder, orderPrefix, segmentsById);
  return {
    ...initialState,
    activeSegmentId,
    nextSegmentSeq: resolveNextSegmentSeq(segments),
    order: segments.map((segment) => segment.id),
    segmentsById,
  };
}

function resolveActiveSegmentId(
  itemOrder: string[],
  orderPrefix: string,
  segmentsById: Record<string, { id: string; content: string }>,
): string {
  const lastItem = itemOrder[itemOrder.length - 1];
  if (!lastItem?.startsWith(orderPrefix)) {
    return '';
  }

  const segmentId = lastItem.slice(orderPrefix.length);
  return segmentsById[segmentId] ? segmentId : '';
}

function resolveNextSegmentSeq(segments: { id: string }[]): number {
  let maxSeq = 0;
  for (const segment of segments) {
    const suffix = Number(segment.id.split(':').at(-1));
    if (Number.isInteger(suffix) && suffix > maxSeq) {
      maxSeq = suffix;
    }
  }
  return maxSeq + 1 || 1;
}

function buildToolState(draft: SessionTurnDraft): StreamingToolTableState {
  const tools: StreamingToolState[] = draft.tools.map((tool) => ({
    id: tool.id,
    content: tool.content,
    toolInput: tool.tool_input,
    toolName: tool.tool_name,
    toolStatus: normalizeDraftToolStatus(draft.status, tool.tool_status),
    toolCallId: tool.tool_call_id,
    traceId: tool.trace_id,
  }));

  return {
    order: tools.map((tool) => tool.id),
    toolsById: Object.fromEntries(tools.map((tool) => [tool.id, tool])),
  };
}

function buildDraftMessages(draft: SessionTurnDraft): ChatMessage[] {
  const assistantById = new Map(
    draft.assistant_segments.map((segment) => [segment.id, buildAssistantMessage(segment.content, segment.id)]),
  );
  const thinkingById = new Map(
    draft.thinking_segments.map((segment) => [segment.id, buildThinkingMessage(segment.content, segment.id)]),
  );
  const toolsById = new Map(
    draft.tools.map((tool) => [tool.id, buildDraftToolMessage(draft, tool)]),
  );
  const pendingQuestionsById = new Map(
    draft.pending_questions.map((question) => [question.question_id, question]),
  );
  const seen = new Set<string>();
  const messages: ChatMessage[] = [];

  for (const item of draft.item_order) {
    const mapped = mapDraftOrderItemToMessage(item, {
      assistantById,
      thinkingById,
      toolsById,
      pendingQuestionsById,
      traceId: draft.trace_id,
    });
    if (!mapped || seen.has(mapped.id)) {
      continue;
    }
    messages.push(mapped);
    seen.add(mapped.id);
  }

  return messages;
}

function mapDraftOrderItemToMessage(
  item: string,
  lookup: {
    assistantById: Map<string, ChatMessage>;
    thinkingById: Map<string, ChatMessage>;
    toolsById: Map<string, ChatMessage>;
    pendingQuestionsById: Map<string, SessionTurnDraft['pending_questions'][number]>;
    traceId: string;
  },
): ChatMessage | null {
  if (item.startsWith(STREAMING_ASSISTANT_ORDER_PREFIX)) {
    return lookup.assistantById.get(item.slice(STREAMING_ASSISTANT_ORDER_PREFIX.length)) ?? null;
  }
  if (item.startsWith(STREAMING_THINKING_ORDER_PREFIX)) {
    return lookup.thinkingById.get(item.slice(STREAMING_THINKING_ORDER_PREFIX.length)) ?? null;
  }
  if (item.startsWith('tool:')) {
    return lookup.toolsById.get(item.slice('tool:'.length)) ?? null;
  }
  if (item.startsWith('question:')) {
    const questionId = item.slice('question:'.length);
    const question = lookup.pendingQuestionsById.get(questionId);
    if (!question) {
      return null;
    }
    return {
      id: `stream-question:${lookup.traceId}:${question.question_id}`,
      kind: 'pending_question',
      content: question.prompt,
      questionId: question.question_id,
      sessionId: '',
      selectionMode: question.selection_mode,
      options: question.options,
    };
  }
  return null;
}

function buildDraftToolMessage(
  draft: SessionTurnDraft,
  tool: SessionTurnDraft['tools'][number],
): ChatMessage {
  return {
    id: tool.id,
    kind: 'tool',
    content: tool.content,
    toolInput: tool.tool_input,
    toolName: tool.tool_name,
    toolStatus: normalizeDraftToolStatus(draft.status, tool.tool_status),
    toolCallId: tool.tool_call_id,
    traceId: tool.trace_id,
  };
}

function countCoveredDraftPrefix(
  committedMessages: ChatMessage[],
  draftMessages: ChatMessage[],
): number {
  let committedIndex = Math.max(0, committedMessages.length - draftMessages.length);
  let matched = 0;

  while (committedIndex < committedMessages.length && matched < draftMessages.length) {
    if (committedMessageCoversDraftMessage(committedMessages[committedIndex], draftMessages[matched])) {
      committedIndex += 1;
      matched += 1;
      continue;
    }
    if (matched > 0) {
      break;
    }
    committedIndex += 1;
  }

  return matched;
}

function trimDraftPrefix(
  draft: SessionTurnDraft,
  removedMessageIds: string[],
): SessionTurnDraft {
  if (removedMessageIds.length === 0) {
    return draft;
  }

  const removedIds = new Set(removedMessageIds);
  const removedQuestionIds = new Set(
    removedMessageIds
      .filter((messageId) => messageId.startsWith(`stream-question:${draft.trace_id}:`))
      .map((messageId) => messageId.slice(`stream-question:${draft.trace_id}:`.length)),
  );

  return {
    ...draft,
    pending_questions: draft.pending_questions.filter((question) => !removedQuestionIds.has(question.question_id)),
    assistant_segments: draft.assistant_segments.filter((segment) => !removedIds.has(segment.id)),
    thinking_segments: draft.thinking_segments.filter((segment) => !removedIds.has(segment.id)),
    tools: draft.tools.filter((tool) => !removedIds.has(tool.id)),
    item_order: draft.item_order.filter((item) => !removedDraftOrderItem(item, removedIds, removedQuestionIds)),
  };
}

function removedDraftOrderItem(
  item: string,
  removedIds: Set<string>,
  removedQuestionIds: Set<string>,
): boolean {
  if (item.startsWith(STREAMING_ASSISTANT_ORDER_PREFIX)) {
    return removedIds.has(item.slice(STREAMING_ASSISTANT_ORDER_PREFIX.length));
  }
  if (item.startsWith(STREAMING_THINKING_ORDER_PREFIX)) {
    return removedIds.has(item.slice(STREAMING_THINKING_ORDER_PREFIX.length));
  }
  if (item.startsWith('tool:')) {
    return removedIds.has(item.slice('tool:'.length));
  }
  if (item.startsWith('question:')) {
    return removedQuestionIds.has(item.slice('question:'.length));
  }
  return false;
}

function normalizeDraftToolStatus(
  draftStatus: SessionTurnDraft['status'],
  toolStatus?: string,
): string | undefined {
  const normalized = toolStatus?.trim().toLowerCase();
  if (draftStatus === 'error' && normalized && ACTIVE_TOOL_STATUSES.has(normalized)) {
    return ERROR_TOOL_STATUS;
  }
  return toolStatus;
}

function buildPendingQuestionState(draft: SessionTurnDraft, sessionId: string): PendingQuestionState {
  const questions = draft.pending_questions.map((question) => ({
    id: `stream-question:${draft.trace_id}:${question.question_id}`,
    kind: 'pending_question' as const,
    content: question.prompt,
    options: question.options,
    questionId: question.question_id,
    selectionMode: question.selection_mode,
    sessionId,
  }));

  return {
    order: questions.map((question) => question.questionId),
    questionsById: Object.fromEntries(questions.map((question) => [question.questionId, question])),
  };
}
