import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { OrchestrationSettingsSection } from './OrchestrationSettingsSection';

const push = jest.fn();
const createOrchestration = jest.fn();
const listOrchestrationSummaries = jest.fn();

jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push,
  }),
}));

jest.mock('@/lib/orchestrationStore', () => ({
  createOrchestration: (...args: unknown[]) => createOrchestration(...args),
  listOrchestrationSummaries: (...args: unknown[]) => listOrchestrationSummaries(...args),
}));

describe('components/config/OrchestrationSettingsSection', () => {
  beforeEach(() => {
    push.mockReset();
    createOrchestration.mockReset();
    listOrchestrationSummaries.mockReset();
    listOrchestrationSummaries.mockReturnValue([
      {
        id: 'orch_1',
        name: '日报编排',
        createdAt: 1,
        updatedAt: 2,
        stepCount: 3,
      },
    ]);
  });

  it('renders orchestration cards from local storage summaries', () => {
    const renderer = renderSection();
    const content = textContent(renderer.root);

    expect(content).toContain('Orchestration');
    expect(content).toContain('日报编排');
    expect(content).toContain('3 steps');
    expect(content).toContain('Open editor');
  });

  it('creates orchestration inline without navigating to a create page', () => {
    listOrchestrationSummaries
      .mockReturnValueOnce([])
      .mockReturnValueOnce([
        {
          id: 'orch_2',
          name: 'Morning Orchestration',
          createdAt: 1,
          updatedAt: 2,
          stepCount: 0,
        },
      ]);

    const renderer = renderSection();

    act(() => {
      findButtonByText(renderer.root, 'New Orchestration').props.onClick();
    });

    const input = renderer.root.findByType('input');
    act(() => {
      input.props.onChange({ target: { value: 'Morning Orchestration' } });
    });

    act(() => {
      renderer.root.findByType('form').props.onSubmit({ preventDefault: () => undefined });
    });

    expect(createOrchestration).toHaveBeenCalledWith({ name: 'Morning Orchestration' });
    expect(push).not.toHaveBeenCalledWith('/orchestration/new');
    expect(textContent(renderer.root)).toContain('Morning Orchestration');
  });
});

function renderSection(): TestRenderer.ReactTestRenderer {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      React.createElement(WebLocaleProvider, {
        initialLocale: 'en-US',
        children: React.createElement(OrchestrationSettingsSection),
      }),
    );
  });

  return renderer;
}

function textContent(node: TestRenderer.ReactTestInstance): string {
  return node.children.map((child: string | number | TestRenderer.ReactTestInstance) => {
    if (typeof child === 'string' || typeof child === 'number') {
      return String(child);
    }
    return textContent(child);
  }).join('');
}

function findButtonByText(root: TestRenderer.ReactTestInstance, text: string): TestRenderer.ReactTestInstance {
  return root.find((node) => node.type === 'button' && textContent(node) === text);
}
