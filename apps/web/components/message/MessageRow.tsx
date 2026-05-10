import type { FC, RefObject } from 'react';
import { useCallback, useEffect, useRef, useState } from 'react';
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

const USER_MESSAGE_COLLAPSED_LINES = 3;
const USER_MESSAGE_HEIGHT_EPSILON = 1;
const USER_MESSAGE_EXPAND_LABEL = '展开用户消息';
const USER_MESSAGE_COLLAPSE_LABEL = '收起用户消息';

const UserMessageRow: FC<{ message: UserChatMessage }> = ({ message }) => {
  const contentRef = useRef<HTMLDivElement | null>(null);
  const { collapsed, expanded, overflowing, setExpanded } = useUserMessageOverflow(message, contentRef);
  const contentClassName = [
    'message-content',
    'message-user-content',
    collapsed ? 'is-collapsed' : '',
  ]
    .filter(Boolean)
    .join(' ');
  const backplateClassName = [
    'message-user-backplate',
    overflowing ? 'has-toggle' : '',
  ]
    .filter(Boolean)
    .join(' ');
  const iconClassName = [
    'message-user-expand-icon',
    expanded ? 'is-expanded' : '',
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <div className="message-row is-user">
      <div className="message-stack">
        {message.images?.length ? <MessageImageGallery images={message.images} /> : null}
        {message.content ? (
          <div className={backplateClassName}>
            <div ref={contentRef} className={contentClassName}>{message.content}</div>
            {overflowing ? (
              <button
                type="button"
                className="message-user-expand-toggle"
                aria-expanded={expanded}
                aria-label={expanded ? USER_MESSAGE_COLLAPSE_LABEL : USER_MESSAGE_EXPAND_LABEL}
                onClick={() => setExpanded((value) => !value)}
              >
                <span className={iconClassName} aria-hidden="true" />
              </button>
            ) : null}
          </div>
        ) : null}
      </div>
    </div>
  );
};

function useUserMessageOverflow(
  message: UserChatMessage,
  contentRef: RefObject<HTMLDivElement>,
) {
  const [expanded, setExpanded] = useState(false);
  const [overflowing, setOverflowing] = useState(false);
  const collapsed = overflowing && !expanded;
  const measureOverflow = useCallback(() => {
    const content = contentRef.current;
    if (!content) {
      return;
    }
    const lineHeight = Number.parseFloat(window.getComputedStyle(content).lineHeight);
    const collapsedHeight = lineHeight * USER_MESSAGE_COLLAPSED_LINES;
    setOverflowing(content.scrollHeight > collapsedHeight + USER_MESSAGE_HEIGHT_EPSILON);
  }, [contentRef]);

  useEffect(() => {
    setExpanded(false);
    measureOverflow();
    window.addEventListener('resize', measureOverflow);
    return () => window.removeEventListener('resize', measureOverflow);
  }, [measureOverflow, message.content, message.id]);

  return { collapsed, expanded, overflowing, setExpanded };
}

const AssistantMessageRow: FC<{
  message: AssistantChatMessage;
  assistantMarkdownEnabled: boolean;
  hasTrailingTool?: boolean;
}> = ({ message, assistantMarkdownEnabled, hasTrailingTool = false }) => {
  const { copy } = useWebLocale();
  const assistantFrameClassName = [
    'message-assistant-frame',
    hasTrailingTool ? 'has-trailing-tool' : '',
  ]
    .filter(Boolean)
    .join(' ');
  const actionClassName = [
    'message-actions',
    hasTrailingTool ? 'is-inline-with-body' : '',
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <div className="message-row is-assistant">
      <div className="message-stack">
        {message.inProgress ? <div className="message-draft-flag">{copy.chat.assistantDraftFlag}</div> : null}
        <div className={assistantFrameClassName}>
          <div className="message-assistant-body">
            <AssistantMarkdownContent content={message.content} enabled={assistantMarkdownEnabled} />
          </div>
          {message.content ? (
            <div className={actionClassName}>
              <MessageCopyButton text={message.content} />
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
};

const ToolMessageRow: FC<{
  isOpen: boolean;
  message: ToolChatMessage;
  toolCallCompactOutputEnabled: boolean;
  onToggle: () => void;
}> = ({ isOpen, message, toolCallCompactOutputEnabled, onToggle }) => (
  <div className="message-row is-tool">
    <div className="message-stack">
      <ToolCard
        isOpen={isOpen}
        onToggle={onToggle}
        tool={message}
        toolCallCompactOutputEnabled={toolCallCompactOutputEnabled}
      />
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
  assistantMarkdownEnabled = true,
  toolCallCompactOutputEnabled = false,
  hasTrailingTool = false,
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
      return (
        <AssistantMessageRow
          message={message}
          assistantMarkdownEnabled={assistantMarkdownEnabled}
          hasTrailingTool={hasTrailingTool}
        />
      );
    case 'tool':
      return (
        <ToolMessageRow
          isOpen={isToolCardOpen}
          message={message}
          toolCallCompactOutputEnabled={toolCallCompactOutputEnabled}
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
