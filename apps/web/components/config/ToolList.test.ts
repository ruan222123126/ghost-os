import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import type { WebLocale } from '@/lib/i18n/locale';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { ToolList } from './ToolList';
import type { ToolPayload } from '@/lib/types';

describe('components/config/ToolList', () => {
  const tools: ToolPayload[] = [
    { name: 'script_exec', enabled: true, prompt_override: 'sensitive internal prompt' },
    { name: 'web_search', enabled: false },
  ];

  it('renders compact cards without exposing prompt content', () => {
    const html = renderToolList({
      tools,
      loading: false,
      controlsDisabled: false,
      onUpdate: async () => {},
    });

    expect(html).toContain('script_exec');
    expect(html).toContain('Edit Prompt');
    expect(html).toContain('Prompt');
    expect(html).toContain('Custom');
    expect(html).not.toContain('sensitive internal prompt');
    expect(html).not.toContain('<textarea');
  });

  it('renders empty state and loading skeleton', () => {
    const emptyHTML = renderToolList({
      tools: [],
      loading: false,
      controlsDisabled: false,
      onUpdate: async () => {},
    });
    expect(emptyHTML).toContain('No tools found.');

    const loadingHTML = renderToolList({
      tools: [],
      loading: true,
      controlsDisabled: true,
      onUpdate: async () => {},
    });
    expect(loadingHTML.match(/animate-pulse/g)?.length ?? 0).toBe(4);
  });

  it('renders localized labels under zh-CN locale', () => {
    const html = renderToolList({
      tools,
      loading: false,
      controlsDisabled: false,
      onUpdate: async () => {},
    }, 'zh-CN');

    expect(html).toContain('编辑提示词');
    expect(html).toContain('提示词');
    expect(html).toContain('自定义');
    expect(html).toContain('已启用');
  });

  it('renders enabled tools before disabled tools', () => {
    const html = renderToolList({
      tools: [
        { name: 'web_search', enabled: false },
        { name: 'script_exec', enabled: true },
        { name: 'task_manage', enabled: true },
        { name: 'codex_cli', enabled: false },
      ],
      loading: false,
      controlsDisabled: false,
      onUpdate: async () => {},
    });

    const scriptExecIndex = html.indexOf('script_exec');
    const taskManageIndex = html.indexOf('task_manage');
    const webSearchIndex = html.indexOf('web_search');
    const codexCLIIndex = html.indexOf('codex_cli');

    expect(scriptExecIndex).toBeGreaterThan(-1);
    expect(taskManageIndex).toBeGreaterThan(-1);
    expect(webSearchIndex).toBeGreaterThan(-1);
    expect(codexCLIIndex).toBeGreaterThan(-1);
    expect(scriptExecIndex).toBeLessThan(taskManageIndex);
    expect(taskManageIndex).toBeLessThan(webSearchIndex);
    expect(webSearchIndex).toBeLessThan(codexCLIIndex);
  });
});

function renderToolList(
  props: React.ComponentProps<typeof ToolList>,
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
          children: React.createElement(ToolList, props),
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
