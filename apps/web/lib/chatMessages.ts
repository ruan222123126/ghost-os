import type {
  AgentSendAwaitingHumanResponse,
  AgentSendResponse,
  ChatMessage,
  ChatImage,
  ChatSelectedSkill,
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
import { parseAgentMessageWithSelectedSkill } from '@/lib/selectedSkillMessage';
import { nextChatMessageID } from './chatMessageIDs';
import { buildPendingQuestionMessage, buildQuestionMessage } from './chatQuestionMessages';

export { buildPendingQuestionMessage } from './chatQuestionMessages';

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
  selectedSkill?: ChatSelectedSkill;
}

function normalizeSessionMessage(message: SessionMessage): NormalizedSessionMessage {
  const {
    index,
    role,
    text = '',
    thinking = '',
    in_progress: inProgress,
    content,
    tool_calls: toolCalls,
    tool_call_id: toolCallId,
    tool_result: toolResult,
    human_interaction: humanInteraction,
  } = message;

  return {
    index,
    role,
    text,
    thinking,
    inProgress,
    content,
    toolCalls,
    toolCallId,
    toolResult: nullToUndefined(toolResult),
    humanInteraction: nullToUndefined(humanInteraction),
  };
}

function nullToUndefined<T>(value: T | null | undefined): T | undefined {
  return value === null ? undefined : value;
}

function buildSessionMessageID(sessionId: string, messageIndex: number, suffix: string): string {
  return `session:${sessionId}:message:${messageIndex}:${suffix}`;
}

function buildSessionQuestionMessages(sessionId: string, message: NormalizedSessionMessage): ChatMessage[] {
  if (!message.humanInteraction?.prompt) {
    return [];
  }

  const mapped: ChatMessage[] = [buildQuestionMessage({
    content: message.humanInteraction.prompt,
    id: buildSessionMessageID(sessionId, message.index, 'question'),
    questionId: message.humanInteraction.question_id,
    selectionMode: message.humanInteraction.selection_mode,
    options: message.humanInteraction.options,
  })];
  if (message.humanInteraction.answer) {
    mapped.push(buildUserMessage(message.humanInteraction.answer, { id: buildSessionMessageID(sessionId, message.index, 'answer') }));
  }
  return mapped;
}

export function buildUserMessage(content: string, options?: BuildUserMessageOptions): ChatMessage {
  const message: ChatMessage = {
    id: options?.id ?? nextChatMessageID(),
    kind: 'user',
    content,
    images: options?.images,
  };
  return options?.selectedSkill ? { ...message, selectedSkill: options.selectedSkill } : message;
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
      return [buildParsedUserSessionMessage(sessionId, normalized)];
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

function buildParsedUserSessionMessage(
  sessionId: string,
  message: NormalizedSessionMessage,
): ChatMessage {
  const parsed = parseAgentMessageWithSelectedSkill(message.text);
  return buildUserMessage(parsed.message, {
    id: buildSessionMessageID(sessionId, message.index, 'user'),
    images: imagesFromContent(message.content),
    selectedSkill: parsed.selectedSkill,
  });
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

export function hasPendingQuestion(messages: ChatMessage[]): boolean {
  return messages.some((message) => message.kind === 'pending_question');
}

export function replacePendingQuestionWithUserAnswer(messages: ChatMessage[], questionId: string, answer: string): ChatMessage[] {
  return messages.flatMap((message) => message.kind === 'pending_question' && message.questionId === questionId ? [buildUserMessage(answer)] : [message]);
}
