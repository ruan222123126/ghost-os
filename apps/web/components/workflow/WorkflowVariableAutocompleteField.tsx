'use client';

import type {
  KeyboardEvent,
  MouseEvent,
  RefObject,
  SyntheticEvent,
} from 'react';
import { useWebLocale } from '@/lib/i18n/provider';
import type {
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowVariableOption,
} from '@/lib/workflow-editor';
import {
  useWorkflowVariableAutocomplete,
  type WorkflowVariableAutocompleteState,
} from './useWorkflowVariableAutocomplete';

interface WorkflowVariableAutocompleteFieldProps {
  draft: WorkflowCanvasDraft;
  selectedNode: WorkflowCanvasNodeDraft;
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
        <AutocompleteOptions
          locale={locale}
          activeIndex={state.activeIndex}
          options={state.filteredOptions}
          onMouseDown={state.handleOptionMouseDown}
          onClick={state.handleOptionClick}
        />
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

function AutocompleteOptions(props: {
  locale: string;
  activeIndex: number;
  options: WorkflowVariableOption[];
  onMouseDown: (event: MouseEvent<HTMLButtonElement>) => void;
  onClick: (option: WorkflowVariableOption) => void;
}) {
  const { locale, activeIndex, options, onMouseDown, onClick } = props;
  return (
    <div className="workflow-variable-autocomplete__dropdown" role="listbox">
      {options.length > 0 ? (
        options.map((option, index) => (
          <button
            key={`${option.token}:${option.section}`}
            type="button"
            className={`workflow-variable-autocomplete__option ${index === activeIndex ? 'workflow-variable-autocomplete__option--active' : ''}`}
            onMouseDown={onMouseDown}
            onClick={() => onClick(option)}
          >
            <span className="workflow-variable-autocomplete__option-head">
              <strong>{option.label}</strong>
              <span>{sectionLabel(option.section, locale)}</span>
            </span>
            <code>{option.token}</code>
          </button>
        ))
      ) : (
        <p className="workflow-variable-autocomplete__empty">{emptyLabel(locale)}</p>
      )}
    </div>
  );
}

function sectionLabel(section: WorkflowVariableOption['section'], locale: string): string {
  if (locale === 'zh-CN') {
    if (section === 'start') {
      return '开始';
    }
    if (section === 'builtin') {
      return '内置';
    }
    return '上游';
  }
  if (section === 'start') {
    return 'Start';
  }
  if (section === 'builtin') {
    return 'Built-in';
  }
  return 'Upstream';
}

function emptyLabel(locale: string): string {
  return locale === 'zh-CN' ? '无匹配变量' : 'No matching variables';
}
