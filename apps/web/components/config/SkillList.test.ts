import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import type { WebLocale } from '@/lib/i18n/locale';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { SkillList, requestSkillDelete, sortSkillsForDisplay } from './SkillList';
import type { SkillPayload } from '@/lib/types';

describe('components/config/SkillList', () => {
  const skills: SkillPayload[] = [
    {
      id: 'skill_repo_release',
      name: 'release',
      description: 'Automate release flow',
      path: '/tmp/project/.agents/skills/release',
      source: 'repo',
      enabled: true,
    },
  ];

  it('renders skill cards with source and path', () => {
    const html = renderSkillList({
      skills,
      loading: false,
      controlsDisabled: false,
      onUpdate: async () => {},
      onDelete: async () => {},
    });

    expect(html).toContain('release');
    expect(html).toContain('Automate release flow');
    expect(html).toContain('/tmp/project/.agents/skills/release');
    expect(html).toContain('Repo');
    expect(html).toContain('Enabled');
    expect(html).toContain('Disable');
    expect(html).toContain('Delete');
    expect(html).toContain('whitespace-nowrap');
  });

  it('renders empty state and loading skeleton', () => {
    const emptyHTML = renderSkillList({
      skills: [],
      loading: false,
      controlsDisabled: false,
      onUpdate: async () => {},
      onDelete: async () => {},
    });
    expect(emptyHTML).toContain('No skills found.');

    const loadingHTML = renderSkillList({
      skills: [],
      loading: true,
      controlsDisabled: true,
      onUpdate: async () => {},
      onDelete: async () => {},
    });
    expect(loadingHTML.match(/animate-pulse/g)?.length ?? 0).toBe(3);
  });

  it('sorts enabled skills before disabled skills', () => {
    const sorted = sortSkillsForDisplay([
      { ...skills[0], id: 'disabled', enabled: false },
      { ...skills[0], id: 'enabled', enabled: true, name: 'alpha' },
    ]);

    expect(sorted.map((item) => item.id)).toEqual(['enabled', 'disabled']);
  });

  it('uses confirmation before delete', async () => {
    const onDelete = jest.fn(async () => {});
    const originalWindow = (globalThis as { window?: Window }).window;
    const confirm = jest.fn(() => true);
    (globalThis as { window?: Window }).window = { confirm } as unknown as Window;

    requestSkillDelete({ id: 'skill_repo_release', name: 'release', onDelete });
    expect(confirm).toHaveBeenCalled();
    expect(onDelete).toHaveBeenCalledWith('skill_repo_release');

    confirm.mockReturnValue(false);
    requestSkillDelete({ id: 'skill_repo_release', name: 'release', onDelete });
    expect(onDelete).toHaveBeenCalledTimes(1);

    (globalThis as { window?: Window }).window = originalWindow;
  });
});

function renderSkillList(
  props: React.ComponentProps<typeof SkillList>,
  locale: WebLocale = 'en-US',
): string {
  const originalWindow = (globalThis as { window?: unknown }).window;
  const localStorageMock = buildLocalStorageMock(locale);
  (globalThis as { window?: unknown }).window = {
    localStorage: localStorageMock,
    navigator: {
      language: locale,
    },
  };

  try {
    return renderToStaticMarkup(
      React.createElement(
        WebLocaleProvider,
        {
          children: React.createElement(SkillList, props),
          initialLocale: locale,
        },
      ),
    );
  } finally {
    (globalThis as { window?: unknown }).window = originalWindow;
  }
}

function buildLocalStorageMock(locale: WebLocale): Storage {
  return {
    getItem: (key: string) => (key === 'ghost.web.locale' ? locale : null),
    setItem: () => undefined,
    removeItem: () => undefined,
    clear: () => undefined,
    key: () => null,
    get length() {
      return 1;
    },
  };
}
