import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { AgentMessageTaskPayload, PresetPayload, TaskRelayConfig } from '@/lib/types';
import { LoopTaskList } from './LoopTaskList';

describe('components/config/LoopTaskList', () => {
  it('renders loop cards with max rounds and preset name', () => {
    const html = renderLoopTaskList({
      loops: [createLoopTask()],
      presets: [createPreset()],
    });

    expect(html).toContain('Loop');
    expect(html).toContain('Stop: max 4 rounds');
    expect(html).toContain('Preset: Worker Preset');
  });

  it('renders AI-decides loops with the hard max round cap', () => {
    const html = renderLoopTaskList({
      loops: [createLoopTask({ stopPolicy: 'ai_decides' })],
      presets: [],
    });

    expect(html).toContain('Stop: AI may finish early; force stop at 4 rounds');
  });

  it('keeps loop action button labels on a single line', () => {
    const html = renderLoopTaskList({
      loops: [createLoopTask()],
      presets: [],
    });

    expect(html).toContain('shrink-0 whitespace-nowrap rounded-full');
  });
});

function renderLoopTaskList(input: {
  loops: AgentMessageTaskPayload[];
  presets: PresetPayload[];
}): string {
  const props: React.ComponentProps<typeof LoopTaskList> = {
    loops: input.loops,
    presets: input.presets,
    loading: false,
    controlsDisabled: false,
    runningLoopID: '',
    onEdit: () => undefined,
    onOpenLogs: async () => undefined,
    onRun: async () => undefined,
    onSetEnabled: async () => undefined,
    onDelete: async () => undefined,
    logsTaskID: '',
    logsData: [],
    logsLoading: false,
    logsError: '',
    onRefreshLogs: async () => [],
    onStopRun: async () => undefined,
    stoppingRunId: '',
    onCloseLogs: () => undefined,
  };

  return renderToStaticMarkup(
    React.createElement(
      WebLocaleProvider,
      { initialLocale: 'en-US' },
      React.createElement(LoopTaskList, props),
    ),
  );
}

function createLoopTask(input: {
  stopPolicy?: TaskRelayConfig['stop_policy'];
} = {}): AgentMessageTaskPayload {
  const stopPolicy = input.stopPolicy ?? 'max_rounds';
  return {
    id: 'loop-1',
    message: 'Finish the migration',
    agent_mode: 'relay',
    relay: createRelayConfig(stopPolicy),
    runtime_overrides: {
      preset_id: 'preset-1',
    },
    task_kind: 'agent_message',
    schedule_type: 'interval',
    interval_seconds: 300,
    enabled: true,
    created_at: '2026-05-10T00:00:00Z',
    updated_at: '2026-05-10T00:00:00Z',
  };
}

function createRelayConfig(stopPolicy: TaskRelayConfig['stop_policy']): TaskRelayConfig {
  return {
    stop_policy: stopPolicy,
    max_rounds: 4,
    execution_timeout_ms: 0,
  };
}

function createPreset(): PresetPayload {
  return {
    id: 'preset-1',
    name: 'Worker Preset',
    tool_allowlist: [],
    prompt_refs: {},
  };
}
