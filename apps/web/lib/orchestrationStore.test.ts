import {
  createOrchestration,
  getOrchestrationByID,
  listOrchestrationSummaries,
  saveOrchestrationDraft,
} from '@/lib/orchestrationStore';

describe('lib/orchestrationStore', () => {
  it('creates, lists, and updates orchestration drafts', () => {
    const storage = createMemoryStorage();
    const created = createOrchestration({ name: '晨间编排' }, storage);

    expect(created.name).toBe('晨间编排');
    expect(created.draft.mode).toBe('edit');
    expect(listOrchestrationSummaries(storage)).toEqual([
      expect.objectContaining({
        id: created.id,
        name: '晨间编排',
        stepCount: 0,
      }),
    ]);

    const nextDraft = {
      ...created.draft,
      nodes: [
        ...created.draft.nodes,
        {
          id: 'agent_1',
          type: 'agent' as const,
          position: { x: 120, y: 160 },
          ui: { toolArgumentsMode: 'json' as const },
          agent: {
            message: '执行编排',
            runtime_overrides: {
              provider_name: 'openai-main',
              model: 'gpt-5.4',
              system_prompt: '只输出结果',
              tool_allowlist_only: true,
              tool_allowlist: ['script_exec'],
              max_turns: 2,
            },
          },
        },
      ],
    };

    const updated = saveOrchestrationDraft(created.id, nextDraft, storage);
    expect(updated.updatedAt).toBeGreaterThanOrEqual(created.updatedAt);

    const loaded = getOrchestrationByID(created.id, storage);
    expect(loaded).toEqual(expect.objectContaining({
      id: created.id,
      name: '晨间编排',
    }));
    expect(loaded?.draft.nodes).toHaveLength(3);
    expect(loaded?.draft.nodes[2]?.agent?.runtime_overrides).toEqual({
      provider_name: 'openai-main',
      model: 'gpt-5.4',
      system_prompt: '只输出结果',
      tool_allowlist_only: true,
      tool_allowlist: ['script_exec'],
      max_turns: 2,
    });
    expect(listOrchestrationSummaries(storage)[0]?.stepCount).toBe(1);
  });

  it('rejects empty orchestration names', () => {
    const storage = createMemoryStorage();

    expect(() => createOrchestration({ name: '   ' }, storage)).toThrow('orchestration name is required');
  });
});

function createMemoryStorage(): Storage {
  const data = new Map<string, string>();

  return {
    get length() {
      return data.size;
    },
    clear() {
      data.clear();
    },
    getItem(key: string) {
      return data.get(key) ?? null;
    },
    key(index: number) {
      return [...data.keys()][index] ?? null;
    },
    removeItem(key: string) {
      data.delete(key);
    },
    setItem(key: string, value: string) {
      data.set(key, value);
    },
  };
}
