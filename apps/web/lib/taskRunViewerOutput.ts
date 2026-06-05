import { filterCommittedMessagesForDisplay } from '@/components/message/messageVisibility';
import { buildAssistantMessage, buildErrorMessage } from '@/lib/chatMessages';
import { getOrderedStreamingRows, type StreamingMessageRow } from '@/lib/chat-view/streamingRows';
import { createInitialChatState, chatStateReducer } from '@/lib/chat-store/reducer';
import { projectAgentEvent } from '@/lib/chatRuntime/eventProjector';
import {
  createChatRuntimeState,
  createChatRuntimeStateFromDraft,
} from '@/lib/chatRuntime/runtimeState';
import { mapSessionMessagesToChat } from '@/lib/chatMessages';
import type { AgentStreamEvent, ChatMessage, SessionDetail } from '@/lib/types';
import { resolveCardSourceSessionId, type LiveTaskRunCard } from '@/lib/taskRunViewerCards';

const ACTIVE_VIEWER_TOOL_STATUSES = new Set(['pending', 'running', 'in_progress']);

export interface TaskRunCardOutput {
  committedMessages: ChatMessage[];
  streamingRows: StreamingMessageRow[];
}

export function buildTaskRunCardOutput(
  card: LiveTaskRunCard | null,
  session: SessionDetail | null,
): TaskRunCardOutput {
  if (!card) {
    return emptyOutput();
  }
  if (shouldUseSummaryFallback(card, session)) {
    return summaryOutput(card);
  }

  const state = buildSessionOutputState(card, session);
  const pendingQuestions = state.pendingQuestionState.order
    .map((questionId) => state.pendingQuestionState.questionsById[questionId])
    .filter(Boolean)
    .map((question) => ({
      id: question.id,
      kind: 'question' as const,
      content: question.content,
      questionId: question.questionId,
      selectionMode: question.selectionMode,
      options: question.options,
    }));

  return {
    committedMessages: filterCommittedMessagesForDisplay(state.committedMessages, [], false)
      .filter(shouldIncludeViewerMessage)
      .map(normalizeViewerMessage),
    streamingRows: getOrderedStreamingRows({
      pendingQuestions: [],
      streamingAssistantSegments: state.streamingAssistantState.order
        .map((segmentId) => state.streamingAssistantState.segmentsById[segmentId])
        .filter(Boolean),
      streamingThinkingSegments: state.streamingThinkingState.order
        .map((segmentId) => state.streamingThinkingState.segmentsById[segmentId])
        .filter(Boolean),
      streamingItemOrder: state.streamingItemOrder,
      streamingTools: state.streamingToolState.order
        .map((toolId) => state.streamingToolState.toolsById[toolId])
        .filter((tool) => Boolean(tool) && isCompletedViewerToolStatus(tool.toolStatus)),
    }).map((row) => ({
      ...row,
      message: normalizeViewerMessage(row.message),
    })).concat(
      pendingQuestions.map((message) => ({
        key: message.id,
        message,
      })),
    ),
  };
}

function buildSessionOutputState(
  card: LiveTaskRunCard,
  session: SessionDetail | null,
) {
  let state = createInitialChatState();
  if (session) {
    state = {
      ...state,
      committedMessages: mapSessionMessagesToChat(session.id, session.messages),
    };
    state = chatStateReducer(state, {
      type: 'hydrate_turn_draft',
      sessionId: session.id,
      draft: session.turn_draft,
    });
  }

  const runtime = session?.turn_draft
    ? createChatRuntimeStateFromDraft(session.turn_draft, resolveSessionID(card, session))
    : createChatRuntimeState(resolveTraceID(card), resolveSessionID(card, session));
  for (const event of resolveReplayEvents(card, session)) {
    state = chatStateReducer(state, {
      type: 'apply_runtime_actions',
      actions: projectAgentEvent({
        event,
        runtime,
      }),
    });
  }
  return state;
}

function resolveReplayEvents(
  card: LiveTaskRunCard,
  session: SessionDetail | null,
): AgentStreamEvent[] {
  if (!session) {
    return card.source_events;
  }

  const snapshotTime = parseTimestamp(session.updated_at);
  if (snapshotTime === null) {
    return card.source_events;
  }

  return card.source_events.filter((event) => {
    const eventTime = parseTimestamp(event.at);
    return eventTime === null || eventTime > snapshotTime;
  });
}

function parseTimestamp(value?: string): number | null {
  if (!value?.trim()) {
    return null;
  }
  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? null : timestamp;
}

function summaryOutput(card: LiveTaskRunCard): TaskRunCardOutput {
  const messages: ChatMessage[] = [];
  const text = card.final_text?.trim() || card.preview?.trim();
  if (text) {
    messages.push(buildAssistantMessage(text, `run-card:${card.card_id}:summary`));
  }
  if (card.error?.trim()) {
    messages.push(buildErrorMessage(card.error));
  }
  return {
    committedMessages: messages,
    streamingRows: [],
  };
}

function emptyOutput(): TaskRunCardOutput {
  return {
    committedMessages: [],
    streamingRows: [],
  };
}

function normalizeViewerMessage(message: ChatMessage): ChatMessage {
  if (message.kind !== 'pending_question') {
    return message;
  }
  return {
    id: message.id,
    kind: 'question',
    content: message.content,
    questionId: message.questionId,
    selectionMode: message.selectionMode,
    options: message.options,
  };
}

function shouldIncludeViewerMessage(message: ChatMessage): boolean {
  return message.kind !== 'tool' || isCompletedViewerToolStatus(message.toolStatus);
}

function isCompletedViewerToolStatus(status?: string): boolean {
  const normalized = status?.trim().toLowerCase();
  return !normalized || !ACTIVE_VIEWER_TOOL_STATUSES.has(normalized);
}

function resolveSessionID(card: LiveTaskRunCard, session: SessionDetail | null): string {
  return session?.id || resolveCardSourceSessionId(card);
}

function resolveTraceID(card: LiveTaskRunCard): string {
  return card.source_events.at(-1)?.trace_id?.trim() || card.card_id;
}

function shouldUseSummaryFallback(
  card: LiveTaskRunCard,
  session: SessionDetail | null,
): boolean {
  if (card.kind === 'relay_round' && !card.source_events.length && card.status !== 'running') {
    return Boolean(card.final_text?.trim() || card.preview?.trim() || card.error?.trim());
  }
  return !session && card.source_events.length === 0 && Boolean(card.final_text?.trim() || card.preview?.trim() || card.error?.trim());
}
