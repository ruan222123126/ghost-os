'use client';

import type { KeyboardEvent, MouseEvent, RefObject, SyntheticEvent } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';
import { useWorkflowVariableAutocomplete, type WorkflowVariableAutocompleteState } from '@/components/workflow/useWorkflowVariableAutocomplete';

interface WorkflowVariableAutocompleteFieldProps {
  mode: 'input' | 'textarea';
  value: string;
  rows?: number;
  placeholder?: string;
  disabled?: boolean;
  readOnly?: boolean;
  onChange: (value: string) => void;
}

type FieldElement = HTMLInputElement | HTMLTextAreaElement;

export function WorkflowVariableAutocompleteField(props: WorkflowVariableAutocompleteFieldProps) {
  const { locale } = useWebLocale();
  const state = useWorkflowVariableAutocomplete(props);

  return (
    <div className="workflow-variable-autocomplete" data-testid="workflow-variable-autocomplete">
      <AutocompleteFieldInput {...props} state={state} />
      {state.dropdownOpen ? (
        <div className="workflow-variable-autocomplete__dropdown" role="listbox">
          {state.filteredOptions.length > 0 ? (
            state.filteredOptions.map((option, index) => (
              <button
                key={option.token}
                type="button"
                className={`workflow-variable-autocomplete__option ${index === state.activeIndex ? 'workflow-variable-autocomplete__option--active' : ''}`}
                onMouseDown={state.handleOptionMouseDown}
                onClick={() => state.handleOptionClick(option)}
              >
                <span className="workflow-variable-autocomplete__option-head">
                  <strong>{option.label}</strong>
                  <span>{locale === 'zh-CN' ? '内置' : 'Built-in'}</span>
                </span>
                <code>{option.token}</code>
              </button>
            ))
          ) : (
            <p className="workflow-variable-autocomplete__empty">{locale === 'zh-CN' ? '无匹配变量' : 'No matching variables'}</p>
          )}
        </div>
      ) : null}
    </div>
  );
}

function AutocompleteFieldInput(
  props: WorkflowVariableAutocompleteFieldProps & { state: WorkflowVariableAutocompleteState },
) {
  const { mode, value, rows, placeholder, disabled, readOnly, state } = props;
  const sharedProps = {
    value,
    placeholder,
    disabled,
    readOnly,
    onFocus: state.handleFocus,
    onBlur: state.handleBlur,
    onSelect: (event: SyntheticEvent<FieldElement>) => state.syncCaretPosition(event.currentTarget),
    onClick: (event: MouseEvent<FieldElement>) => state.syncCaretPosition(event.currentTarget),
    onKeyUp: (event: KeyboardEvent<FieldElement>) => state.syncCaretPosition(event.currentTarget),
    onKeyDown: state.handleKeyDown,
    className: 'workflow-variable-autocomplete__field',
  } as const;

  if (mode === 'textarea') {
    return (
      <textarea
        {...sharedProps}
        ref={state.fieldRef as RefObject<HTMLTextAreaElement>}
        rows={rows}
        onChange={(event) => state.handleChange(event.currentTarget)}
      />
    );
  }

  return (
    <input
      {...sharedProps}
      ref={state.fieldRef as RefObject<HTMLInputElement>}
      type="text"
      onChange={(event) => state.handleChange(event.currentTarget)}
    />
  );
}
