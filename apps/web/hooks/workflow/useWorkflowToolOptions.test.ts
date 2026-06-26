import type { ToolPayload } from '@/lib/types';
import { buildWorkflowToolOptions } from '@/hooks/workflow/useWorkflowToolOptions';

describe('hooks/workflow/useWorkflowToolOptions', () => {
  const labels = {
    disabled: (name: string) => `${name} (disabled)`,
    unavailable: (name: string) => `${name} (unavailable)`,
  };

  it('sorts enabled tools and keeps a disabled current tool visible', () => {
    const tools: ToolPayload[] = [
      { name: 'z_tool', enabled: true, input_schema: { type: 'object' } },
      { name: 'a_tool', enabled: true },
      { name: 'legacy_tool', enabled: false },
    ];

    expect(buildWorkflowToolOptions({
      tools,
      currentToolName: ' legacy_tool ',
      labels,
    })).toEqual([
      { name: 'legacy_tool', label: 'legacy_tool (disabled)', disabled: true, inputSchema: undefined },
      { name: 'a_tool', label: 'a_tool', disabled: false, inputSchema: undefined },
      { name: 'z_tool', label: 'z_tool', disabled: false, inputSchema: { type: 'object' } },
    ]);
  });

  it('marks a missing current tool as unavailable', () => {
    expect(buildWorkflowToolOptions({
      tools: [{ name: 'enabled_tool', enabled: true }],
      currentToolName: 'missing_tool',
      labels,
    })).toEqual([
      { name: 'missing_tool', label: 'missing_tool (unavailable)', disabled: true, inputSchema: undefined },
      { name: 'enabled_tool', label: 'enabled_tool', disabled: false, inputSchema: undefined },
    ]);
  });
});
