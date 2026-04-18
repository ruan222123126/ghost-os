// SessionItem component used by the web console chat/session interface.

'use client';

import type { FC, MouseEvent } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';
import type { SessionMetadata } from '@/lib/types';

interface SessionItemProps {
  session: SessionMetadata;
  isActive: boolean;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
}

function formatRelativeTime(
  value: string,
  locale: ReturnType<typeof useWebLocale>['locale'],
  copy: ReturnType<typeof useWebLocale>['copy'],
): string {
  const timestamp = Date.parse(value);
  if (Number.isNaN(timestamp)) {
    return value;
  }

  const diffSeconds = Math.max(0, Math.floor((Date.now() - timestamp) / 1000));
  if (diffSeconds < 60) {
    return copy.chat.sessionRelativeNow;
  }
  if (diffSeconds < 3600) {
    return copy.chat.sessionRelativeMinutes(Math.floor(diffSeconds / 60));
  }
  if (diffSeconds < 86400) {
    return copy.chat.sessionRelativeHours(Math.floor(diffSeconds / 3600));
  }
  if (diffSeconds < 604800) {
    return copy.chat.sessionRelativeDays(Math.floor(diffSeconds / 86400));
  }
  return new Date(timestamp).toLocaleDateString(locale);
}

export const SessionItem: FC<SessionItemProps> = ({ session, isActive, onSelect, onDelete }) => {
  const { copy, locale } = useWebLocale();
  const shortID = session.id.slice(0, 8);

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => onSelect(session.id)}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault();
          onSelect(session.id);
        }
      }}
      className={`session-card${isActive ? ' is-active' : ''}`}
    >
      <div className="session-card-head">
        <div>
          <p className="session-card-title mono">{copy.chat.sidebarSessionTitle(shortID)}</p>
          <p className="session-card-meta">{copy.chat.sessionCreatedPrefix} {formatRelativeTime(session.created_at, locale, copy)}</p>
        </div>

        <button
          type="button"
          onClick={(event: MouseEvent<HTMLButtonElement>) => {
            event.stopPropagation();
            onDelete(session.id);
          }}
          className="button-danger session-card-delete"
          aria-label={copy.chat.sessionDeleteAria(shortID)}
        >
          {copy.chat.sessionDelete}
        </button>
      </div>
    </div>
  );
};
