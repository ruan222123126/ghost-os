import type {
  SessionContentPart,
  SessionToolCall,
  SessionToolResult,
  ToolChatMessage,
} from '@/lib/types';
import { attachmentsFromContent, imagesFromContent } from './chatMessageMedia';

interface BuildToolChatMessageInput {
  content?: SessionContentPart[];
  index: number;
  sessionId: string;
  text: string;
  toolCallId?: string;
  toolCall?: SessionToolCall;
  toolResult?: SessionToolResult;
}

export function buildToolChatMessage(input: BuildToolChatMessageInput): ToolChatMessage {
  const { content, index, sessionId, text, toolCallId, toolCall, toolResult } = input;
  return {
    id: `session:${sessionId}:message:${index}:tool`,
    kind: 'tool',
    content: toolResult ? formatToolContent(toolResult, text) : text || '[tool]',
    images: imagesFromContent(content),
    attachments: attachmentsFromContent(content),
    rawOutput: toolResult?.output,
    toolCallId,
    toolCalls: toolCall ? [toolCall] : undefined,
    toolName: toolResult?.tool,
    toolStatus: toolResult?.status,
    traceId: toolResult?.trace_id,
  };
}

export function resolveSessionToolCall(
  toolCallId: string | undefined,
  toolCallLookup?: Map<string, SessionToolCall>,
): SessionToolCall | undefined {
  const normalizedToolCallId = toolCallId?.trim();
  if (!normalizedToolCallId || !toolCallLookup) {
    return undefined;
  }
  return toolCallLookup.get(normalizedToolCallId);
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
