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
  SessionToolResult,
  ToolChatMessage,
} from '@/lib/types';
import { attachmentsFromContent, imagesFromContent } from './chatMessageMedia';
import { filterToolTagResultToLoadedTools } from '@/lib/toolTagResultText';
import { stripToolTagCalls } from '@/lib/toolTagText';

interface NormalizedSessionMessage {
  index: number;
  role: SessionMessageRole | '';
  text: string;
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

function formatToolContent(toolResult: SessionToolResult, fallbackText: string): string {
  if (toolResult.error) {
    return toolResult.error;
  }
  if (toolResult.output) {
    return toolResult.output;
  }
  if (toolResult.tool) {
    return `[${toolResult.tool}]`;
  }
  return fallbackText || '[tool]';
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

  const mapped: ChatMessage[] = [buildQuestionMessage(
    message.humanInteraction.prompt,
    message.humanInteraction.question_id,
    message.humanInteraction.selection_mode,
    message.humanInteraction.options,
    buildSessionMessageID(sessionId, message.index, 'question'),
  )];
  if (message.humanInteraction.answer) {
    mapped.push(buildUserMessage(message.humanInteraction.answer, {
      id: buildSessionMessageID(sessionId, message.index, 'answer'),
    }));
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

export function buildSystemMessage(content: string, id?: string): ChatMessage {
  return {
    id: id ?? nextChatMessageID(),
    kind: 'system',
    content,
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

function mapToolSessionMessage(sessionId: string, message: NormalizedSessionMessage): ChatMessage[] {
  if (message.humanInteraction?.prompt) {
    return buildSessionQuestionMessages(sessionId, message);
  }

  const toolResult = message.toolResult;
  return [{
    id: buildSessionMessageID(sessionId, message.index, 'tool'),
    kind: 'tool',
    content: toolResult ? formatToolContent(toolResult, message.text) : message.text || '[tool]',
    attachments: attachmentsFromContent(message.content),
    rawOutput: toolResult?.output,
    toolCallId: message.toolCallId,
    toolCalls: message.toolCalls,
    toolName: toolResult?.tool,
    toolStatus: toolResult?.status,
    traceId: toolResult?.trace_id,
  }];
}

export function mapSessionMessageToChatMessages(sessionId: string, message: SessionMessage): ChatMessage[] {
  const normalized = normalizeSessionMessage(message);
  switch (normalized.role) {
    case 'user':
      return [buildUserMessage(normalized.text, {
        id: buildSessionMessageID(sessionId, normalized.index, 'user'),
        images: imagesFromContent(normalized.content),
      })];
    case 'assistant':
      const visibleText = stripToolTagCalls(normalized.text);
      if (!visibleText.trim()) {
        return [];
      }
      return [buildAssistantMessage(
        visibleText,
        buildSessionMessageID(sessionId, normalized.index, 'assistant'),
        normalized.inProgress,
      )];
    case 'system':
      return [buildSystemMessage(
        normalized.text || '[system]',
        buildSessionMessageID(sessionId, normalized.index, 'system'),
      )];
    case 'internal':
      return [buildSystemMessage(
        filterToolTagResultToLoadedTools(normalized.text) || '[internal]',
        buildSessionMessageID(sessionId, normalized.index, 'internal'),
      )];
    case 'tool':
      return mapToolSessionMessage(sessionId, normalized);
    default:
      return [];
  }
}

export function mapSessionMessagesToChat(sessionId: string, messages: SessionMessage[]): ChatMessage[] {
  const mapped: ChatMessage[] = [];

  for (const message of messages) {
    mapped.push(...mapSessionMessageToChatMessages(sessionId, message));
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
  return messages.find(
    (message): message is PendingQuestionMessage =>
      message.kind === 'pending_question' && message.questionId === questionId,
  );
}

export function hasPendingQuestion(messages: ChatMessage[]): boolean {
  return messages.some((message) => message.kind === 'pending_question');
}

export function replacePendingQuestionWithUserAnswer(messages: ChatMessage[], questionId: string, answer: string): ChatMessage[] {
  return messages.flatMap((message) => {
    if (message.kind === 'pending_question' && message.questionId === questionId) {
      return [buildUserMessage(answer)];
    }
    return [message];
  });
}

export function removePendingQuestion(messages: ChatMessage[], questionId: string): ChatMessage[] {
  return messages.filter(
    (message) => !(message.kind === 'pending_question' && message.questionId === questionId),
  );
}
