import type { ChatMessage, PendingQuestionMessage } from '@/lib/types';

export function filterCommittedMessagesForDisplay(
  committedMessages: ChatMessage[],
  pendingQuestions: PendingQuestionMessage[],
  showSystemPromptMessages: boolean,
): ChatMessage[] {
  const pendingQuestionIDs = pendingQuestions.length === 0
    ? null
    : new Set(pendingQuestions.map((question) => question.questionId));

  return committedMessages.filter((message) => {
    if (!showSystemPromptMessages && message.kind === 'system' && message.sourceRole !== 'internal') {
      return false;
    }
    return message.kind !== 'question'
      || !message.questionId
      || pendingQuestionIDs === null
      || !pendingQuestionIDs.has(message.questionId);
  });
}
