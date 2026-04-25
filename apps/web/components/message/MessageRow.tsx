import type { FC } from 'react';
import { QuestionInput } from '@/components/QuestionInput';
import { useWebLocale } from '@/lib/i18n/provider';
import type {
  AssistantChatMessage,
  PendingQuestionMessage,
  ThinkingChatMessage,
  ToolChatMessage,
  UserChatMessage,
} from '@/lib/types';
import { MessageAttachments } from './MessageAttachments';
import { MessageCopyButton } from './MessageCopyButton';
import { MessageImageGallery } from './MessageImageGallery';
import { ToolCard } from './ToolCard';
import { AssistantMarkdownContent } from './AssistantMarkdownContent';
import { ThinkingPanel } from './ThinkingPanel';
import type { MessageRowProps } from './types';

const USER_CHANNEL_LABEL = '// USER_INPUT';
const ASSISTANT_CHANNEL_LABEL = '// SYSTEM_OUTPUT';

const BotGlyph: FC<{ className?: string }> = ({ className }) => (
  <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" className={className}>
    <rect x="4.5" y="5.5" width="11" height="10" rx="2.5" stroke="currentColor" strokeWidth="1.4" />
    <circle cx="8" cy="10.5" r="1" fill="currentColor" />
    <circle cx="12" cy="10.5" r="1" fill="currentColor" />
    <path d="M10 5.5V3.5" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" />
  </svg>
);

const UserMessageRow: FC<{ message: UserChatMessage }> = ({ message }) => (
  <div className="message-row is-user">
    <div className="message-stack">
      <span className="message-channel-label is-user">{USER_CHANNEL_LABEL}</span>
      {message.images?.length ? <MessageImageGallery images={message.images} /> : null}
      {message.content ? (
        <div className="message-bubble is-user">
          <p className="message-bubble-text">{message.content}</p>
          <span className="message-user-corner" aria-hidden="true" />
        </div>
      ) : null}
    </div>
  </div>
);

const AssistantMessageRow: FC<{ message: AssistantChatMessage }> = ({ message }) => {
  const { copy } = useWebLocale();

  return (
    <div className="message-row is-assistant">
      <div className="message-stack">
        <span className="message-channel-label is-assistant">
          <BotGlyph className="message-channel-icon" />
          {ASSISTANT_CHANNEL_LABEL}
        </span>
        {message.inProgress ? <div className="message-draft-flag">{copy.chat.assistantDraftFlag}</div> : null}
        <div className="message-assistant-body">
          <AssistantMarkdownContent content={message.content} />
        </div>
        {message.content ? (
          <div className="message-actions">
            <MessageCopyButton text={message.content} />
          </div>
        ) : null}
      </div>
    </div>
  );
};

const ToolMessageRow: FC<{
  isOpen: boolean;
  message: ToolChatMessage;
  onToggle: () => void;
}> = ({ isOpen, message, onToggle }) => (
  <div className="message-row is-tool">
    <div className="message-stack">
      <ToolCard isOpen={isOpen} onToggle={onToggle} tool={message} />
      {message.images?.length ? <MessageImageGallery images={message.images} /> : null}
      {message.attachments?.length ? <MessageAttachments attachments={message.attachments} /> : null}
    </div>
  </div>
);

const ThinkingMessageRow: FC<{
  isOpen: boolean;
  message: ThinkingChatMessage;
  onToggle: () => void;
}> = ({ isOpen, message, onToggle }) => (
  <div className="message-row is-thinking">
    <div className="message-stack">
      <ThinkingPanel expanded={isOpen} text={message.content} onToggleExpanded={onToggle} />
    </div>
  </div>
);

const NoteMessageRow: FC<{ content: string; tone: 'system' | 'error' }> = ({ content, tone }) => (
  <div className={`message-row is-${tone}`}>
    <div className={`message-note is-${tone}`}>{content}</div>
  </div>
);

const QuestionMessageRow: FC<{
  loading: boolean;
  message: PendingQuestionMessage;
  onAnswerQuestion: MessageRowProps['onAnswerQuestion'];
  onCancelQuestion: MessageRowProps['onCancelQuestion'];
}> = ({ loading, message, onAnswerQuestion, onCancelQuestion }) => (
  <div className="message-row is-question">
    <div className="message-stack">
      <div className="message-note is-question">{message.content}</div>
      <QuestionInput
        loading={loading}
        selectionMode={message.selectionMode}
        options={message.options}
        onAnswer={(answer) => onAnswerQuestion(message.questionId, answer)}
        onCancel={() => onCancelQuestion(message.questionId)}
      />
    </div>
  </div>
);

export const MessageRow: FC<MessageRowProps> = ({
  message,
  isToolCardOpen = false,
  isThinkingPanelOpen = false,
  loading,
  onAnswerQuestion,
  onCancelQuestion,
  onToggleThinkingPanel,
  onToggleToolCard,
}) => {
  switch (message.kind) {
    case 'user':
      return <UserMessageRow message={message} />;
    case 'assistant':
      return <AssistantMessageRow message={message} />;
    case 'tool':
      return (
        <ToolMessageRow
          isOpen={isToolCardOpen}
          message={message}
          onToggle={() => onToggleToolCard?.(message.id)}
        />
      );
    case 'thinking':
      return (
        <ThinkingMessageRow
          isOpen={isThinkingPanelOpen}
          message={message}
          onToggle={() => onToggleThinkingPanel?.(message.id)}
        />
      );
    case 'system':
      return <NoteMessageRow content={message.content} tone="system" />;
    case 'error':
      return <NoteMessageRow content={message.content} tone="error" />;
    case 'pending_question':
      return (
        <QuestionMessageRow
          loading={loading}
          message={message}
          onAnswerQuestion={onAnswerQuestion}
          onCancelQuestion={onCancelQuestion}
        />
      );
    case 'question':
      return (
        <div className="message-row is-question">
          <div className="message-stack">
            <div className="message-note is-question">{message.content}</div>
          </div>
        </div>
      );
    default:
      return null;
  }
};
