import { buildHomeSettingsURL, parseSettingsQuery, stripSettingsQuery } from '@/lib/settingsQuery';

describe('lib/settingsQuery', () => {
  it('parses supported settings tabs from query string', () => {
    expect(parseSettingsQuery('?settings=tasks')).toBe('tasks');
    expect(parseSettingsQuery('?settings=relay')).toBe('relay');
    expect(parseSettingsQuery('?settings=orchestration')).toBe('orchestration');
    expect(parseSettingsQuery('?settings=skills')).toBe('skills');
    expect(parseSettingsQuery('?settings=tools')).toBe('tools');
    expect(parseSettingsQuery('?settings=presets')).toBe('presets');
    expect(parseSettingsQuery('?settings=prompts_library')).toBe('prompts_library');
    expect(parseSettingsQuery('?settings=prompts_preview')).toBe('prompts_preview');
    expect(parseSettingsQuery('settings=tasks')).toBe('tasks');
    expect(parseSettingsQuery('?settings=provider')).toBeNull();
  });

  it('ignores legacy prompts query tab', () => {
    expect(parseSettingsQuery('?settings=prompts')).toBeNull();
  });

  it('builds and strips settings query', () => {
    expect(buildHomeSettingsURL('tasks')).toBe('/?settings=tasks');
    expect(buildHomeSettingsURL('relay')).toBe('/?settings=relay');
    expect(buildHomeSettingsURL('orchestration')).toBe('/?settings=orchestration');
    expect(buildHomeSettingsURL('skills')).toBe('/?settings=skills');
    expect(buildHomeSettingsURL('tools')).toBe('/?settings=tools');
    expect(buildHomeSettingsURL('presets')).toBe('/?settings=presets');
    expect(buildHomeSettingsURL('prompts_library')).toBe('/?settings=prompts_library');
    expect(buildHomeSettingsURL('prompts_preview')).toBe('/?settings=prompts_preview');
    expect(stripSettingsQuery('?settings=tasks&foo=bar')).toBe('?foo=bar');
    expect(stripSettingsQuery('?settings=tasks')).toBe('');
  });
});
