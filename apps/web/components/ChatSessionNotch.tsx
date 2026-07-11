'use client';

import type { FC } from 'react';

interface ChatSessionNotchProps {
  sessionId: string;
  title: string;
}

export const ChatSessionNotch: FC<ChatSessionNotchProps> = ({ sessionId, title }) => {
  const normalizedSessionId = sessionId.trim();
  if (!normalizedSessionId) {
    return null;
  }

  const displayTitle = title.trim();

  return (
    <header className="chat-session-notch">
      <span className="chat-session-notch-id" title={normalizedSessionId}>{normalizedSessionId}</span>
      <span className="chat-session-notch-title" title={displayTitle}>{displayTitle}</span>
      <span className="chat-session-notch-spacer" aria-hidden="true" />
    </header>
  );
};
