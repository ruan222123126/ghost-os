import {
  draftToWorkflowCreatePayload,
  draftToWorkflowDefinition,
  draftToWorkflowUpdatePayload,
  workflowDefinitionToDraft,
} from '@/lib/workflow-editor/draft';
import type { WorkflowDefinitionImport } from '@/lib/workflow-editor/types';

describe('lib/workflow-editor/draft', () => {
  it('converts workflow definition to draft and back for all node types', () => {
    const source: WorkflowDefinitionImport = {
      scheduleType: 'interval',
      intervalSeconds: 90,
      workflow: {
        nodes: [
          {
            id: 'start',
            type: 'start',
            start: {
              inputs: [
                {
                  name: 'title',
                  type: 'string',
                  required: true,
                  default: 'Daily Sync',
                },
              ],
            },
          },
          {
            id: 'tool',
            type: 'tool',
            tool: {
              tool_name: 'web_search',
              arguments: { query: 'ghost-os' },
            },
          },
          {
            id: 'llm',
            type: 'llm',
            llm: {
              prompt: 'Summarize {{tool.output}}',
              system_prompt: 'You are concise.',
            },
          },
          {
            id: 'if-node',
            type: 'if',
            if: {
              source_node_id: 'llm',
              operator: 'contains',
              value: 'ok',
              true_node_id: 'loop-node',
              false_node_id: 'end',
            },
          },
          {
            id: 'loop-node',
            type: 'loop',
            loop: {
              max_iterations: 2,
              body_node_id: 'agent',
              exit_node_id: 'end',
            },
          },
          {
            id: 'agent',
            type: 'agent',
            agent: {
              message: 'Send summary to session',
            },
          },
          {
            id: 'end',
            type: 'end',
          },
        ],
        edges: [
          { from_node_id: 'start', to_node_id: 'tool' },
          { from_node_id: 'tool', to_node_id: 'llm' },
          { from_node_id: 'llm', to_node_id: 'if-node' },
          { from_node_id: 'if-node', to_node_id: 'loop-node' },
          { from_node_id: 'if-node', to_node_id: 'end' },
          { from_node_id: 'loop-node', to_node_id: 'agent' },
          { from_node_id: 'loop-node', to_node_id: 'end' },
          { from_node_id: 'agent', to_node_id: 'loop-node' },
        ],
      },
    };

    const draft = workflowDefinitionToDraft(source);

    expect(draft.schedule.mode).toBe('interval');
    expect(draft.nodes).toHaveLength(8);
    expect(draft.edges).toHaveLength(8);

    const definition = draftToWorkflowDefinition(draft);

    expect(definition.nodes).toHaveLength(source.workflow.nodes.length);
    expect(definition.edges).toHaveLength(source.workflow.edges.length);
    expect(definition.nodes).toEqual(
      expect.arrayContaining(source.workflow.nodes.map((node) => expect.objectContaining(node))),
    );
    expect(definition.edges).toEqual(
      expect.arrayContaining(source.workflow.edges.map((edge) => expect.objectContaining(edge))),
    );
  });

  it('builds create/update payload from draft schedule fields', () => {
    const draft = workflowDefinitionToDraft({
      scheduleType: 'cron',
      cronExpr: '*/5 * * * *',
      workflow: {
        nodes: [
          { id: 'start', type: 'start' },
          { id: 'end', type: 'end' },
        ],
        edges: [{ from_node_id: 'start', to_node_id: 'end' }],
      },
    });

    const createPayload = draftToWorkflowCreatePayload(draft);
    const updatePayload = draftToWorkflowUpdatePayload(draft);

    expect(createPayload).toEqual({
      task_kind: 'workflow',
      cron_expr: '*/5 * * * *',
      workflow: {
        nodes: [
          {
            id: 'start',
            type: 'start',
            start: { inputs: undefined },
            tool: undefined,
            llm: undefined,
            agent: undefined,
            if: undefined,
            loop: undefined,
          },
          {
            id: 'end',
            type: 'end',
            start: undefined,
            tool: undefined,
            llm: undefined,
            agent: undefined,
            if: undefined,
            loop: undefined,
          },
        ],
        edges: [{ from_node_id: 'start', to_node_id: 'end' }],
      },
    });
    expect(updatePayload).toEqual(createPayload);
  });

  it('expands backend loop node into fixed start/end pair and compiles back', () => {
    const draft = workflowDefinitionToDraft({
      scheduleType: 'interval',
      intervalSeconds: 120,
      workflow: {
        nodes: [
          { id: 'start', type: 'start' },
          { id: 'main-loop', type: 'loop', loop: { max_iterations: 3, body_node_id: 'tool-node', exit_node_id: 'end' } },
          { id: 'tool-node', type: 'tool', tool: { tool_name: 'script_exec', arguments: { command: 'pwd' } } },
          { id: 'end', type: 'end' },
        ],
        edges: [
          { from_node_id: 'start', to_node_id: 'main-loop' },
          { from_node_id: 'main-loop', to_node_id: 'tool-node' },
          { from_node_id: 'main-loop', to_node_id: 'end' },
          { from_node_id: 'tool-node', to_node_id: 'main-loop' },
        ],
      },
    });

    const loopStart = draft.nodes.find((node) => node.id === 'main-loop-start');
    const loopEnd = draft.nodes.find((node) => node.id === 'main-loop-end');

    expect(loopStart?.loop).toMatchObject({ role: 'start', loop_id: 'main-loop', max_iterations: 3 });
    expect(loopEnd?.loop).toMatchObject({ role: 'end', loop_id: 'main-loop' });
    expect(draft.edges).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ from_node_id: 'start', to_node_id: 'main-loop-start' }),
        expect.objectContaining({ from_node_id: 'main-loop-start', to_node_id: 'tool-node' }),
        expect.objectContaining({ from_node_id: 'tool-node', to_node_id: 'main-loop-end' }),
        expect.objectContaining({ from_node_id: 'main-loop-end', to_node_id: 'end' }),
      ]),
    );

    const definition = draftToWorkflowDefinition(draft);
    const loopNode = definition.nodes.find((node) => node.id === 'main-loop');

    expect(definition.nodes.some((node) => node.id === 'main-loop-end')).toBe(false);
    expect(loopNode).toMatchObject({
      id: 'main-loop',
      type: 'loop',
      loop: {
        max_iterations: 3,
        body_node_id: 'tool-node',
        exit_node_id: 'end',
      },
    });
  });

  it('keeps if node as regular single node payload', () => {
    const draft = workflowDefinitionToDraft({
      scheduleType: 'interval',
      intervalSeconds: 120,
      workflow: {
        nodes: [
          { id: 'start', type: 'start' },
          { id: 'if-node', type: 'if', if: { operator: 'contains', value: 'ok', true_node_id: 'tool-true', false_node_id: 'tool-false' } },
          { id: 'tool-true', type: 'tool', tool: { tool_name: 'script_exec', arguments: { command: 'echo true' } } },
          { id: 'tool-false', type: 'tool', tool: { tool_name: 'script_exec', arguments: { command: 'echo false' } } },
          { id: 'end', type: 'end' },
        ],
        edges: [
          { from_node_id: 'start', to_node_id: 'if-node' },
          { from_node_id: 'if-node', to_node_id: 'tool-true' },
          { from_node_id: 'if-node', to_node_id: 'tool-false' },
          { from_node_id: 'tool-true', to_node_id: 'end' },
          { from_node_id: 'tool-false', to_node_id: 'end' },
        ],
      },
    });

    const definition = draftToWorkflowDefinition(draft);
    const ifNode = definition.nodes.find((node) => node.id === 'if-node');

    expect(definition.nodes.some((node) => node.id === 'if-end')).toBe(false);
    expect(ifNode).toMatchObject({
      id: 'if-node',
      type: 'if',
      if: {
        operator: 'contains',
        value: 'ok',
        true_node_id: 'tool-true',
        false_node_id: 'tool-false',
      },
    });
    expect(definition.edges).toEqual(
      expect.arrayContaining([
        { from_node_id: 'if-node', to_node_id: 'tool-true' },
        { from_node_id: 'if-node', to_node_id: 'tool-false' },
      ]),
    );
  });

  it('hydrates screen_control composer steps from workflow_steps arguments', () => {
    const draft = workflowDefinitionToDraft({
      scheduleType: 'interval',
      intervalSeconds: 60,
      workflow: {
        nodes: [
          { id: 'start', type: 'start' },
          {
            id: 'tool',
            type: 'tool',
            tool: {
              tool_name: 'screen_control',
              arguments: {
                mode: 'atomic',
                workflow_steps: [
                  { action: 'screenshot' },
                  { action: 'find_icon', params: { template_path: '/tmp/icon.png', threshold: 0.92 } },
                  { action: 'click', params: { x: 120, y: 240 } },
                ],
              },
            },
          },
          { id: 'end', type: 'end' },
        ],
        edges: [
          { from_node_id: 'start', to_node_id: 'tool' },
          { from_node_id: 'tool', to_node_id: 'end' },
        ],
      },
    });

    const toolNode = draft.nodes.find((node) => node.id === 'tool');
    expect(toolNode?.ui.screenControlComposer?.steps).toEqual([
      { action: 'screenshot' },
      { action: 'find_icon', params: { template_path: '/tmp/icon.png', threshold: 0.92 } },
      { action: 'click', params: { x: 120, y: 240 } },
    ]);
  });

  it('hydrates single screen_control action into composer queue when workflow_steps is absent', () => {
    const draft = workflowDefinitionToDraft({
      scheduleType: 'interval',
      intervalSeconds: 60,
      workflow: {
        nodes: [
          { id: 'start', type: 'start' },
          {
            id: 'tool',
            type: 'tool',
            tool: {
              tool_name: 'screen_control',
              arguments: {
                mode: 'atomic',
                action: 'click_icon',
                params: { x: 16, y: 24 },
              },
            },
          },
          { id: 'end', type: 'end' },
        ],
        edges: [
          { from_node_id: 'start', to_node_id: 'tool' },
          { from_node_id: 'tool', to_node_id: 'end' },
        ],
      },
    });

    const toolNode = draft.nodes.find((node) => node.id === 'tool');
    expect(toolNode?.ui.screenControlComposer?.steps).toEqual([
      { action: 'click', params: { x: 16, y: 24 } },
    ]);
  });
});
