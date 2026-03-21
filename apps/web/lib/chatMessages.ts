import type {
  AskHumanOption,
  AgentSendAwaitingHumanResponse,
  AgentSendResponse,
  ChatFileAttachment,
  ChatMessage,
  PendingQuestionMessage,
  QuestionChatMessage,
  SessionContentPart,
  SessionHumanInteraction,
  SessionMessage,
  SessionMessageRole,
  SessionToolResult,
  ToolChatMessage,
} from '@/lib/types';

interface NormalizedSessionMessage {
  role: SessionMessageRole | '';
  text: string;
  content?: SessionContentPart[];
  toolCalls?: unknown[];
  toolCallId?: string;
  toolResult?: SessionToolResult;
  humanInteraction?: SessionHumanInteraction;
}

function nextChatMessageID() {
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function normalizeSessionMessage(message: SessionMessage): NormalizedSessionMessage {
  return {
    role: message.role,
    text: message.text ?? '',
    content: message.content ?? undefined,
    toolCalls: message.tool_calls,
    toolCallId: message.tool_call_id,
    toolResult: message.tool_result ?? undefined,
    humanInteraction: message.human_interaction ?? undefined,
  };
}

function attachmentsFromContent(content?: SessionContentPart[]): ChatFileAttachment[] | undefined {
  if (!content || content.length === 0) {
    return undefined;
  }

  const attachments = content
    .filter((part) => part.type === 'file' && part.file)
    .map((part) => ({
      artifactId: part.file!.artifact_id,
      name: part.file!.name,
      downloadUrl: part.file!.download_url,
      mimeType: part.file!.mime_type,
      bytes: part.file!.bytes,
      sha256: part.file!.sha256,
      sourcePath: part.file!.source_path,
      note: part.file!.note,
    }));

  return attachments.length > 0 ? attachments : undefined;
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
  options?: AskHumanOption[]
): QuestionChatMessage {
  return {
    id: nextChatMessageID(),
    kind: 'question',
    content,
    questionId,
    selectionMode,
    options: cloneAskHumanOptions(options),
  };
}

export function buildUserMessage(content: string): ChatMessage {
  return {
    id: nextChatMessageID(),
    kind: 'user',
    content,
  };
}

export function buildAssistantMessage(content: string): ChatMessage {
  return {
    id: nextChatMessageID(),
    kind: 'assistant',
    content,
  };
}

export function buildSystemMessage(content: string): ChatMessage {
  return {
    id: nextChatMessageID(),
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

function mapToolSessionMessage(message: NormalizedSessionMessage): ChatMessage[] {
  if (message.humanInteraction?.prompt) {
    const mapped: ChatMessage[] = [buildQuestionMessage(
      message.humanInteraction.prompt,
      message.humanInteraction.question_id,
      message.humanInteraction.selection_mode,
      message.humanInteraction.options
    )];
    if (message.humanInteraction.answer) {
      mapped.push(buildUserMessage(message.humanInteraction.answer));
    }
    return mapped;
  }

  const toolResult = message.toolResult;
  return [
    buildToolMessage({
      content: toolResult ? formatToolContent(toolResult, message.text) : message.text || '[tool]',
      attachments: attachmentsFromContent(message.content),
      rawOutput: toolResult?.output,
      toolCallId: message.toolCallId,
      toolCalls: message.toolCalls,
      toolName: toolResult?.tool,
      toolStatus: toolResult?.status,
      traceId: toolResult?.trace_id,
    }),
  ];
}

export function mapSessionMessageToChatMessages(message: SessionMessage): ChatMessage[] {
  const normalized = normalizeSessionMessage(message);
  switch (normalized.role) {
    case 'user':
      return [buildUserMessage(normalized.text)];
    case 'assistant':
      return [buildAssistantMessage(normalized.text)];
    case 'system':
      return [buildSystemMessage(normalized.text || '[system]')];
    case 'internal':
      return [buildSystemMessage(normalized.text || '[internal]')];
    case 'tool':
      return mapToolSessionMessage(normalized);
    default:
      return [];
  }
}

export function mapSessionMessagesToChat(messages: SessionMessage[]): ChatMessage[] {
  return messages.flatMap((message) => mapSessionMessageToChatMessages(message));
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
      message.kind === 'pending_question' && message.questionId === questionId
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
    (message) => !(message.kind === 'pending_question' && message.questionId === questionId)
  );
}
