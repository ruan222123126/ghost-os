import type { ChatMessage } from '@/lib/types';
import type { TaskRunCardOutput } from '@/lib/taskRunViewerOutput';

export function buildLiveRunViewerOutputSignature(
  selectedCardId: string,
  output: TaskRunCardOutput,
): string {
  return [
    selectedCardId,
    output.committedMessages.map(messageSignature).join(','),
    output.streamingRows.map((row) => `${row.key}:${messageSignature(row.message)}`).join(','),
  ].join('|');
}

function messageSignature(message: ChatMessage): string {
  return [
    message.id,
    message.kind,
    message.content.length,
    message.kind === 'tool' ? message.toolStatus ?? '' : '',
  ].join(':');
}
