'use client';

import { WorkflowScreenControlComposerModal } from '@/components/workflow/WorkflowScreenControlComposerModal';
import { WorkflowVariableAutocompleteField } from '@/components/workflow/WorkflowVariableAutocompleteField';
import { useWebLocale } from '@/lib/i18n/provider';
import {
  type WorkflowEditorKind,
  type WorkflowCanvasNodeDraft,
  type WorkflowToolArgumentRow,
  withToolName,
} from '@/lib/workflow-editor';
import { useWorkflowScreenComposerState } from '@/hooks/workflow/useWorkflowScreenComposerState';
import { useWorkflowToolOptions } from '@/hooks/workflow/useWorkflowToolOptions';
import {
  extractSchemaFields,
  type ToolSchemaField,
  useToolSchemaRowsState,
} from '@/components/workflow/workflowToolSchemaEditorState';
import type { WorkflowToolOption } from '@/hooks/workflow/useWorkflowToolOptions';

interface WorkflowCanvasToolNodeEditorProps {
  editorKind: WorkflowEditorKind;
  selectedNode: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

interface ToolNodeEditorState {
  composer: ReturnType<typeof useWorkflowScreenComposerState>;
  errorText: string;
  onRowValueChange: (index: number, value: string) => void;
  onSelectToolName: (toolName: string) => void;
  rows: WorkflowToolArgumentRow[];
  schemaFields: ToolSchemaField[];
  selectedToolName: string;
  toolOptions: WorkflowToolOption[];
  toolOptionsError: string;
  toolOptionsLoading: boolean;
}

export function WorkflowCanvasToolNodeEditor(props: WorkflowCanvasToolNodeEditorProps) {
  const editor = useToolNodeEditorState(props);
  return <ToolNodeEditorView editor={editor} editorKind={props.editorKind} />;
}

function useToolNodeEditorState(props: WorkflowCanvasToolNodeEditorProps): ToolNodeEditorState {
  const { selectedNode, onUpdateNode } = props;
  const selectedToolName = selectedNode.tool?.tool_name ?? '';
  const toolOptionsState = useWorkflowToolOptions(selectedToolName);
  const selectedTool = findToolOption(toolOptionsState.options, selectedToolName);
  const schemaFields = extractSchemaFields(selectedTool?.inputSchema, { toolName: selectedToolName });
  const schemaRows = useToolSchemaRowsState({ selectedNode, schemaFields, onUpdateNode });
  const composer = useWorkflowScreenComposerState({ selectedNode, selectedToolName, onUpdateNode });

  return {
    composer,
    errorText: schemaRows.errorText,
    onRowValueChange: schemaRows.onRowValueChange,
    onSelectToolName: createToolNameSelectHandler(selectedNode, onUpdateNode),
    rows: schemaRows.rows,
    schemaFields,
    selectedToolName,
    toolOptions: toolOptionsState.options,
    toolOptionsError: toolOptionsState.error,
    toolOptionsLoading: toolOptionsState.loading,
  };
}

function findToolOption(options: WorkflowToolOption[], selectedToolName: string): WorkflowToolOption | undefined {
  return options.find((option) => option.name === selectedToolName);
}

function createToolNameSelectHandler(
  selectedNode: WorkflowCanvasNodeDraft,
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void,
): (toolName: string) => void {
  return function selectToolName(toolName) {
    onUpdateNode(withToolName(selectedNode, toolName));
  };
}

function ToolNodeEditorView(props: {
  editor: ToolNodeEditorState;
  editorKind: WorkflowEditorKind;
}) {
  const { copy } = useWebLocale();
  const { editor, editorKind } = props;

  return (
    <div className="workflow-arch-prop-group">
      <ToolNameSelect
        selectedToolName={editor.selectedToolName}
        loading={editor.toolOptionsLoading}
        options={editor.toolOptions}
        onSelect={editor.onSelectToolName}
      />
      <ToolSchemaRows
        editorKind={editorKind}
        selectedToolName={editor.selectedToolName}
        schemaFields={editor.schemaFields}
        rows={editor.rows}
        onRowValueChange={editor.onRowValueChange}
      />
      <ScreenComposerOpenButton
        isScreenControlTool={editor.composer.isScreenControlTool}
        label={copy.workflow.screenComposerOpenButton}
        onOpen={editor.composer.onOpen}
      />
      <ToolEditorErrorNotes errors={[editor.toolOptionsError, editor.errorText]} />
      <ToolRuntimeVariableHint editorKind={editorKind} text={copy.workflow.runtimeVariableHint} />
      <ScreenComposerModal composer={editor.composer} />
    </div>
  );
}

function ScreenComposerOpenButton(props: {
  isScreenControlTool: boolean;
  label: string;
  onOpen: () => void;
}) {
  if (!props.isScreenControlTool) {
    return null;
  }
  return (
    <button type="button" className="workflow-arch-inline-button workflow-arch-screen-compose-entry" onClick={props.onOpen}>
      {props.label}
    </button>
  );
}

function ToolEditorErrorNotes(props: { errors: string[] }) {
  return (
    <>
      {props.errors.filter(Boolean).map((error, index) => (
        <p key={`${index}:${error}`} className="workflow-arch-field-note workflow-arch-field-note--error">{error}</p>
      ))}
    </>
  );
}

function ToolRuntimeVariableHint(props: { editorKind: WorkflowEditorKind; text: string }) {
  if (props.editorKind !== 'workflow') {
    return null;
  }
  return <p className="workflow-arch-field-note">{props.text}</p>;
}

function ScreenComposerModal(props: { composer: ToolNodeEditorState['composer'] }) {
  const { composer } = props;
  return (
    <WorkflowScreenControlComposerModal
      open={composer.open}
      steps={composer.steps}
      onClose={composer.onClose}
      onAppend={composer.onAppend}
      onMove={composer.onMove}
      onRemove={composer.onRemove}
      onUpdateStep={composer.onUpdateStep}
    />
  );
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
  editorKind: WorkflowEditorKind;
  selectedToolName: string;
  schemaFields: ToolSchemaField[];
  rows: WorkflowToolArgumentRow[];
  onRowValueChange: (index: number, value: string) => void;
}) {
  const { copy } = useWebLocale();
  const { editorKind, selectedToolName, schemaFields, rows, onRowValueChange } = props;

  return (
    <>
      <label className="workflow-arch-field-label">{copy.workflow.toolArgumentsJSON}</label>
      {schemaFields.length > 0 ? (
        <div className="workflow-arch-tool-rows">
          {schemaFields.map((field, index) => (
            <ToolSchemaRow
              key={field.key}
              editorKind={editorKind}
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
  editorKind: WorkflowEditorKind;
  field: ToolSchemaField;
  row: WorkflowToolArgumentRow | undefined;
  onValueChange: (value: string) => void;
}) {
  const { copy } = useWebLocale();
  const { editorKind, field, row, onValueChange } = props;
  const value = row?.value ?? '';

  return (
    <div className="workflow-arch-tool-row workflow-arch-tool-row--schema">
      <input type="text" readOnly value={field.key} className="workflow-arch-tool-key" />
      <ToolSchemaValueInput
        editorKind={editorKind}
        field={field}
        value={value}
        onValueChange={onValueChange}
      />
      <ToolSchemaRowNote field={field} inputRequired={copy.workflow.inputRequired} />
    </div>
  );
}

function ToolSchemaValueInput(props: {
  editorKind: WorkflowEditorKind;
  field: ToolSchemaField;
  value: string;
  onValueChange: (value: string) => void;
}) {
  const { copy } = useWebLocale();
  const { editorKind, field, value, onValueChange } = props;
  const disabled = field.valueType === 'null';
  const placeholder = toolFieldPlaceholder(field, copy);

  if (editorKind === 'workflow') {
    return (
      <WorkflowVariableAutocompleteField
        mode="input"
        value={value}
        disabled={disabled}
        placeholder={placeholder}
        onChange={onValueChange}
      />
    );
  }

  return (
    <input
      type="text"
      value={value}
      disabled={disabled}
      placeholder={placeholder}
      onChange={(event) => onValueChange(event.target.value)}
    />
  );
}

function ToolSchemaRowNote(props: { field: ToolSchemaField; inputRequired: string }) {
  const { field, inputRequired } = props;
  return (
    <p className="workflow-arch-field-note">
      {field.required ? `${inputRequired} · ` : ''}
      {field.schemaType}
      {field.description ? ` · ${field.description}` : ''}
    </p>
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
