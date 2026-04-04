import type { FC } from 'react';
import { QuestionInput } from '@/components/QuestionInput';
import type { PendingQuestionMessage, ToolChatMessage, UserChatMessage } from '@/lib/types';
import { MessageAttachments } from './MessageAttachments';
import { MessageCopyButton } from './MessageCopyButton';
import { MessageImageGallery } from './MessageImageGallery';
import { ToolCard } from './ToolCard';
import type { MessageRowProps } from './types';

const UserMessageRow: FC<{ message: UserChatMessage }> = ({ message }) => (
  <div className="message-row is-user">
    <div className="message-stack">
      {message.images?.length ? <MessageImageGallery images={message.images} /> : null}
      {message.content ? <div className="message-bubble is-user">{message.content}</div> : null}
    </div>
  </div>
);

const AssistantMessageRow: FC<{ content: string }> = ({ content }) => (
  <div className="message-row is-assistant">
    <div className="message-stack">
      <div className="message-content">{content}</div>
      {content ? (
        <div className="message-actions">
          <MessageCopyButton text={content} />
        </div>
      ) : null}
    </div>
  </div>
);

const ToolMessageRow: FC<{
  isOpen: boolean;
  message: ToolChatMessage;
  onToggle: () => void;
}> = ({ isOpen, message, onToggle }) => (
  <div className="message-row is-tool">
    <div className="message-stack">
      <ToolCard isOpen={isOpen} onToggle={onToggle} tool={message} />
      {message.attachments?.length ? <MessageAttachments attachments={message.attachments} /> : null}
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
  loading,
  onAnswerQuestion,
  onCancelQuestion,
  onToggleToolCard,
}) => {
  switch (message.kind) {
    case 'user':
      return <UserMessageRow message={message} />;
    case 'assistant':
      return <AssistantMessageRow content={message.content} />;
    case 'tool':
      return (
        <ToolMessageRow
          isOpen={isToolCardOpen}
          message={message}
          onToggle={() => onToggleToolCard?.(message.id)}
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
