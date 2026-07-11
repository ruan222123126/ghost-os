import type { MessageListRow } from '@/lib/chat-view/types';

interface AssistantCopyPlacementInput {
  currentRow: MessageListRow;
  currentIndex: number;
  rowCount: number;
  getRowAtIndex: (index: number) => MessageListRow;
}

export function shouldPlaceAssistantCopyInline(input: AssistantCopyPlacementInput): boolean {
  const { currentRow, currentIndex, rowCount, getRowAtIndex } = input;

  if (currentRow.kind !== 'message' || currentRow.message.kind !== 'assistant') {
    return false;
  }

  for (let index = currentIndex + 1; index < rowCount; index += 1) {
    const nextRow = getRowAtIndex(index);
    if (nextRow.kind !== 'message') {
      continue;
    }
    if (nextRow.message.kind === 'tool') {
      return true;
    }
    if (nextRow.message.kind !== 'thinking') {
      return false;
    }
  }

  return false;
}
