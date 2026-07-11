import { resolveSessionTitleValue } from './useSessionSidebarAliases';

describe('hooks/useSessionSidebarAliases', () => {
  it('resolves alias before backend title and default title', () => {
    expect(resolveSessionTitleValue(' Local ', 'Backend', 'Session 123')).toBe('Local');
    expect(resolveSessionTitleValue(undefined, ' Backend ', 'Session 123')).toBe('Backend');
    expect(resolveSessionTitleValue('   ', '   ', 'Session 123')).toBe('Session 123');
  });
});
