import type { SessionMessage, SessionToolCall } from '@/lib/types';

export function buildSessionToolCallLookup(messages: SessionMessage[]): Map<string, SessionToolCall> {
  const lookup = new Map<string, SessionToolCall>();
  for (const message of messages) {
    for (const toolCall of message.tool_calls || []) {
      const toolCallId = toolCall.id?.trim();
      if (!toolCallId || lookup.has(toolCallId)) {
        continue;
      }
      lookup.set(toolCallId, toolCall);
    }
  }
  return lookup;
}
