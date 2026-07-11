import {
  createInitialOrchestrationState,
  orchestrationReducer,
} from '@/lib/config-panel/orchestrationMachine';
import type { OrchestrationTaskPayload } from '@/lib/types';

describe('lib/config-panel/orchestrationMachine', () => {
  it('enters create mode, edits name, and cancels', () => {
    const creating = orchestrationReducer(createInitialOrchestrationState(), {
      type: 'enter_create',
    });
    const named = orchestrationReducer(creating, {
      type: 'set_name',
      name: 'daily orchestration',
    });
    const cancelled = orchestrationReducer(named, { type: 'cancel_create' });

    expect(creating.view).toBe('create');
    expect(named.name).toBe('daily orchestration');
    expect(cancelled.view).toBe('list');
    expect(cancelled.name).toBe('');
  });

  it('loads, replaces, and removes orchestrations', () => {
    const first = buildTask({ id: 'orch-1', name: 'First' });
    const second = buildTask({ id: 'orch-2', name: 'Second' });
    const updated = buildTask({ id: 'orch-1', name: 'Updated' });
    const loaded = orchestrationReducer(createInitialOrchestrationState(), {
      type: 'load_success',
      orchestrations: [first, second],
    });
    const replaced = orchestrationReducer(loaded, {
      type: 'replace_orchestration',
      orchestration: updated,
    });
    const removed = orchestrationReducer(replaced, {
      type: 'remove_orchestration',
      id: 'orch-2',
    });

    expect(loaded.loading).toBe(false);
    expect(replaced.orchestrations).toEqual([updated, second]);
    expect(removed.orchestrations).toEqual([updated]);
  });
});

function buildTask(overrides: Partial<OrchestrationTaskPayload>): OrchestrationTaskPayload {
  return {
    id: 'orch-1',
    name: 'Orchestration',
    task_kind: 'orchestration',
    orchestration: {
      nodes: [],
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
