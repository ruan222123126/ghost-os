import { buildAssistantMessage, buildErrorMessage } from '@/lib/chatMessages';
import { buildChatViewProjection } from '@/lib/chat-view/messageRows';
import type { StreamingMessageRow } from '@/lib/chat-view/types';
import { createInitialChatState, chatStateReducer } from '@/lib/chat-store/reducer';
import { projectAgentEvent } from '@/lib/chatRuntime/eventProjector';
import {
  createChatRuntimeState,
  createChatRuntimeStateFromDraft,
} from '@/lib/chatRuntime/runtimeState';
import { mapSessionMessagesToChat } from '@/lib/chatMessages';
import type { ChatMessage, SessionDetail, SessionMessage } from '@/lib/types';
import { isSummaryOnlyCard, resolveCardSourceSessionId, type LiveTaskRunCard } from '@/lib/taskRunViewerCards';

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

  const state = buildSessionOutputState(card, sessionForCardOutput(card, session));
  const pendingQuestions = shouldShowPendingQuestions(card)
    ? state.pendingQuestionState.order
      .map((questionId) => state.pendingQuestionState.questionsById[questionId])
      .filter(Boolean)
      .map((question) => ({
        id: question.id,
        kind: 'question' as const,
        content: question.content,
        questionId: question.questionId,
        selectionMode: question.selectionMode,
        options: question.options,
      }))
    : [];
  const includeActiveStreamingContent = !isTerminalCard(card);

  const projection = buildChatViewProjection({
    committedMessages: state.committedMessages,
    loading: false,
    pendingQuestions: [],
    showSystemPromptMessages: false,
    streamingAssistantSegments: includeActiveStreamingContent
      ? state.streamingAssistantState.order
        .map((segmentId) => state.streamingAssistantState.segmentsById[segmentId])
        .filter(Boolean)
      : [],
    streamingThinkingSegments: includeActiveStreamingContent
      ? state.streamingThinkingState.order
        .map((segmentId) => state.streamingThinkingState.segmentsById[segmentId])
        .filter(Boolean)
      : [],
    activeStreamingThinkingId: includeActiveStreamingContent
      ? state.streamingThinkingState.activeSegmentId || null
      : null,
    streamingItemOrder: state.streamingItemOrder,
    streamingTools: state.streamingToolState.order
      .map((toolId) => state.streamingToolState.toolsById[toolId])
      .filter((tool) => Boolean(tool) && isCompletedViewerToolStatus(tool.toolStatus)),
  });

  return {
    committedMessages: projection.visibleCommittedMessages
      .filter(shouldIncludeViewerMessage)
      .map(normalizeViewerMessage),
    streamingRows: projection.streamingRows.map((row) => ({
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

function sessionForCardOutput(
  card: LiveTaskRunCard,
  session: SessionDetail | null,
): SessionDetail | null {
  if (card.source_events.length > 0) {
    return null;
  }
  if (!session) {
    return null;
  }
  const messages = sessionMessagesForCard(card, session.messages);
  if (messages.length === 0) {
    return null;
  }
  if (messages === session.messages) {
    return session;
  }
  return {
    ...session,
    messages,
    turn_draft: null,
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
  for (const event of card.source_events) {
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

function sessionMessagesForCard(
  card: LiveTaskRunCard,
  messages: SessionMessage[],
): SessionMessage[] {
  if (isPositiveInteger(card.round)) {
    const roundMessages = sessionMessagesForUserTurn(messages, card.round);
    if (roundMessages.length > 0) {
      return roundMessages;
    }
  }
  if (hasMultipleUserTurns(messages)) {
    return [];
  }
  return messages;
}

function sessionMessagesForUserTurn(
  messages: SessionMessage[],
  targetTurn: number,
): SessionMessage[] {
  let currentTurn = 0;
  const selected: SessionMessage[] = [];
  for (const message of messages) {
    if (message.role === 'user') {
      currentTurn += 1;
    }
    if (currentTurn === targetTurn) {
      selected.push(message);
      continue;
    }
    if (currentTurn > targetTurn) {
      break;
    }
  }
  return selected;
}

function hasMultipleUserTurns(messages: SessionMessage[]): boolean {
  let count = 0;
  for (const message of messages) {
    if (message.role !== 'user') {
      continue;
    }
    count += 1;
    if (count > 1) {
      return true;
    }
  }
  return false;
}

function isPositiveInteger(value: unknown): value is number {
  return Number.isInteger(value) && Number(value) > 0;
}

function shouldUseSummaryFallback(
  card: LiveTaskRunCard,
  session: SessionDetail | null,
): boolean {
  if (isSummaryOnlyCard(card)) {
    return true;
  }
  if (card.source_events.length > 0) {
    return false;
  }
  const hasCardSummary = Boolean(card.final_text?.trim() || card.preview?.trim() || card.error?.trim());
  if (!hasCardSummary) {
    return false;
  }
  if (card.status !== 'running') {
    return true;
  }
  return !session;
}

function isTerminalCard(card: LiveTaskRunCard): boolean {
  const status = card.status?.trim();
  return Boolean(status && status !== 'running');
}

function shouldShowPendingQuestions(card: LiveTaskRunCard): boolean {
  return card.status?.trim() === 'awaiting_human';
}
