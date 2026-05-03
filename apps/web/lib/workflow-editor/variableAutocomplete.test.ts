import {
  applyVariableOption,
  filterWorkflowVariableOptions,
  resolveVariableTriggerRange,
  type WorkflowVariableOption,
} from '@/lib/workflow-editor';

describe('lib/workflow-editor/variableAutocomplete', () => {
  it('detects both { and ${ trigger fragments', () => {
    expect(resolveVariableTriggerRange('hello {inp', 10)).toEqual({
      start: 6,
      end: 10,
      query: 'inp',
      trigger: '{',
    });

    expect(resolveVariableTriggerRange('hello ${last', 12)).toEqual({
      start: 6,
      end: 12,
      query: 'last',
      trigger: '${',
    });
  });

  it('replaces the active fragment with the chosen token and returns the next caret', () => {
    const range = resolveVariableTriggerRange('prefix {las suffix', 11);
    expect(range).toBeDefined();

    const result = applyVariableOption(
      'prefix {las suffix',
      range!,
      createOption('${last_output}', 'last output', 'builtin'),
    );

    expect(result).toEqual({
      value: 'prefix ${last_output} suffix',
      caretPosition: 21,
    });
  });

  it('filters suggestions by token path variable name and node id without case sensitivity', () => {
    const options = [
      createOption('${inputs.customer_name}', 'customer_name', 'start', 'inputs.customer_name customer_name customer label string'),
      createOption('${nodes.Tool-Read.status}', 'Tool-Read status', 'upstream', 'nodes.tool-read.status Tool-Read'),
      createOption('${last_output}', 'last output', 'builtin', 'last_output'),
    ];

    expect(filterWorkflowVariableOptions(options, 'CUSTOMER')).toEqual([options[0]]);
    expect(filterWorkflowVariableOptions(options, 'tool-read')).toEqual([options[1]]);
    expect(filterWorkflowVariableOptions(options, 'nodes.tool-read.status')).toEqual([options[1]]);
  });
});

function createOption(
  token: string,
  label: string,
  section: WorkflowVariableOption['section'],
  searchText = `${token} ${label}`,
): WorkflowVariableOption {
  return {
    token,
    label,
    section,
    searchText: searchText.toLowerCase(),
  };
}
