import type {
  VariableTriggerRange,
  WorkflowVariableOption,
} from '@/lib/workflow-editor/types';

const VARIABLE_QUERY_PATTERN = /^[A-Za-z0-9._-]*$/;

export interface VariableAutocompleteApplyResult {
  value: string;
  caretPosition: number;
}

export function resolveVariableTriggerRange(
  value: string,
  caretPosition: number,
): VariableTriggerRange | undefined {
  const safeCaretPosition = clampCaretPosition(value, caretPosition);
  if (safeCaretPosition === 0) {
    return undefined;
  }

  for (let index = safeCaretPosition - 1; index >= 0; index -= 1) {
    const current = value[index];
    if (current === '{') {
      const query = value.slice(index + 1, safeCaretPosition);
      if (!VARIABLE_QUERY_PATTERN.test(query)) {
        return undefined;
      }

      const triggeredByTemplate = index > 0 && value[index - 1] === '$';
      return {
        start: triggeredByTemplate ? index - 1 : index,
        end: safeCaretPosition,
        query,
        trigger: triggeredByTemplate ? '${' : '{',
      };
    }

    if (!VARIABLE_QUERY_PATTERN.test(current)) {
      return undefined;
    }
  }

  return undefined;
}

export function filterWorkflowVariableOptions(
  options: WorkflowVariableOption[],
  query: string,
): WorkflowVariableOption[] {
  const normalizedQuery = query.trim().toLowerCase();
  if (normalizedQuery.length === 0) {
    return options;
  }

  return options.filter((option) => option.searchText.includes(normalizedQuery));
}

export function applyVariableOption(
  value: string,
  range: VariableTriggerRange,
  option: WorkflowVariableOption,
): VariableAutocompleteApplyResult {
  const nextValue = `${value.slice(0, range.start)}${option.token}${value.slice(range.end)}`;
  return {
    value: nextValue,
    caretPosition: range.start + option.token.length,
  };
}

function clampCaretPosition(value: string, caretPosition: number): number {
  if (!Number.isFinite(caretPosition)) {
    return value.length;
  }
  if (caretPosition < 0) {
    return 0;
  }
  if (caretPosition > value.length) {
    return value.length;
  }
  return caretPosition;
}
