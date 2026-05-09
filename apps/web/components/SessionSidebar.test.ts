import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { SessionSidebar } from './SessionSidebar';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { SESSION_SIDEBAR_GROUPING_STORAGE_KEY } from '@/hooks/useSessionSidebarGroupingPreference';
import type { SessionMetadata } from '@/lib/types';

describe('components/SessionSidebar', () => {
  it('renders flat sessions by recent activity when grouping is disabled', () => {
    const sessions: SessionMetadata[] = [
      createSession('older-3333', '2026-04-10T00:00:00Z'),
      createSession('latest-1111', '2026-04-12T12:00:00Z'),
      createSession('middle-2222', '2026-04-11T08:00:00Z'),
    ];

    const html = renderSidebar(sessions, { groupingEnabled: false });

    expect(html.indexOf('Session latest-1')).toBeLessThan(html.indexOf('Session middle-2'));
    expect(html.indexOf('Session middle-2')).toBeLessThan(html.indexOf('Session older-33'));
  });

  it('sorts grouped sessions by recent activity', () => {
    const sessions: SessionMetadata[] = [
      createSession('older-3333', '2026-04-10T00:00:00Z'),
      createSession('latest-1111', '2026-04-12T12:00:00Z'),
      createSession('middle-2222', '2026-04-11T08:00:00Z'),
    ];
    const html = renderSidebar(sessions, { groupingEnabled: true });

    expect(html.indexOf('Session latest-1')).toBeLessThan(html.indexOf('Session middle-2'));
    expect(html.indexOf('Session middle-2')).toBeLessThan(html.indexOf('Session older-33'));
  });
});

function renderSidebar(
  sessions: SessionMetadata[],
  options?: {
    groupingEnabled?: boolean;
  },
): string {
  const originalWindow = (globalThis as { window?: unknown }).window;
  const groupingEnabled = options?.groupingEnabled ?? true;
  const store = {
    'ghost.web.locale': 'en-US',
    [SESSION_SIDEBAR_GROUPING_STORAGE_KEY]: groupingEnabled ? '1' : '0',
  };

  (globalThis as { window?: unknown }).window = {
    localStorage: buildLocalStorageMock(store),
    navigator: {
      language: 'en-US',
    },
  };

  try {
    return renderToStaticMarkup(
      React.createElement(
        WebLocaleProvider,
        {
          initialLocale: 'en-US',
        },
        React.createElement(SessionSidebar, {
          sessions,
          currentSessionId: '',
          loading: false,
          error: '',
          onSelect: () => {},
          onDelete: () => {},
          onNewChat: () => {},
          onOpenSettings: () => {},
        }),
      ),
    );
  } finally {
    (globalThis as { window?: unknown }).window = originalWindow;
  }
}

function createSession(id: string, updatedAt = '2026-04-12T00:00:00Z'): SessionMetadata {
  return {
    id,
    created_at: '2026-04-12T00:00:00Z',
    updated_at: updatedAt,
    message_count: 0,
    token_count: 0,
  };
}

function buildLocalStorageMock(store: Record<string, string>): Storage {
  return {
    getItem: (key: string) => store[key] ?? null,
    setItem: () => undefined,
    removeItem: () => undefined,
    clear: () => undefined,
    key: () => null,
    get length() {
      return Object.keys(store).length;
    },
  };
}
