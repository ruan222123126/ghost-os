'use client';

import { WorkflowVariableAutocompleteField } from '@/components/workflow/WorkflowVariableAutocompleteField';
import type { WorkflowEditorKind } from '@/lib/workflow-editor';

interface WorkflowTemplateEnabledFieldProps {
  editorKind: WorkflowEditorKind;
  mode: 'input' | 'textarea';
  value: string;
  rows?: number;
  disabled?: boolean;
  placeholder?: string;
  onChange: (value: string) => void;
}

export function WorkflowTemplateEnabledField(props: WorkflowTemplateEnabledFieldProps) {
  const { editorKind, mode, value, rows, disabled = false, placeholder, onChange } = props;

  if (editorKind === 'workflow') {
    return (
      <WorkflowVariableAutocompleteField
        mode={mode}
        value={value}
        rows={rows}
        disabled={disabled}
        placeholder={placeholder}
        onChange={onChange}
      />
    );
  }

  if (mode === 'textarea') {
    return (
      <textarea
        rows={rows}
        disabled={disabled}
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
      />
    );
  }

  return (
    <input
      type="text"
      disabled={disabled}
      value={value}
      placeholder={placeholder}
      onChange={(event) => onChange(event.target.value)}
    />
  );
}
