// MessageList component used by the web console chat/session interface.

'use client';

import type { FC } from 'react';
import { useEffect, useRef, useState } from 'react';
import { QuestionInput } from '@/components/QuestionInput';
import type { ChatMessage, ToolChatMessage } from '@/lib/types';

interface MessageListProps {
  messages: ChatMessage[];
  loading: boolean;
  onAnswerQuestion: (questionId: string, answer: string) => Promise<void>;
  onCancelQuestion: (questionId: string) => Promise<void>;
}

function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) {
    return '';
  }
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  if (bytes < 1024 * 1024 * 1024) {
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  }
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

function formatToolAction(tool: ToolChatMessage): string {
  if (tool.toolName) {
    return tool.toolName;
  }
  const fallback = tool.content?.split('\n')[0]?.trim();
  if (fallback) {
    return fallback.length > 80 ? `${fallback.slice(0, 77)}...` : fallback;
  }
  return 'Tool';
}

function formatToolDetails(tool: ToolChatMessage): string {
  const lines: string[] = [];

  if (tool.toolStatus) {
    lines.push(`status: ${tool.toolStatus}`);
  }
  if (tool.traceId) {
    lines.push(`trace_id: ${tool.traceId}`);
  }
  if (tool.toolCallId) {
    lines.push(`tool_call_id: ${tool.toolCallId}`);
  }

  const outputParts = [tool.content, tool.rawOutput].filter((value): value is string => Boolean(value && value.trim()));
  if (outputParts.length > 0) {
    lines.push(outputParts.join('\n\n'));
  }

  return lines.join('\n');
}

const CopyIcon: FC<{ className?: string }> = ({ className }) => (
  <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" className={className}>
    <rect x="6" y="6" width="10" height="10" rx="2" stroke="currentColor" strokeWidth="1.4" />
    <rect x="4" y="4" width="10" height="10" rx="2" stroke="currentColor" strokeWidth="1.4" opacity="0.55" />
  </svg>
);

const CheckIcon: FC<{ className?: string }> = ({ className }) => (
  <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" className={className}>
    <path
      d="M5 10.5l3.4 3.4L15.5 6.8"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

const ChevronIcon: FC<{ className?: string }> = ({ className }) => (
  <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" className={className}>
    <path d="M6.2 8.4L10 12.2l3.8-3.8" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
  </svg>
);

const CopyButton: FC<{ text: string }> = ({ text }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    if (!text) {
      return;
    }

    let didCopy = false;

    if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(text);
        didCopy = true;
      } catch {
        didCopy = false;
      }
    }

    if (!didCopy && typeof document !== 'undefined') {
      const textArea = document.createElement('textarea');
      textArea.value = text;
      textArea.style.position = 'fixed';
      textArea.style.opacity = '0';
      document.body.appendChild(textArea);
      textArea.select();
      try {
        didCopy = document.execCommand('copy');
      } catch {
        didCopy = false;
      }
      document.body.removeChild(textArea);
    }

    if (didCopy) {
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <button type="button" onClick={handleCopy} className={`copy-button${copied ? ' is-copied' : ''}`}>
      {copied ? <CheckIcon className="copy-icon" /> : <CopyIcon className="copy-icon" />}
      <span className="copy-label">{copied ? 'Copied' : 'Copy'}</span>
    </button>
  );
};

const ToolCard: FC<{ tool: ToolChatMessage }> = ({ tool }) => {
  const [isOpen, setIsOpen] = useState(false);
  const action = formatToolAction(tool);
  const details = formatToolDetails(tool);
  const status = tool.toolStatus?.toLowerCase();
  const isRunning = status === 'running' || status === 'pending' || status === 'in_progress';
  const isError = status === 'error' || status === 'failed';

  return (
    <div className="tool-card">
      <button
        type="button"
        onClick={() => setIsOpen((current) => !current)}
        className={`tool-card-button${isOpen ? ' is-open' : ''}${isError ? ' is-error' : ''}`}
      >
        <span className="tool-card-title" title={action}>
          {action}
        </span>
        <span className="tool-card-meta">
          <span className={`tool-status${isError ? ' is-error' : ''}`}>
            {isRunning ? <span className="tool-spinner" /> : <CheckIcon className="tool-check" />}
          </span>
          <ChevronIcon className="tool-chevron" />
        </span>
      </button>

      {isOpen ? (
        <div className="tool-details">
          <pre>{details || 'Preparing tool output...'}</pre>
        </div>
      ) : null}
    </div>
  );
};

