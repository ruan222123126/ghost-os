import { buildHomeSettingsURL, parseSettingsQuery, stripSettingsQuery } from '@/lib/settingsQuery';

describe('lib/settingsQuery', () => {
  it('parses settings=tasks from query string', () => {
    expect(parseSettingsQuery('?settings=tasks')).toBe('tasks');
    expect(parseSettingsQuery('settings=tasks')).toBe('tasks');
    expect(parseSettingsQuery('?settings=provider')).toBeNull();
  });

  it('builds and strips settings query', () => {
    expect(buildHomeSettingsURL('tasks')).toBe('/?settings=tasks');
    expect(stripSettingsQuery('?settings=tasks&foo=bar')).toBe('?foo=bar');
    expect(stripSettingsQuery('?settings=tasks')).toBe('');
  });
});
