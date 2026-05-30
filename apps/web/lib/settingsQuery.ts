export type SettingsQueryTab =
  | 'tasks'
  | 'relay'
  | 'orchestration'
  | 'skills'
  | 'tools'
  | 'presets'
  | 'prompts_library'
  | 'prompts_preview';

const SETTINGS_QUERY_KEY = 'settings';

export function parseSettingsQuery(rawSearch: string): SettingsQueryTab | null {
  const search = rawSearch.startsWith('?') ? rawSearch.slice(1) : rawSearch;
  const params = new URLSearchParams(search);
  const tab = params.get(SETTINGS_QUERY_KEY);

  if (
    tab === 'tasks'
    || tab === 'relay'
    || tab === 'orchestration'
    || tab === 'skills'
    || tab === 'tools'
    || tab === 'presets'
    || tab === 'prompts_library'
    || tab === 'prompts_preview'
  ) {
    return tab;
  }
  return null;
}

export function buildHomeSettingsURL(tab: SettingsQueryTab): string {
  return `/?${SETTINGS_QUERY_KEY}=${tab}`;
}

export function stripSettingsQuery(rawSearch: string): string {
  const search = rawSearch.startsWith('?') ? rawSearch.slice(1) : rawSearch;
  const params = new URLSearchParams(search);
  params.delete(SETTINGS_QUERY_KEY);

  const next = params.toString();
  return next.length === 0 ? '' : `?${next}`;
}
