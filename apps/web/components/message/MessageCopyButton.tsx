'use client';

import type { FC } from 'react';
import { useState } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';

const COPY_RESET_DELAY_MS = 2000;

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

async function copyWithClipboard(text: string): Promise<boolean> {
  if (typeof navigator === 'undefined' || !navigator.clipboard?.writeText) {
    return false;
  }

  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

function copyWithTextArea(text: string): boolean {
  if (typeof document === 'undefined') {
    return false;
  }

  const textArea = document.createElement('textarea');
  textArea.value = text;
  textArea.style.position = 'fixed';
  textArea.style.opacity = '0';
  document.body.appendChild(textArea);
  textArea.select();

  try {
    return document.execCommand('copy');
  } catch {
    return false;
  } finally {
    document.body.removeChild(textArea);
  }
}

export const MessageCopyButton: FC<{ text: string }> = ({ text }) => {
  const { copy } = useWebLocale();
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    if (!text) {
      return;
    }

    const didCopy = (await copyWithClipboard(text)) || copyWithTextArea(text);
    if (!didCopy) {
      return;
    }

    setCopied(true);
    window.setTimeout(() => setCopied(false), COPY_RESET_DELAY_MS);
  };

  return (
    <button type="button" onClick={handleCopy} className={`copy-button${copied ? ' is-copied' : ''}`}>
      {copied ? <CheckIcon className="copy-icon" /> : <CopyIcon className="copy-icon" />}
      <span className="copy-label">{copied ? copy.chat.copied : copy.chat.copy}</span>
    </button>
  );
};
