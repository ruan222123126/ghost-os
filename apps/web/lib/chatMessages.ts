import type {
  AskHumanOption,
  AgentSendAwaitingHumanResponse,
  AgentSendResponse,
  ChatMessage,
  ChatImage,
  PendingQuestionMessage,
  QuestionChatMessage,
  SessionContentPart,
  SessionHumanInteraction,
  SessionMessage,
  SessionMessageRole,
  SessionToolCall,
  SessionToolResult,
  SystemChatMessageSourceRole,
  ToolChatMessage,
} from '@/lib/types';
import { imagesFromContent } from './chatMessageMedia';
import { buildSessionToolCallLookup } from './chatToolCalls';
import { buildToolChatMessage, resolveSessionToolCall } from './chatToolMessages';
import { filterToolTagResultToLoadedTools } from '@/lib/toolTagResultText';
import { stripToolTagCalls } from '@/lib/toolTagText';

const TASK_RUN_EVENT_MARKER = '[TASK_RUN_EVENT]';

interface NormalizedSessionMessage {
  index: number;
  role: SessionMessageRole | '';
  text: string;
  thinking: string;
  inProgress?: boolean;
  content?: SessionContentPart[];
  toolCalls?: unknown[];
  toolCallId?: string;
  toolResult?: SessionToolResult;
  humanInteraction?: SessionHumanInteraction;
}

interface BuildUserMessageOptions {
  id?: string;
  images?: ChatImage[];
}

