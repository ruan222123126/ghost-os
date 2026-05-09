import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { OrchestrationSettingsSection } from './OrchestrationSettingsSection';

const push = jest.fn();
const useOrchestrationSectionState = jest.fn();

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push }),
}));

jest.mock('./useOrchestrationSectionState', () => ({
  useOrchestrationSectionState: (...args: unknown[]) => useOrchestrationSectionState(...args),
}));

jest.mock('@/lib/api/orchestrations/api', () => ({
  listOrchestrationLogs: jest.fn(async () => []),
}));

const listOrchestrationLogs = jest.requireMock('@/lib/api/orchestrations/api').listOrchestrationLogs as jest.Mock;

describe('components/config/OrchestrationSettingsSection', () => {
  beforeEach(() => {
    push.mockReset();
    listOrchestrationLogs.mockReset();
    listOrchestrationLogs.mockResolvedValue([]);
    useOrchestrationSectionState.mockReset();
    useOrchestrationSectionState.mockReturnValue(buildState());
  });

  it('renders orchestration cards with run and logs actions', () => {
    const renderer = renderSection();
    const content = textContent(renderer.root);

    expect(content).toContain('Orchestration');
    expect(content).toContain('日报编排');
    expect(content).toContain('Enabled');
    expect(content).toContain('1 groups');
    expect(content).toContain('Run');
    expect(content).toContain('Logs');
    expect(content).toContain('Disable');
  });

  it('renders inline create form when creating', () => {
    useOrchestrationSectionState.mockReturnValue(buildState({ creating: true }));
    const renderer = renderSection();

    expect(renderer.root.findByType('form')).toBeTruthy();
  });

  it('delegates enable toggle to section state', async () => {
    const setEnabledByID = jest.fn(async () => undefined);
    useOrchestrationSectionState.mockReturnValue(buildState({ setEnabledByID }));
    const renderer = renderSection();

    await act(async () => {
      findButtonByText(renderer.root, 'Disable').props.onClick({
        stopPropagation: jest.fn(),
      });
      await Promise.resolve();
    });

    expect(setEnabledByID).toHaveBeenCalledWith('orch_1', false);
  });

  it('renders disabled orchestrations with enable action', () => {
    useOrchestrationSectionState.mockReturnValue(buildState({
      orchestrations: [buildTask({ enabled: false })],
    }));
    const renderer = renderSection();
    const content = textContent(renderer.root);

    expect(content).toContain('Disabled');
    expect(content).toContain('Enable');
  });

  it('shows running label for the in-flight orchestration', () => {
    useOrchestrationSectionState.mockReturnValue(buildState({ runningOrchestrationID: 'orch_1' }));
    const renderer = renderSection();

    expect(findButtonByText(renderer.root, 'Running...')).toBeTruthy();
    expect(findOptionalButtonByText(renderer.root, 'Run')).toBeUndefined();
  });

  it('delegates delete to section state after confirmation', async () => {
    const deleteByID = jest.fn(async () => undefined);
    const originalConfirm = globalThis.confirm;
    const confirmSpy = jest.fn(() => true);
    Object.assign(globalThis, { confirm: confirmSpy });
    useOrchestrationSectionState.mockReturnValue(buildState({ deleteByID }));
    const renderer = renderSection();

    await act(async () => {
      findButtonByAriaLabel(renderer.root, 'Delete orchestration orch_1').props.onClick({
        stopPropagation: jest.fn(),
      });
      await Promise.resolve();
    });

    expect(confirmSpy).toHaveBeenCalledWith('Delete orchestration "orch_1"?');
    expect(deleteByID).toHaveBeenCalledWith('orch_1');
    Object.assign(globalThis, { confirm: originalConfirm });
  });

  it('refreshes visible logs after run completes', async () => {
    const runByID = jest.fn(async () => undefined);
    listOrchestrationLogs.mockResolvedValue([]);
    useOrchestrationSectionState.mockReturnValue(buildState({ runByID }));
    const renderer = renderSection();

    await act(async () => {
      findButtonByText(renderer.root, 'Logs').props.onClick({
        stopPropagation: jest.fn(),
      });
      await Promise.resolve();
    });

    expect(listOrchestrationLogs).toHaveBeenCalledTimes(1);

    await act(async () => {
      findButtonByText(renderer.root, 'Run').props.onClick({
        stopPropagation: jest.fn(),
      });
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(runByID).toHaveBeenCalledWith('orch_1');
    expect(listOrchestrationLogs).toHaveBeenCalledTimes(2);
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

function buildState(overrides?: Record<string, unknown>) {
  return {
    orchestrations: [buildTask()],
    loading: false,
    error: '',
    creating: false,
    name: '',
    submitting: false,
    runningOrchestrationID: '',
    controlsDisabled: false,
    refresh: jest.fn(),
    startCreate: jest.fn(),
    setName: jest.fn(),
    submitCreate: jest.fn(async () => undefined),
    cancelCreate: jest.fn(),
    runByID: jest.fn(async () => undefined),
    setEnabledByID: jest.fn(async () => undefined),
    deleteByID: jest.fn(async () => undefined),
    ...overrides,
  };
}

function buildTask(overrides?: Record<string, unknown>) {
  return {
    id: 'orch_1',
    name: '日报编排',
    task_kind: 'orchestration',
    orchestration: {
      nodes: [
        { id: 'start-node', type: 'start' },
        { id: 'group-1', type: 'group', group: { title: '群组 1', shared_context: '', speaking_mode: 'sequential', max_rounds: 1 } },
        { id: 'agent-1', type: 'agent', agent: { title: '角色 1', message: 'member' } },
        { id: 'end-node', type: 'end' },
      ],
      edges: [],
    },
    schedule_type: 'interval',
    interval_seconds: 60,
    enabled: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

function findButtonByText(
  root: TestRenderer.ReactTestInstance,
  label: string,
): TestRenderer.ReactTestInstance {
  return root.findAllByType('button').find((node) => textContent(node) === label) as TestRenderer.ReactTestInstance;
}

function findOptionalButtonByText(
  root: TestRenderer.ReactTestInstance,
  label: string,
): TestRenderer.ReactTestInstance | undefined {
  return root.findAllByType('button').find((node) => textContent(node) === label);
}

function findButtonByAriaLabel(
  root: TestRenderer.ReactTestInstance,
  label: string,
): TestRenderer.ReactTestInstance {
  return root.findAllByType('button').find((node) => node.props['aria-label'] === label) as TestRenderer.ReactTestInstance;
}

function textContent(node: TestRenderer.ReactTestInstance): string {
  return node.children.map((child: string | number | TestRenderer.ReactTestInstance) => {
    if (typeof child === 'string' || typeof child === 'number') {
      return String(child);
    }
    return textContent(child);
  }).join('');
}
