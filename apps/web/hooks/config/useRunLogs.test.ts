import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import type { TaskRunLog } from '@/lib/types';
import { useRunLogs, type RunLogsState } from './useRunLogs';

describe('hooks/config/useRunLogs', () => {
  it('opens, refreshes, stops, and closes run logs through injected APIs', async () => {
    const firstLog = buildRunLog('run-1');
    const refreshedLog = buildRunLog('run-2');
    const listLogs = jest.fn()
      .mockResolvedValueOnce([firstLog])
      .mockResolvedValueOnce([refreshedLog])
      .mockResolvedValueOnce([refreshedLog]);
    const stopRun = jest.fn().mockResolvedValue(undefined);
    let latest!: RunLogsState;

    renderProbe((state) => {
      latest = state;
    }, { listLogs, stopRun });

    await act(async () => {
      await latest.open('task-1');
    });

    expect(listLogs).toHaveBeenCalledWith('task-1', 20);
    expect(latest.taskID).toBe('task-1');
    expect(latest.entries).toEqual([firstLog]);

    await act(async () => {
      await latest.refresh();
    });

    expect(listLogs).toHaveBeenLastCalledWith('task-1', 20);
    expect(latest.entries).toEqual([refreshedLog]);

    await act(async () => {
      await latest.stopRun(refreshedLog);
    });

    expect(stopRun).toHaveBeenCalledWith('task-1', 'run-2');
    expect(listLogs).toHaveBeenCalledTimes(3);
    expect(latest.stoppingRunId).toBe('');

    act(() => {
      latest.close();
    });

    expect(latest.taskID).toBe('');
    expect(latest.entries).toEqual([]);
  });
});

function renderProbe(
  onRender: (state: RunLogsState) => void,
  apis: {
    listLogs: (taskID: string, limit: number) => Promise<TaskRunLog[]>;
    stopRun: (taskID: string, runID: string) => Promise<void>;
  },
): TestRenderer.ReactTestRenderer {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      React.createElement(RunLogsProbe, { onRender, ...apis }),
    );
  });

  return renderer;
}

function RunLogsProbe(props: {
  onRender: (state: RunLogsState) => void;
  listLogs: (taskID: string, limit: number) => Promise<TaskRunLog[]>;
  stopRun: (taskID: string, runID: string) => Promise<void>;
}) {
  const state = useRunLogs({
    messages: {
      empty: 'empty',
      stopFailed: 'stop failed',
    },
    listLogs: props.listLogs,
    stopRun: props.stopRun,
  });
  props.onRender(state);
  return null;
}

function buildRunLog(runID: string): TaskRunLog {
  return {
    task_id: 'task-1',
    run_id: runID,
    trace_id: `trace-${runID}`,
    scheduled_at: '2026-06-27T00:00:00Z',
    status: 'running',
  };
}