function nextChatMessageID() {
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function normalizeSessionMessage(message: SessionMessage): NormalizedSessionMessage {
  return {
    index: message.index,
    role: message.role,
    text: message.text ?? '',
    thinking: message.thinking ?? '',
    inProgress: message.in_progress,
    content: message.content ?? undefined,
    toolCalls: message.tool_calls,
    toolCallId: message.tool_call_id,
    toolResult: message.tool_result ?? undefined,
    humanInteraction: message.human_interaction ?? undefined,
  };
}

function buildSessionMessageID(sessionId: string, messageIndex: number, suffix: string): string {
  return `session:${sessionId}:message:${messageIndex}:${suffix}`;
}

function cloneAskHumanOptions(options?: AskHumanOption[]): AskHumanOption[] | undefined {
  if (!options || options.length === 0) {
    return undefined;
  }

  const normalized = options
    .map((option) => ({
      label: option.label.trim(),
      allow_custom: option.allow_custom,
    }))
    .filter((option) => option.label.length > 0);

  return normalized.length > 0 ? normalized : undefined;
}

function buildQuestionMessage(
  content: string,
  questionId?: string,
  selectionMode?: 'single' | 'multiple',
  options?: AskHumanOption[],
  id?: string,
): QuestionChatMessage {
  return {
    id: id ?? nextChatMessageID(),
    kind: 'question',
    content,
    questionId,
    selectionMode,
    options: cloneAskHumanOptions(options),
  };
}

function buildSessionQuestionMessages(sessionId: string, message: NormalizedSessionMessage): ChatMessage[] {
  if (!message.humanInteraction?.prompt) {
    return [];
  }

  const mapped: ChatMessage[] = [buildQuestionMessage(message.humanInteraction.prompt, message.humanInteraction.question_id, message.humanInteraction.selection_mode, message.humanInteraction.options, buildSessionMessageID(sessionId, message.index, 'question'))];
  if (message.humanInteraction.answer) {
    mapped.push(buildUserMessage(message.humanInteraction.answer, { id: buildSessionMessageID(sessionId, message.index, 'answer') }));
  }
  return mapped;
}

export function buildUserMessage(content: string, options?: BuildUserMessageOptions): ChatMessage {
  return {
    id: options?.id ?? nextChatMessageID(),
    kind: 'user',
    content,
    images: options?.images,
  };
}

export function buildAssistantMessage(content: string, id?: string, inProgress?: boolean): ChatMessage {
  return {
    id: id ?? nextChatMessageID(),
    kind: 'assistant',
    content,
    inProgress,
  };
}

export function buildThinkingMessage(content: string, id?: string, inProgress?: boolean): ChatMessage {
  return {
    id: id ?? nextChatMessageID(),
    kind: 'thinking',
    content,
    ...(inProgress ? { inProgress } : {}),
  };
}

export function buildEventMessage(content: string, id?: string): ChatMessage {
  return {
    id: id ?? nextChatMessageID(),
    kind: 'event',
    content,
  };
}

export function buildSystemMessage(
  content: string,
  id?: string,
  sourceRole: SystemChatMessageSourceRole = 'system',
): ChatMessage {
  return {
    id: id ?? nextChatMessageID(),
    kind: 'system',
    content,
    sourceRole,
  };
}

export function buildToolMessage(message: Omit<ToolChatMessage, 'id' | 'kind'>): ToolChatMessage {
  return {
    id: nextChatMessageID(),
    kind: 'tool',
    ...message,
  };
}

export function buildErrorMessage(messageText: string): ChatMessage {
  return {
    id: nextChatMessageID(),
    kind: 'error',
    content: messageText,
    error: messageText,
  };
}

export function isAwaitingHumanResponse(response: AgentSendResponse): response is AgentSendAwaitingHumanResponse {
  return 'status' in response && response.status === 'awaiting_human';
}

export function buildPendingQuestionMessage(response: AgentSendAwaitingHumanResponse): PendingQuestionMessage {
  return {
    id: nextChatMessageID(),
    kind: 'pending_question',
    content: response.prompt,
    questionId: response.question_id,
    sessionId: response.session_id,
    selectionMode: response.selection_mode,
    options: cloneAskHumanOptions(response.options),
  };
}

function mapToolSessionMessage(
  sessionId: string,
  message: NormalizedSessionMessage,
  toolCallLookup?: Map<string, SessionToolCall>,
): ChatMessage[] {
  if (message.humanInteraction?.prompt) {
    return buildSessionQuestionMessages(sessionId, message);
  }

  const toolResult = message.toolResult;
  const matchedToolCall = resolveSessionToolCall(message.toolCallId, toolCallLookup);
  return [buildToolChatMessage({ content: message.content, index: message.index, sessionId, text: message.text, toolCallId: message.toolCallId, toolCall: matchedToolCall, toolResult })];
}

function buildSessionAssistantMessages(
  sessionId: string,
  message: NormalizedSessionMessage,
): ChatMessage[] {
  const mapped: ChatMessage[] = [];
  if (message.thinking.trim()) {
    mapped.push(buildThinkingMessage(message.thinking, buildSessionMessageID(sessionId, message.index, 'thinking')));
  }

  const visibleText = stripToolTagCalls(message.text);
  if (!visibleText.trim()) {
    return mapped;
  }

  mapped.push(buildAssistantMessage(visibleText, buildSessionMessageID(sessionId, message.index, 'assistant'), message.inProgress));
  return mapped;
}

function buildSessionInternalMessages(sessionId: string, message: NormalizedSessionMessage): ChatMessage[] {
  const eventText = extractTaskRunEventText(message.text);
  if (eventText !== null) {
    return [buildEventMessage(eventText || '任务运行事件', buildSessionMessageID(sessionId, message.index, 'event'))];
  }

  return [
    buildSystemMessage(
      filterToolTagResultToLoadedTools(message.text) || '[internal]',
      buildSessionMessageID(sessionId, message.index, 'internal'),
      'internal',
    ),
  ];
}

function extractTaskRunEventText(text: string): string | null {
  const trimmed = text.trim();
  if (!trimmed.startsWith(TASK_RUN_EVENT_MARKER)) {
    return null;
  }
  return trimmed.slice(TASK_RUN_EVENT_MARKER.length).trim();
}

export function mapSessionMessageToChatMessages(
  sessionId: string,
  message: SessionMessage,
  toolCallLookup?: Map<string, SessionToolCall>,
): ChatMessage[] {
  const normalized = normalizeSessionMessage(message);
  switch (normalized.role) {
    case 'user':
      return [buildUserMessage(normalized.text, { id: buildSessionMessageID(sessionId, normalized.index, 'user'), images: imagesFromContent(normalized.content) })];
    case 'assistant':
      return buildSessionAssistantMessages(sessionId, normalized);
    case 'system':
      return [buildSystemMessage(normalized.text || '[system]', buildSessionMessageID(sessionId, normalized.index, 'system'), 'system')];
    case 'internal':
      return buildSessionInternalMessages(sessionId, normalized);
    case 'tool':
      return mapToolSessionMessage(sessionId, normalized, toolCallLookup);
    default:
      return [];
  }
}

export function mapSessionMessagesToChat(sessionId: string, messages: SessionMessage[]): ChatMessage[] {
  const mapped: ChatMessage[] = [];
  const toolCallLookup = buildSessionToolCallLookup(messages);

  for (const message of messages) {
    mapped.push(...mapSessionMessageToChatMessages(sessionId, message, toolCallLookup));
  }

  return mapped;
}

export function mapAgentReplyToChatMessages(reply: AgentSendResponse): ChatMessage[] {
  if (isAwaitingHumanResponse(reply)) {
    return [buildPendingQuestionMessage(reply)];
  }
  return [buildAssistantMessage(reply.message)];
}

export function findPendingQuestion(messages: ChatMessage[], questionId: string): PendingQuestionMessage | undefined {
  return messages.find((message): message is PendingQuestionMessage => message.kind === 'pending_question' && message.questionId === questionId);
}

export function hasPendingQuestion(messages: ChatMessage[]): boolean {
  return messages.some((message) => message.kind === 'pending_question');
}

export function replacePendingQuestionWithUserAnswer(messages: ChatMessage[], questionId: string, answer: string): ChatMessage[] {
  return messages.flatMap((message) => message.kind === 'pending_question' && message.questionId === questionId ? [buildUserMessage(answer)] : [message]);
}

export function removePendingQuestion(messages: ChatMessage[], questionId: string): ChatMessage[] {
  return messages.filter((message) => !(message.kind === 'pending_question' && message.questionId === questionId));
}
