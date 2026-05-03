import { buildAssistantMessage, buildThinkingMessage } from '@/lib/chatMessages';
import {
  STREAMING_ASSISTANT_ORDER_PREFIX,
  STREAMING_THINKING_ORDER_PREFIX,
  STREAMING_QUESTION_ORDER_PREFIX,
  STREAMING_TOOL_ORDER_PREFIX,
} from '@/lib/chatStream';
import type {
  ChatMessage,
  PendingQuestionMessage,
  StreamingAssistantSegment,
  StreamingThinkingSegment,
  ToolChatMessage,
} from '@/lib/types';
import type { MessageListProps } from './types';

export interface StreamingMessageRow {
  key: string;
  message: ChatMessage;
}

interface StreamingRowOrderInput {
  pendingQuestions: PendingQuestionMessage[];
  streamingAssistantSegments: StreamingAssistantSegment[];
  streamingThinkingSegments: StreamingThinkingSegment[];
  streamingItemOrder: string[];
  streamingTools: MessageListProps['streamingTools'];
}

interface StreamingOrderLookup {
  assistantSegmentsById: Map<string, StreamingAssistantSegment>;
  thinkingSegmentsById: Map<string, StreamingThinkingSegment>;
  questionsById: Map<string, PendingQuestionMessage>;
  toolsById: Map<string, MessageListProps['streamingTools'][number]>;
}

export function getOrderedStreamingRows(options: StreamingRowOrderInput): StreamingMessageRow[] {
  const rows: StreamingMessageRow[] = [];
  const rowKeys = new Set<string>();
  const assistantSegmentsById = new Map(
    options.streamingAssistantSegments.map((segment) => [segment.id, segment]),
  );
  const thinkingSegmentsById = new Map(
    options.streamingThinkingSegments.map((segment) => [segment.id, segment]),
  );
  const toolsById = new Map(
    options.streamingTools.map((tool) => [tool.id, tool]),
  );
  const questionsById = new Map(
    options.pendingQuestions.map((question) => [question.questionId, question]),
  );

  for (const orderKey of options.streamingItemOrder) {
    const row = mapStreamingOrderToRow(orderKey, {
      assistantSegmentsById,
      thinkingSegmentsById,
      questionsById,
      toolsById,
    });
    if (!row || rowKeys.has(row.key)) {
      continue;
    }
    rows.push(row);
    rowKeys.add(row.key);
  }

  appendMissingStreamingRows(rows, rowKeys, options);
  return rows;
}

function appendMissingStreamingRows(
  rows: StreamingMessageRow[],
  rowKeys: Set<string>,
  options: Omit<StreamingRowOrderInput, 'streamingItemOrder'>,
): void {
  for (const segment of options.streamingAssistantSegments) {
    appendStreamingRow(rows, rowKeys, buildStreamingAssistantRow(segment));
  }

  for (const segment of options.streamingThinkingSegments) {
    appendStreamingRow(rows, rowKeys, buildStreamingThinkingRow(segment));
  }

  for (const tool of options.streamingTools) {
    appendStreamingRow(rows, rowKeys, buildStreamingToolRow(tool));
  }

  for (const question of options.pendingQuestions) {
    appendStreamingRow(rows, rowKeys, {
      key: question.id,
      message: question,
    });
  }
}

function appendStreamingRow(
  rows: StreamingMessageRow[],
  rowKeys: Set<string>,
  row: StreamingMessageRow,
): void {
  if (rowKeys.has(row.key)) {
    return;
  }

  rows.push(row);
  rowKeys.add(row.key);
}

function mapStreamingOrderToRow(
  orderKey: string,
  options: StreamingOrderLookup,
): StreamingMessageRow | null {
  if (orderKey.startsWith(STREAMING_ASSISTANT_ORDER_PREFIX)) {
    const assistantSegmentId = orderKey.slice(STREAMING_ASSISTANT_ORDER_PREFIX.length);
    const segment = options.assistantSegmentsById.get(assistantSegmentId);
    return segment ? buildStreamingAssistantRow(segment) : null;
  }

  if (orderKey.startsWith(STREAMING_THINKING_ORDER_PREFIX)) {
    const thinkingSegmentId = orderKey.slice(STREAMING_THINKING_ORDER_PREFIX.length);
    const segment = options.thinkingSegmentsById.get(thinkingSegmentId);
    return segment ? buildStreamingThinkingRow(segment) : null;
  }

  if (orderKey.startsWith(STREAMING_TOOL_ORDER_PREFIX)) {
    const toolId = orderKey.slice(STREAMING_TOOL_ORDER_PREFIX.length);
    const tool = options.toolsById.get(toolId);
    return tool ? buildStreamingToolRow(tool) : null;
  }

  if (orderKey.startsWith(STREAMING_QUESTION_ORDER_PREFIX)) {
    const questionId = orderKey.slice(STREAMING_QUESTION_ORDER_PREFIX.length);
    const question = options.questionsById.get(questionId);
    return question
      ? {
        key: question.id,
        message: question,
      }
      : null;
  }

  return null;
}

function buildStreamingAssistantRow(segment: StreamingAssistantSegment): StreamingMessageRow {
  return {
    key: segment.id,
    message: buildAssistantMessage(segment.content, segment.id),
  };
}

function buildStreamingThinkingRow(segment: StreamingThinkingSegment): StreamingMessageRow {
  return {
    key: segment.id,
    message: buildThinkingMessage(segment.content, segment.id),
  };
}

function buildStreamingToolRow(
  tool: MessageListProps['streamingTools'][number],
): StreamingMessageRow {
  return {
    key: tool.id,
    message: buildStreamingToolMessage(tool),
  };
}

function buildStreamingToolMessage(tool: MessageListProps['streamingTools'][number]): ToolChatMessage {
  return {
    id: tool.id,
    kind: 'tool',
    content: tool.content,
    ...(tool.toolInput ? { toolInput: tool.toolInput } : {}),
    toolCallId: tool.toolCallId,
    toolName: tool.toolName,
    toolStatus: tool.toolStatus,
    traceId: tool.traceId,
  };
}
