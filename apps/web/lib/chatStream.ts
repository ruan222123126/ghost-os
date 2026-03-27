import type { AskHumanOption, ChatMessage } from '@/lib/types';
import { isGraphQLToolDocument } from '@/lib/graphqlToolText';

export interface AssistantMessageUpdate {
  messageId: string;
  text: string;
}

export interface ToolMessageUpdate {
  messageId: string;
  content: string;
  rawOutput?: string;
  toolCallId?: string;
  toolName?: string;
  toolStatus?: string;
  traceId?: string;
}

export interface PendingQuestionUpdate {
  messageId: string;
  content: string;
  questionId: string;
  sessionId: string;
  selectionMode?: 'single' | 'multiple';
  options?: AskHumanOption[];
}

export function appendAssistantText(
  messages: ChatMessage[],
  update: AssistantMessageUpdate,
): ChatMessage[] {
  if (!update.text) {
    return messages;
  }

  const index = findMessageIndex(messages, update.messageId, 'assistant');
  if (index < 0) {
    return [...messages, { id: update.messageId, kind: 'assistant', content: update.text }];
  }

  const current = messages[index];
  if (current.kind !== 'assistant') {
    return messages;
  }

  return replaceMessage(messages, index, {
    ...current,
    content: `${current.content}${update.text}`,
  });
}

export function replaceAssistantText(
  messages: ChatMessage[],
  update: AssistantMessageUpdate,
): ChatMessage[] {
  if (!update.text) {
    return messages;
  }

  const index = findMessageIndex(messages, update.messageId, 'assistant');
  if (index < 0) {
    return [...messages, { id: update.messageId, kind: 'assistant', content: update.text }];
  }

  const current = messages[index];
  if (current.kind !== 'assistant') {
    return messages;
  }

  return replaceMessage(messages, index, {
    ...current,
    content: update.text,
  });
}

export function suppressGraphQLAssistantToolText(messages: ChatMessage[], messageId: string): ChatMessage[] {
  const index = findMessageIndex(messages, messageId, 'assistant');
  if (index < 0) {
    return messages;
  }

  const current = messages[index];
  if (current.kind !== 'assistant' || !isGraphQLToolDocument(current.content)) {
    return messages;
  }

  return messages.filter((_, currentIndex) => currentIndex !== index);
}

export function upsertToolMessage(
  messages: ChatMessage[],
  update: ToolMessageUpdate,
): ChatMessage[] {
  const index = findMessageIndex(messages, update.messageId, 'tool');
  if (index < 0) {
    return [...messages, {
      id: update.messageId,
      kind: 'tool',
      content: update.content,
      rawOutput: update.rawOutput,
      toolCallId: update.toolCallId,
      toolName: update.toolName,
      toolStatus: update.toolStatus,
      traceId: update.traceId,
    }];
  }

  const current = messages[index];
  if (current.kind !== 'tool') {
    return messages;
  }

  return replaceMessage(messages, index, {
    ...current,
    content: update.content,
    rawOutput: update.rawOutput ?? current.rawOutput,
    toolCallId: update.toolCallId ?? current.toolCallId,
    toolName: update.toolName ?? current.toolName,
    toolStatus: update.toolStatus ?? current.toolStatus,
    traceId: update.traceId ?? current.traceId,
  });
}

export function upsertPendingQuestion(
  messages: ChatMessage[],
  update: PendingQuestionUpdate,
): ChatMessage[] {
  const index = findMessageIndex(messages, update.messageId, 'pending_question');
  if (index < 0) {
    return [...messages, {
      id: update.messageId,
      kind: 'pending_question',
      content: update.content,
      questionId: update.questionId,
      sessionId: update.sessionId,
      selectionMode: update.selectionMode,
      options: update.options,
    }];
  }

  const current = messages[index];
  if (current.kind !== 'pending_question') {
    return messages;
  }

  return replaceMessage(messages, index, {
    ...current,
    content: update.content,
    questionId: update.questionId,
    sessionId: update.sessionId,
    selectionMode: update.selectionMode,
    options: update.options,
  });
}

function findMessageIndex(
  messages: ChatMessage[],
  messageId: string,
  kind: ChatMessage['kind'],
): number {
  return messages.findIndex((message) => message.id === messageId && message.kind === kind);
}

function replaceMessage(
  messages: ChatMessage[],
  index: number,
  nextMessage: ChatMessage,
): ChatMessage[] {
  return messages.map((message, currentIndex) => (
    currentIndex === index ? nextMessage : message
  ));
}
