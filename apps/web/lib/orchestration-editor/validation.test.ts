import { createEmptyOrchestrationDraft } from '@/lib/orchestration-editor/draft';
import { validateOrchestrationDraft } from '@/lib/orchestration-editor/validation';
import type { WorkflowCanvasDraft } from '@/lib/workflow-editor/types';

describe('lib/orchestration-editor/validation', () => {
  it('accepts an empty draft so a new orchestration resource can be created before editing', () => {
    const result = validateOrchestrationDraft(createEmptyOrchestrationDraft());

    expect(result.valid).toBe(true);
    expect(result.errors).toEqual([]);
  });

  it('accepts a single group entry without start/end nodes', () => {
    const draft: WorkflowCanvasDraft = {
      mode: 'edit',
      schedule: { mode: 'interval', intervalSeconds: '60', cronExpr: '' },
      nodes: [
        {
          id: 'group-a',
          type: 'group',
          position: { x: 0, y: 0 },
          ui: { toolArgumentsMode: 'kv' },
          group: { title: 'Group A', shared_context: '', speaking_mode: 'sequential', owner_agent_id: '', max_rounds: 1 },
        },
        {
          id: 'agent-1',
          type: 'agent',
          position: { x: 0, y: -120 },
          ui: { toolArgumentsMode: 'kv' },
          agent: { title: 'Alpha', message: 'hello' },
        },
      ],
      edges: [
        { id: 'edge-member', from_node_id: 'agent-1', to_node_id: 'group-a', kind: 'member' },
      ],
    };

    const result = validateOrchestrationDraft(draft);

    expect(result.valid).toBe(true);
    expect(result.errors).toEqual([]);
  });

  it('rejects removed start/end nodes', () => {
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
          agent: { title: 'Alpha', message: 'hello' },
        },
        { id: 'end-node', type: 'end', position: { x: 400, y: 0 }, ui: { toolArgumentsMode: 'kv' } },
      ],
      edges: [
        { id: 'edge-start', from_node_id: 'start-node', to_node_id: 'group-a', kind: 'control' },
        { id: 'edge-end', from_node_id: 'group-a', to_node_id: 'end-node', kind: 'control' },
        { id: 'edge-member', from_node_id: 'agent-1', to_node_id: 'group-a', kind: 'member' },
      ],
    };

    const result = validateOrchestrationDraft(draft);

    expect(result.valid).toBe(false);
    expect(result.errors).toEqual(expect.arrayContaining([
      'orchestration node type "start" is no longer supported',
      'orchestration node type "end" is no longer supported',
      'orchestration control edge "start-node" -> "group-a" is invalid',
      'orchestration control edge "group-a" -> "end-node" is invalid',
    ]));
  });

  it('rejects multiple entry and exit groups', () => {
    const draft: WorkflowCanvasDraft = {
      mode: 'edit',
      schedule: { mode: 'interval', intervalSeconds: '60', cronExpr: '' },
      nodes: [
        {
          id: 'group-a',
          type: 'group',
          position: { x: 0, y: 0 },
          ui: { toolArgumentsMode: 'kv' },
          group: { title: 'Group A', shared_context: '', speaking_mode: 'sequential', owner_agent_id: '', max_rounds: 1 },
        },
        {
          id: 'group-b',
          type: 'group',
          position: { x: 220, y: 0 },
          ui: { toolArgumentsMode: 'kv' },
          group: { title: 'Group B', shared_context: '', speaking_mode: 'parallel', owner_agent_id: '', max_rounds: 2 },
        },
        {
          id: 'agent-1',
          type: 'agent',
          position: { x: 0, y: -120 },
          ui: { toolArgumentsMode: 'kv' },
          agent: { title: 'Alpha', message: 'hello' },
        },
        {
          id: 'agent-2',
          type: 'agent',
          position: { x: 220, y: -120 },
          ui: { toolArgumentsMode: 'kv' },
          agent: { title: 'Beta', message: 'world' },
        },
      ],
      edges: [
        { id: 'edge-member-a', from_node_id: 'agent-1', to_node_id: 'group-a', kind: 'member' },
        { id: 'edge-member-b', from_node_id: 'agent-2', to_node_id: 'group-b', kind: 'member' },
      ],
    };

    const result = validateOrchestrationDraft(draft);

    expect(result.valid).toBe(false);
    expect(result.errors).toEqual(expect.arrayContaining([
      'orchestration requires exactly 1 entry group',
      'orchestration requires exactly 1 exit group',
      'orchestration node "group-b" is disconnected from control flow',
    ]));
  });

  it('accepts tool_allowlist entries for configured but globally disabled tools', () => {
    const draft: WorkflowCanvasDraft = {
      mode: 'edit',
      schedule: { mode: 'interval', intervalSeconds: '60', cronExpr: '' },
      nodes: [
        {
          id: 'group-a',
          type: 'group',
          position: { x: 0, y: 0 },
          ui: { toolArgumentsMode: 'kv' },
          group: { title: 'Group A', shared_context: '', speaking_mode: 'sequential', owner_agent_id: '', max_rounds: 1 },
        },
        {
          id: 'agent-1',
          type: 'agent',
          position: { x: 0, y: -120 },
          ui: { toolArgumentsMode: 'kv' },
          agent: {
            title: 'Alpha',
            message: 'hello',
            runtime_overrides: {
              tool_allowlist_only: true,
              tool_allowlist: ['ghost_tool'],
            },
          },
        },
      ],
      edges: [
        { id: 'edge-member', from_node_id: 'agent-1', to_node_id: 'group-a', kind: 'member' },
      ],
    };

    const result = validateOrchestrationDraft(draft, {
      agentRuntimeCatalog: {
        activeProvider: 'openai-main',
        providers: [],
        tools: [
          { name: 'ghost_tool', enabled: false },
        ],
      },
    });

    expect(result.valid).toBe(true);
    expect(result.errors).toEqual([]);
  });

  it('rejects missing preset references after preset catalog loads', () => {
    const draft: WorkflowCanvasDraft = {
      mode: 'edit',
      schedule: { mode: 'interval', intervalSeconds: '60', cronExpr: '' },
      nodes: [
        {
          id: 'group-a',
          type: 'group',
          position: { x: 0, y: 0 },
          ui: { toolArgumentsMode: 'kv' },
          group: { title: 'Group A', shared_context: '', speaking_mode: 'sequential', owner_agent_id: '', max_rounds: 1 },
        },
        {
          id: 'agent-1',
          type: 'agent',
          position: { x: 0, y: -120 },
          ui: { toolArgumentsMode: 'kv' },
          agent: {
            title: 'Alpha',
            message: 'hello',
            runtime_overrides: {
              preset_id: 'missing-preset',
            },
          },
        },
      ],
      edges: [
        { id: 'edge-member', from_node_id: 'agent-1', to_node_id: 'group-a', kind: 'member' },
      ],
    };

    const result = validateOrchestrationDraft(draft, {
      agentRuntimeCatalog: {
        activeProvider: 'openai-main',
        providers: [],
        tools: [],
      },
      presets: [],
    });

    expect(result.valid).toBe(false);
    expect(result.errors).toEqual(expect.arrayContaining([
      'orchestration agent node "agent-1" preset_id "missing-preset" is not available',
    ]));
  });

  it('rejects owner mode without owner_agent_id', () => {
    const draft: WorkflowCanvasDraft = {
      mode: 'edit',
      schedule: { mode: 'interval', intervalSeconds: '60', cronExpr: '' },
      nodes: [
        {
          id: 'group-a',
          type: 'group',
          position: { x: 0, y: 0 },
          ui: { toolArgumentsMode: 'kv' },
          group: { title: 'Group A', shared_context: '', speaking_mode: 'owner', owner_agent_id: '', max_rounds: 1 },
        },
        {
          id: 'agent-1',
          type: 'agent',
          position: { x: 0, y: -120 },
          ui: { toolArgumentsMode: 'kv' },
          agent: { title: 'Owner', message: 'hello' },
        },
      ],
      edges: [
        { id: 'edge-member', from_node_id: 'agent-1', to_node_id: 'group-a', kind: 'member' },
      ],
    };

    const result = validateOrchestrationDraft(draft);
    expect(result.valid).toBe(false);
    expect(result.errors).toContain('orchestration group node "group-a" requires owner_agent_id in owner mode');
  });

  it('rejects owner mode when owner is not a member', () => {
    const draft: WorkflowCanvasDraft = {
      mode: 'edit',
      schedule: { mode: 'interval', intervalSeconds: '60', cronExpr: '' },
      nodes: [
        {
          id: 'group-a',
          type: 'group',
          position: { x: 0, y: 0 },
          ui: { toolArgumentsMode: 'kv' },
          group: { title: 'Group A', shared_context: '', speaking_mode: 'owner', owner_agent_id: 'agent-2', max_rounds: 1 },
        },
        {
          id: 'agent-1',
          type: 'agent',
          position: { x: 0, y: -120 },
          ui: { toolArgumentsMode: 'kv' },
          agent: { title: 'Owner', message: 'hello' },
        },
      ],
      edges: [
        { id: 'edge-member', from_node_id: 'agent-1', to_node_id: 'group-a', kind: 'member' },
      ],
    };

    const result = validateOrchestrationDraft(draft);
    expect(result.valid).toBe(false);
    expect(result.errors).toContain('orchestration group node "group-a" owner_agent_id "agent-2" must be an existing member');
  });
});
