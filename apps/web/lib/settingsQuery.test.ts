import { buildHomeSettingsURL, parseSettingsQuery, stripSettingsQuery } from '@/lib/settingsQuery';

describe('lib/settingsQuery', () => {
  it('parses supported settings tabs from query string', () => {
    expect(parseSettingsQuery('?settings=tasks')).toBe('tasks');
    expect(parseSettingsQuery('?settings=skills')).toBe('skills');
    expect(parseSettingsQuery('?settings=tools')).toBe('tools');
    expect(parseSettingsQuery('settings=tasks')).toBe('tasks');
    expect(parseSettingsQuery('?settings=provider')).toBeNull();
  });

  it('builds and strips settings query', () => {
    expect(buildHomeSettingsURL('tasks')).toBe('/?settings=tasks');
    expect(buildHomeSettingsURL('skills')).toBe('/?settings=skills');
    expect(buildHomeSettingsURL('tools')).toBe('/?settings=tools');
    expect(stripSettingsQuery('?settings=tasks&foo=bar')).toBe('?foo=bar');
    expect(stripSettingsQuery('?settings=tasks')).toBe('');
  });
});
