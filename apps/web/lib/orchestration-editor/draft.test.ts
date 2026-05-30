import {
  createEmptyOrchestrationDraft,
  draftToOrchestrationDefinition,
  orchestrationDefinitionToDraft,
} from '@/lib/orchestration-editor/draft';
import type { WorkflowCanvasDraft } from '@/lib/workflow-editor/types';

describe('lib/orchestration-editor/draft', () => {
  it('creates an empty draft without start/end boundary nodes', () => {
    expect(createEmptyOrchestrationDraft()).toEqual({
      mode: 'create',
      schedule: {
        mode: 'interval',
        intervalSeconds: '300',
        cronExpr: '',
      },
      nodes: [],
      edges: [],
    });
  });

  it('preserves legacy boundary nodes when loading a definition', () => {
    const draft = orchestrationDefinitionToDraft({
      scheduleType: 'interval',
      intervalSeconds: 60,
      orchestration: {
        nodes: [
          { id: 'start-node', type: 'start' },
          {
            id: 'group-a',
            type: 'group',
            group: { title: 'Group A', shared_context: '', speaking_mode: 'sequential', owner_agent_id: '', max_rounds: 1 },
          },
          {
            id: 'group-b',
            type: 'group',
            group: { title: 'Group B', shared_context: '', speaking_mode: 'parallel', owner_agent_id: '', max_rounds: 2 },
          },
          {
            id: 'agent-1',
            type: 'agent',
            agent: {
              title: 'Alpha',
              message: 'hello',
              runtime_overrides: {
                preset_id: 'preset-a',
                max_turns: 3,
              },
            },
          },
          { id: 'end-node', type: 'end' },
        ],
        edges: [
          { from_node_id: 'start-node', to_node_id: 'group-a', kind: 'control' },
          { from_node_id: 'group-a', to_node_id: 'group-b', kind: 'control' },
          { from_node_id: 'group-b', to_node_id: 'end-node', kind: 'control' },
          { from_node_id: 'agent-1', to_node_id: 'group-a', kind: 'member' },
        ],
      },
    });

    expect(draft.nodes.map((node) => node.id)).toEqual(['start-node', 'group-a', 'group-b', 'agent-1', 'end-node']);
    expect(draft.nodes.find((node) => node.id === 'agent-1')?.agent?.runtime_overrides).toEqual({
      preset_id: 'preset-a',
    });
    expect(draft.edges).toEqual([
      expect.objectContaining({ from_node_id: 'start-node', to_node_id: 'group-a', kind: 'control' }),
      expect.objectContaining({ from_node_id: 'group-a', to_node_id: 'group-b', kind: 'control' }),
      expect.objectContaining({ from_node_id: 'group-b', to_node_id: 'end-node', kind: 'control' }),
      expect.objectContaining({ from_node_id: 'agent-1', to_node_id: 'group-a', kind: 'member' }),
    ]);
  });

  it('preserves boundary nodes when compiling orchestration payloads', () => {
    const draft: WorkflowCanvasDraft = {
      mode: 'edit',
      schedule: { mode: 'interval', intervalSeconds: '60', cronExpr: '' },
      nodes: [
        { id: 'start-node', type: 'start', position: { x: 0, y: 0 }, ui: { toolArgumentsMode: 'kv' }, start: { inputs: [] } },
        {
          id: 'group-a',
          type: 'group',
          position: { x: 200, y: 0 },
          ui: { toolArgumentsMode: 'kv' },
          group: { title: 'Group A', shared_context: '', speaking_mode: 'sequential', owner_agent_id: '', max_rounds: 1 },
        },
        {
          id: 'agent-1',
          type: 'agent',
          position: { x: 200, y: -120 },
          ui: { toolArgumentsMode: 'kv' },
          agent: {
            title: 'Alpha',
            message: 'hello',
            runtime_overrides: {
              preset_id: 'preset-a',
              max_turns: 3,
            },
          },
        },
        { id: 'end-node', type: 'end', position: { x: 400, y: 0 }, ui: { toolArgumentsMode: 'kv' } },
      ],
      edges: [
        { id: 'edge-1', from_node_id: 'start-node', to_node_id: 'group-a', kind: 'control' },
        { id: 'edge-2', from_node_id: 'group-a', to_node_id: 'end-node', kind: 'control' },
        { id: 'edge-3', from_node_id: 'agent-1', to_node_id: 'group-a', kind: 'member' },
      ],
    };

    expect(draftToOrchestrationDefinition(draft)).toEqual({
      nodes: [
        {
          id: 'start-node',
          type: 'start',
          group: undefined,
          agent: undefined,
        },
        {
          id: 'group-a',
          type: 'group',
          group: { title: 'Group A', shared_context: '', speaking_mode: 'sequential', owner_agent_id: '', max_rounds: 1 },
          agent: undefined,
        },
        {
          id: 'agent-1',
          type: 'agent',
          group: undefined,
          agent: {
            title: 'Alpha',
            message: 'hello',
            runtime_overrides: {
              preset_id: 'preset-a',
            },
          },
        },
        {
          id: 'end-node',
          type: 'end',
          group: undefined,
          agent: undefined,
        },
      ],
      edges: [
        { from_node_id: 'start-node', to_node_id: 'group-a', kind: 'control' },
        { from_node_id: 'group-a', to_node_id: 'end-node', kind: 'control' },
        { from_node_id: 'agent-1', to_node_id: 'group-a', kind: 'member' },
      ],
    });
  });

  it('preserves owner_agent_id during orchestration round-trip', () => {
    const draft = orchestrationDefinitionToDraft({
      scheduleType: 'interval',
      intervalSeconds: 60,
      orchestration: {
        nodes: [
          {
            id: 'group-a',
            type: 'group',
            group: {
              title: 'Group A',
              shared_context: 'shared',
              speaking_mode: 'owner',
              owner_agent_id: 'agent-1',
              max_rounds: 2,
            },
          },
          {
            id: 'agent-1',
            type: 'agent',
            agent: { title: 'Owner', message: 'dispatch' },
          },
        ],
        edges: [{ from_node_id: 'agent-1', to_node_id: 'group-a', kind: 'member' }],
      },
    });

    expect(draft.nodes[0].group).toMatchObject({
      speaking_mode: 'owner',
      owner_agent_id: 'agent-1',
    });
    expect(draftToOrchestrationDefinition(draft).nodes[0].group).toMatchObject({
      speaking_mode: 'owner',
      owner_agent_id: 'agent-1',
    });
  });
});
