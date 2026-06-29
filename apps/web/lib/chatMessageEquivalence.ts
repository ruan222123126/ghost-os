import type {
  ChatMessage,
  PendingQuestionMessage,
  QuestionChatMessage,
  ToolChatMessage,
  UserChatMessage,
} from '@/lib/types';

type QuestionLikeMessage = PendingQuestionMessage | QuestionChatMessage;

export function chatMessagesEquivalent(left: ChatMessage, right: ChatMessage): boolean {
  if (left.id === right.id) {
    return true;
  }
  if (isQuestionLike(left) && isQuestionLike(right)) {
    return questionMessagesEquivalent(left, right);
  }
  if (left.kind !== right.kind) {
    return false;
  }

  switch (left.kind) {
    case 'user':
      return userMessagesEquivalent(left, right as UserChatMessage);
    case 'assistant':
    case 'thinking':
    case 'error':
    case 'event':
    case 'system':
      return normalizeText(left.content) === normalizeText(right.content);
    case 'tool':
      return toolMessagesEquivalent(left, right as ToolChatMessage);
    default:
      return false;
  }
}

export function committedMessageCoversDraftMessage(
  committed: ChatMessage,
  draft: ChatMessage,
): boolean {
  if (chatMessagesEquivalent(committed, draft)) {
    return true;
  }
  if (isQuestionLike(committed) && isQuestionLike(draft)) {
    return questionMessagesEquivalent(committed, draft);
  }
  if (committed.kind !== draft.kind) {
    return false;
  }

  switch (committed.kind) {
    case 'assistant':
    case 'thinking':
    case 'error':
    case 'event':
    case 'system':
      return contentCoveredByCommitted(committed.content, draft.content);
    case 'tool':
      return toolMessageCoveredByCommitted(committed, draft as ToolChatMessage);
    default:
      return false;
  }
}

function isQuestionLike(message: ChatMessage): message is QuestionLikeMessage {
  return message.kind === 'question' || message.kind === 'pending_question';
}

function userMessagesEquivalent(left: UserChatMessage, right: UserChatMessage): boolean {
  return normalizeText(left.content) === normalizeText(right.content)
    && countImages(left) === countImages(right)
    && normalizeText(left.selectedSkill?.id) === normalizeText(right.selectedSkill?.id);
}

function questionMessagesEquivalent(left: QuestionLikeMessage, right: QuestionLikeMessage): boolean {
  const leftQuestionId = normalizeText(left.questionId);
  const rightQuestionId = normalizeText(right.questionId);
  if (leftQuestionId && rightQuestionId) {
    return leftQuestionId === rightQuestionId;
  }

  return normalizeText(left.content) === normalizeText(right.content);
}

function toolMessagesEquivalent(left: ToolChatMessage, right: ToolChatMessage): boolean {
  const leftToolCallId = normalizeText(left.toolCallId);
  const rightToolCallId = normalizeText(right.toolCallId);
  if (leftToolCallId && rightToolCallId) {
    return leftToolCallId === rightToolCallId;
  }

  return normalizeText(left.toolName) === normalizeText(right.toolName)
    && normalizeText(left.toolInput) === normalizeText(right.toolInput)
    && normalizeText(left.content) === normalizeText(right.content);
}

function toolMessageCoveredByCommitted(
  committed: ToolChatMessage,
  draft: ToolChatMessage,
): boolean {
  const committedToolCallId = normalizeText(committed.toolCallId);
  const draftToolCallId = normalizeText(draft.toolCallId);
  if (committedToolCallId && draftToolCallId) {
    return committedToolCallId === draftToolCallId;
  }

  if (normalizeText(committed.toolName) !== normalizeText(draft.toolName)) {
    return false;
  }
  if (draft.toolInput && normalizeText(committed.toolInput) !== normalizeText(draft.toolInput)) {
    return false;
  }

  return contentCoveredByCommitted(committed.content, draft.content);
}

function contentCoveredByCommitted(committed: string, draft: string): boolean {
  const normalizedCommitted = normalizeText(committed);
  const normalizedDraft = normalizeText(draft);
  if (!normalizedCommitted || !normalizedDraft) {
    return false;
  }

  return normalizedCommitted === normalizedDraft
    || normalizedCommitted.startsWith(normalizedDraft);
}

function countImages(message: UserChatMessage): number {
  return message.images?.length ?? 0;
}

function normalizeText(value?: string): string {
  return value?.trim() ?? '';
}
