'use client';

import { useEffect, useMemo, useState } from 'react';
import { WorkflowScreenControlComposerModal } from '@/components/workflow/WorkflowScreenControlComposerModal';
import { useWebLocale } from '@/lib/i18n/provider';
import {
  appendScreenControlComposerStep,
  moveScreenControlComposerStep,
  removeScreenControlComposerStep,
  syncScreenControlComposerStepsToToolArguments,
  updateScreenControlComposerStep,
  type ScreenControlAtomicAction,
  type ScreenControlComposerStep,
  type WorkflowCanvasNodeDraft,
  type WorkflowToolArgumentRow,
  withScreenControlComposerSteps,
  withToolName,
} from '@/lib/workflow-editor';
import { useWorkflowToolOptions } from '@/components/workflow/useWorkflowToolOptions';
import {
  extractSchemaFields,
  type ToolSchemaField,
  useToolSchemaRowsState,
} from '@/components/workflow/workflowToolSchemaEditorState';

interface WorkflowCanvasToolNodeEditorProps {
  selectedNode: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

interface ScreenComposerState {
  open: boolean;
  steps: ScreenControlComposerStep[];
  isScreenControlTool: boolean;
  onOpen: () => void;
  onClose: () => void;
  onAppend: (action: ScreenControlAtomicAction) => void;
  onMove: (index: number, direction: 'up' | 'down') => void;
  onRemove: (index: number) => void;
  onUpdateStep: (index: number, step: ScreenControlComposerStep) => void;
}

export function WorkflowCanvasToolNodeEditor(props: WorkflowCanvasToolNodeEditorProps) {
  const { copy } = useWebLocale();
  const { selectedNode, onUpdateNode } = props;
  const selectedToolName = selectedNode.tool?.tool_name ?? '';
  const { options: toolOptions, loading: toolOptionsLoading, error: toolOptionsError } = useWorkflowToolOptions(selectedToolName);
  const selectedTool = useMemo(
    () => toolOptions.find((option) => option.name === selectedToolName),
    [selectedToolName, toolOptions],
  );
  const schemaFields = useMemo(
    () => extractSchemaFields(selectedTool?.inputSchema, { toolName: selectedToolName }),
    [selectedTool?.inputSchema, selectedToolName],
  );
  const { rows, errorText, onRowValueChange } = useToolSchemaRowsState({ selectedNode, schemaFields, onUpdateNode });
  const composer = useScreenComposerState({ selectedNode, selectedToolName, onUpdateNode });

  return (
    <div className="workflow-arch-prop-group">
      <ToolNameSelect
        selectedToolName={selectedToolName}
        loading={toolOptionsLoading}
        options={toolOptions}
        onSelect={(toolName) => onUpdateNode(withToolName(selectedNode, toolName))}
      />
      <ToolSchemaRows
        selectedToolName={selectedToolName}
        schemaFields={schemaFields}
        rows={rows}
        onRowValueChange={onRowValueChange}
      />
      {composer.isScreenControlTool ? (
        <button type="button" className="workflow-arch-inline-button workflow-arch-screen-compose-entry" onClick={composer.onOpen}>
          {copy.workflow.screenComposerOpenButton}
        </button>
      ) : null}
      {toolOptionsError ? <p className="workflow-arch-field-note workflow-arch-field-note--error">{toolOptionsError}</p> : null}
      {errorText ? <p className="workflow-arch-field-note workflow-arch-field-note--error">{errorText}</p> : null}
      <p className="workflow-arch-field-note">{copy.workflow.runtimeVariableHint}</p>
      <WorkflowScreenControlComposerModal
        open={composer.open}
        steps={composer.steps}
        onClose={composer.onClose}
        onAppend={composer.onAppend}
        onMove={composer.onMove}
        onRemove={composer.onRemove}
        onUpdateStep={composer.onUpdateStep}
      />
    </div>
  );
}

function useScreenComposerState(options: {
  selectedNode: WorkflowCanvasNodeDraft;
  selectedToolName: string;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}): ScreenComposerState {
  const { selectedNode, selectedToolName, onUpdateNode } = options;
  const [open, setOpen] = useState(false);
  const isScreenControlTool = selectedToolName.trim() === 'screen_control';
  const steps = selectedNode.ui.screenControlComposer?.steps ?? [];

  useEffect(() => {
    if (isScreenControlTool) {
      return;
    }
    setOpen(false);
  }, [isScreenControlTool]);

  const updateSteps = (nextSteps: ScreenControlComposerStep[]) => {
    const nodeWithSteps = withScreenControlComposerSteps(selectedNode, nextSteps);
    onUpdateNode(syncScreenControlComposerStepsToToolArguments(nodeWithSteps, nextSteps));
  };

  const updateStepAt = (index: number, step: ScreenControlComposerStep) => {
    updateSteps(updateScreenControlComposerStep(steps, index, step));
  };

  return {
    open: isScreenControlTool && open,
    steps,
    isScreenControlTool,
    onOpen: () => setOpen(true),
    onClose: () => setOpen(false),
    onAppend: (action) => updateSteps(appendScreenControlComposerStep(steps, action)),
    onMove: (index, direction) => updateSteps(moveScreenControlComposerStep(steps, index, direction)),
    onRemove: (index) => updateSteps(removeScreenControlComposerStep(steps, index)),
    onUpdateStep: updateStepAt,
  };
}

function ToolNameSelect(props: {
  selectedToolName: string;
  loading: boolean;
  options: { name: string; label: string; disabled: boolean }[];
  onSelect: (toolName: string) => void;
}) {
  const { copy } = useWebLocale();
  const { selectedToolName, loading, options, onSelect } = props;

  return (
    <>
      <label className="workflow-arch-field-label">{copy.workflow.toolName}</label>
      <select value={selectedToolName} onChange={(event) => onSelect(event.target.value)}>
        <option value="">{loading ? copy.workflow.toolNameLoading : copy.workflow.toolNameSelectPlaceholder}</option>
        {options.map((option) => (
          <option key={option.name} value={option.name} disabled={option.disabled}>{option.label}</option>
        ))}
      </select>
    </>
  );
}

function ToolSchemaRows(props: {
  selectedToolName: string;
  schemaFields: ToolSchemaField[];
  rows: WorkflowToolArgumentRow[];
  onRowValueChange: (index: number, value: string) => void;
}) {
  const { copy } = useWebLocale();
  const { selectedToolName, schemaFields, rows, onRowValueChange } = props;

  return (
    <>
      <label className="workflow-arch-field-label">{copy.workflow.toolArgumentsJSON}</label>
      {schemaFields.length > 0 ? (
        <div className="workflow-arch-tool-rows">
          {schemaFields.map((field, index) => (
            <ToolSchemaRow
              key={field.key}
              field={field}
              row={rows[index]}
              onValueChange={(value) => onRowValueChange(index, value)}
            />
          ))}
        </div>
      ) : (
        <p className="workflow-arch-field-note">{toolArgumentsEmptyStateText(copy, selectedToolName)}</p>
      )}
    </>
  );
}

function ToolSchemaRow(props: {
  field: ToolSchemaField;
  row: WorkflowToolArgumentRow | undefined;
  onValueChange: (value: string) => void;
}) {
  const { copy } = useWebLocale();
  const { field, row, onValueChange } = props;
  const value = row?.value ?? '';

  return (
    <div className="workflow-arch-tool-row workflow-arch-tool-row--schema">
      <input type="text" readOnly value={field.key} className="workflow-arch-tool-key" />
      <input
        type="text"
        value={value}
        disabled={field.valueType === 'null'}
        placeholder={toolFieldPlaceholder(field, copy)}
        onChange={(event) => onValueChange(event.target.value)}
      />
      <p className="workflow-arch-field-note">
        {field.required ? `${copy.workflow.inputRequired} · ` : ''}
        {field.schemaType}
        {field.description ? ` · ${field.description}` : ''}
      </p>
    </div>
  );
}

function toolFieldPlaceholder(field: ToolSchemaField, copy: ReturnType<typeof useWebLocale>['copy']): string {
  if (field.valueType === 'object') {
    return '{"key":"value"}';
  }
  if (field.valueType === 'array') {
    return '["item"]';
  }
  if (field.valueType === 'boolean') {
    return 'true | false';
  }
  return copy.workflow.toolArgumentsKVValue;
}

function toolArgumentsEmptyStateText(copy: ReturnType<typeof useWebLocale>['copy'], toolName: string): string {
  if (toolName.trim().length === 0) {
    return copy.workflow.toolArgumentsSchemaSelectTool;
  }
  return copy.workflow.toolArgumentsSchemaMissing;
}
