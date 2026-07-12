'use client';

import type { FC } from 'react';
import { useState } from 'react';
import { COPY_FEEDBACK_RESET_DELAY_MS, copyTextToClipboard } from '../../../shared/browserClipboard';
import { useWebLocale } from '@/lib/i18n/provider';

const CODE_BUTTON_VARIANT = 'code';

interface MessageCopyButtonProps {
  text: string;
  variant?: 'default' | 'code';
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

export const MessageCopyButton: FC<MessageCopyButtonProps> = ({
  text,
  variant = 'default',
}) => {
  const { copy } = useWebLocale();
  const [copied, setCopied] = useState(false);
  const isCodeVariant = variant === CODE_BUTTON_VARIANT;
  const buttonLabel = copied ? copy.chat.copied : copy.chat.copy;
  const buttonClassName = [
    'copy-button',
    isCodeVariant ? 'is-code-block' : '',
    copied ? 'is-copied' : '',
  ]
    .filter(Boolean)
    .join(' ');

  const handleCopy = async () => {
    if (!text) {
      return;
    }

    const didCopy = await copyTextToClipboard(text);
    if (!didCopy) {
      return;
    }

    setCopied(true);
    window.setTimeout(() => setCopied(false), COPY_FEEDBACK_RESET_DELAY_MS);
  };

  return (
    <button
      type="button"
      onClick={handleCopy}
      className={buttonClassName}
      aria-label={buttonLabel}
    >
      {copied ? (
        <CheckIcon className="copy-icon" />
      ) : (
        <CopyIcon className="copy-icon" />
      )}
      {!isCodeVariant ? <span className="copy-label">{buttonLabel}</span> : null}
    </button>
  );
};
