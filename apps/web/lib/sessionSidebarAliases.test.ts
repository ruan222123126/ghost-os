import {
  createInitialSessionAliasStore,
  parseSessionAliasStore,
  sanitizeSessionAliasStore,
  setSessionAlias,
  stringifySessionAliasStore,
} from '@/lib/sessionSidebarAliases';
import type { SessionMetadata } from '@/lib/types';

describe('lib/sessionSidebarAliases', () => {
  it('sets and serializes aliases', () => {
    const store = setSessionAlias(createInitialSessionAliasStore(), 'session-1', 'Planning');

    expect(store.aliases).toEqual({ 'session-1': 'Planning' });
    expect(parseSessionAliasStore(stringifySessionAliasStore(store))).toEqual(store);
  });

  it('sanitizes deleted sessions and empty aliases', () => {
    const sessions = [createSession('session-1'), createSession('session-2')];
    const sanitized = sanitizeSessionAliasStore(
      {
        version: 1,
        aliases: {
          'session-1': 'Inbox',
          'session-2': '   ',
          'session-3': 'legacy',
        },
      },
      sessions,
    );

    expect(sanitized.aliases).toEqual({ 'session-1': 'Inbox' });
  });

  it('throws when parsing malformed payload', () => {
    expect(() => parseSessionAliasStore('{bad json')).toThrow('Invalid session alias payload JSON');
    expect(() => parseSessionAliasStore('{"version":1,"aliases":[]}')).toThrow(
      'Invalid session alias payload.aliases: expected object',
    );
  });

  it('rejects empty alias updates', () => {
    expect(() => setSessionAlias(createInitialSessionAliasStore(), 'session-1', '   ')).toThrow(
      'Session alias cannot be empty',
    );
  });
});

function createSession(id: string): SessionMetadata {
  return {
    id,
    title: '',
    created_at: '2026-04-12T00:00:00Z',
    updated_at: '2026-04-12T00:00:00Z',
    message_count: 0,
    token_count: 0,
  };
}
