import type {
  AgentSendAwaitingHumanResponse,
  AskHumanOption,
  PendingQuestionMessage,
  QuestionChatMessage,
} from '@/lib/types';
import { nextChatMessageID } from './chatMessageIDs';

export interface BuildQuestionMessageOptions {
  content: string;
  id?: string;
  questionId?: string;
  selectionMode?: 'single' | 'multiple';
  options?: AskHumanOption[];
}

export function buildQuestionMessage(options: BuildQuestionMessageOptions): QuestionChatMessage {
  return {
    id: options.id ?? nextChatMessageID(),
    kind: 'question',
    content: options.content,
    questionId: options.questionId,
    selectionMode: options.selectionMode,
    options: cloneAskHumanOptions(options.options),
  };
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
