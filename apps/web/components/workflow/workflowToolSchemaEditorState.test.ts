import { extractSchemaFields } from '@/components/workflow/workflowToolSchemaEditorState';

describe('components/workflow/workflowToolSchemaEditorState', () => {
  it('hides mode and top-level display_id for screen_control schema rows', () => {
    const fields = extractSchemaFields(
      {
        type: 'object',
        required: ['action'],
        properties: {
          action: { type: 'string' },
          mode: { type: 'string' },
          params: { type: 'object' },
          display_id: { type: 'integer' },
        },
      },
      { toolName: 'screen_control' },
    );

    expect(fields.map((field) => field.key)).toEqual(['action', 'params']);
    expect(fields.find((field) => field.key === 'action')?.required).toBe(true);
  });

  it('keeps mode and display_id for non-screen_control tools', () => {
    const fields = extractSchemaFields(
      {
        type: 'object',
        properties: {
          mode: { type: 'string' },
          display_id: { type: 'integer' },
        },
      },
      { toolName: 'script_exec' },
    );

    expect(fields.map((field) => field.key)).toEqual(['display_id', 'mode']);
  });
});
