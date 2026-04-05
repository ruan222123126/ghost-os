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
          { from_node_id: 'llm', to_node_id: 'agent' },
          { from_node_id: 'agent', to_node_id: 'end' },
        ],
      },
    };

    const draft = workflowDefinitionToDraft(source);

    expect(draft.schedule.mode).toBe('interval');
    expect(draft.nodes).toHaveLength(5);
    expect(draft.edges).toHaveLength(4);

    const definition = draftToWorkflowDefinition(draft);

    expect(definition).toEqual(source.workflow);
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
          { id: 'start', type: 'start', start: { inputs: undefined }, tool: undefined, llm: undefined, agent: undefined },
          { id: 'end', type: 'end', start: undefined, tool: undefined, llm: undefined, agent: undefined },
        ],
        edges: [{ from_node_id: 'start', to_node_id: 'end' }],
      },
    });
    expect(updatePayload).toEqual(createPayload);
  });
});
