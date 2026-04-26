import { buildHomeSettingsURL, parseSettingsQuery, stripSettingsQuery } from '@/lib/settingsQuery';

describe('lib/settingsQuery', () => {
  it('parses supported settings tabs from query string', () => {
    expect(parseSettingsQuery('?settings=tasks')).toBe('tasks');
    expect(parseSettingsQuery('?settings=skills')).toBe('skills');
    expect(parseSettingsQuery('?settings=tools')).toBe('tools');
    expect(parseSettingsQuery('?settings=prompts_library')).toBe('prompts_library');
    expect(parseSettingsQuery('?settings=prompts_preview')).toBe('prompts_preview');
    expect(parseSettingsQuery('settings=tasks')).toBe('tasks');
    expect(parseSettingsQuery('?settings=provider')).toBeNull();
  });

  it('maps legacy prompts query to prompts_library', () => {
    expect(parseSettingsQuery('?settings=prompts')).toBe('prompts_library');
  });

  it('builds and strips settings query', () => {
    expect(buildHomeSettingsURL('tasks')).toBe('/?settings=tasks');
    expect(buildHomeSettingsURL('skills')).toBe('/?settings=skills');
    expect(buildHomeSettingsURL('tools')).toBe('/?settings=tools');
    expect(buildHomeSettingsURL('prompts_library')).toBe('/?settings=prompts_library');
    expect(buildHomeSettingsURL('prompts_preview')).toBe('/?settings=prompts_preview');
    expect(stripSettingsQuery('?settings=tasks&foo=bar')).toBe('?foo=bar');
    expect(stripSettingsQuery('?settings=tasks')).toBe('');
  });
});
