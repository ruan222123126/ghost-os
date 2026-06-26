import {
  WORKFLOW_AUTOCOMPLETE_OPTIONS,
  WORKFLOW_FIND_ICON_VARIABLE,
  applyVariableOption,
  filterWorkflowVariableOptions,
  resolveVariableAutocompleteKeyAction,
  resolveVariableTriggerRange,
} from '@/components/workflow/workflowVariableAutocomplete';

describe('components/workflow/workflowVariableAutocomplete', () => {
  it('detects { and ${ triggers for the find_icon variable', () => {
    expect(resolveVariableTriggerRange('hello {fin', 10)).toEqual({
      start: 6,
      end: 10,
      query: 'fin',
    });
    expect(resolveVariableTriggerRange('hello ${find', 12)).toEqual({
      start: 6,
      end: 12,
      query: 'find',
    });
  });

  it('filters and applies the single find_icon option', () => {
    expect(filterWorkflowVariableOptions(WORKFLOW_AUTOCOMPLETE_OPTIONS, 'find_icon')).toEqual(WORKFLOW_AUTOCOMPLETE_OPTIONS);
    const range = resolveVariableTriggerRange('prefix {fi suffix', 10);
    expect(applyVariableOption('prefix {fi suffix', range!, WORKFLOW_AUTOCOMPLETE_OPTIONS[0])).toEqual({
      value: `prefix ${WORKFLOW_FIND_ICON_VARIABLE} suffix`,
      caretPosition: 7 + WORKFLOW_FIND_ICON_VARIABLE.length,
    });
  });

  it('resolves keyboard actions from dropdown state', () => {
    expect(resolveVariableAutocompleteKeyAction({
      dropdownOpen: true,
      key: 'ArrowDown',
      optionCount: 1,
    })).toBe('move-next');
    expect(resolveVariableAutocompleteKeyAction({
      dropdownOpen: true,
      key: 'Enter',
      optionCount: 1,
    })).toBe('commit');
    expect(resolveVariableAutocompleteKeyAction({
      dropdownOpen: false,
      key: 'Escape',
      optionCount: 0,
    })).toBe('clear-dismissed');
    expect(resolveVariableAutocompleteKeyAction({
      dropdownOpen: true,
      key: 'ArrowDown',
      optionCount: 0,
    })).toBeUndefined();
  });
});