export const MessageList: FC<MessageListProps> = ({ messages, loading, onAnswerQuestion, onCancelQuestion }) => {
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
  }, [messages, loading]);

  if (messages.length === 0) {
    return (
      <div className="messages is-empty">
        <div className="empty-state">Start by configuring a model, then send a task to Ghost-OS.</div>
      </div>
    );
  }

  return (
    <div className="messages ui-scroll" aria-live="polite">
      {messages.map((message) => {
        const isUser = message.kind === 'user';
        const isTool = message.kind === 'tool';
        const isPendingQuestion = message.kind === 'pending_question';
        const rowClass = `message-row${isUser ? ' is-user' : ''}${isTool ? ' is-tool' : ''}`;

        if (message.kind === 'user') {
          return (
            <div key={message.id} className={rowClass}>
              <div className="message-bubble is-user">{message.content}</div>
            </div>
          );
        }

        if (message.kind === 'assistant') {
          return (
            <div key={message.id} className="message-row is-assistant">
              <div className="message-stack">
                <div className="message-content">{message.content}</div>
                {message.content ? (
                  <div className="message-actions">
                    <CopyButton text={message.content} />
                  </div>
                ) : null}
              </div>
            </div>
          );
        }

        if (message.kind === 'tool') {
          return (
            <div key={message.id} className="message-row is-tool">
              <div className="message-stack">
                <ToolCard tool={message} />

                {message.attachments?.length ? (
                  <div className="attachment-list">
                    {message.attachments.map((attachment) => (
                      <div key={attachment.artifactId} className="attachment-card">
                        <div className="attachment-main">
                          <span className="attachment-name">{attachment.name}</span>
                          <span className="attachment-meta">
                            {[attachment.mimeType, formatBytes(attachment.bytes)].filter(Boolean).join(' · ') || 'Attachment'}
                          </span>
                          {attachment.note ? <span className="attachment-note">{attachment.note}</span> : null}
                        </div>
                        <div className="attachment-actions">
                          <a className="attachment-link" href={attachment.downloadUrl} download>
                            Download
                          </a>
                          <a className="attachment-link is-secondary" href={`${attachment.downloadUrl}?disposition=inline`} target="_blank" rel="noreferrer">
                            Open
                          </a>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : null}
              </div>
            </div>
          );
        }

        if (message.kind === 'system') {
          return (
            <div key={message.id} className="message-row is-system">
              <div className="message-note is-system">{message.content}</div>
            </div>
          );
        }

        if (message.kind === 'error') {
          return (
            <div key={message.id} className="message-row is-error">
              <div className="message-note is-error">{message.content}</div>
            </div>
          );
        }

        if (message.kind === 'question' || message.kind === 'pending_question') {
          return (
            <div key={message.id} className="message-row is-question">
              <div className="message-stack">
                <div className="message-note is-question">{message.content}</div>
                {isPendingQuestion ? (
                  <QuestionInput
                    loading={loading}
                    selectionMode={message.selectionMode}
                    options={message.options}
                    onAnswer={(answer) => onAnswerQuestion(message.questionId, answer)}
                    onCancel={() => onCancelQuestion(message.questionId)}
                  />
                ) : null}
              </div>
            </div>
          );
        }

        return null;
      })}

      {loading ? (
        <div className="thinking-indicator">
          <div className="thinking-dots" aria-hidden="true">
            <span className="thinking-dot" />
            <span className="thinking-dot" />
            <span className="thinking-dot" />
          </div>
          <span className="thinking-text">Thinking...</span>
        </div>
      ) : null}

      <div ref={endRef} />
    </div>
  );
};
