'use client';

import { useEffect, useState, type KeyboardEvent } from 'react';
import { WorkflowStartInputModal } from '@/components/workflow/WorkflowStartInputModal';
import { useWebLocale } from '@/lib/i18n/provider';
import type { WorkflowInputVariable } from '@/lib/types';
import { INPUT_NAME_MAX_LENGTH } from '@/lib/workflow-editor/constants';
import {
  withAddedStartInput,
  withRemovedStartInput,
  withRenamedStartInput,
  withUpdatedStartInput,
  type WorkflowCanvasNodeDraft,
} from '@/lib/workflow-editor';

interface WorkflowCanvasStartVariablesEditorProps {
  selectedNode: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

export function WorkflowCanvasStartVariablesEditor(props: WorkflowCanvasStartVariablesEditorProps) {
  const { copy } = useWebLocale();
  const { selectedNode, onUpdateNode } = props;
  const variables = selectedNode.start?.inputs ?? [];
  const [editingIndex, setEditingIndex] = useState<number>();
  const [pendingNames, setPendingNames] = useState<Record<number, string>>({});
  const editingInput = editingIndex !== undefined ? variables[editingIndex] : undefined;
  const variableNamesKey = buildVariableNamesKey(variables);

  useEffect(() => {
    setPendingNames({});
  }, [selectedNode.id, variableNamesKey]);

  const commitPendingName = (index: number) => {
    const pendingName = pendingNames[index];
    const currentName = variables[index]?.name;
    setPendingNames((state) => withoutIndex(state, index));
    if (pendingName === undefined || currentName === undefined || pendingName === currentName) {
      return;
    }
    onUpdateNode(withRenamedStartInput(selectedNode, index, pendingName));
  };

  return (
    <>
      <StartVariablesPanel
        copy={copy}
        variables={variables}
        pendingNames={pendingNames}
        onAdd={() => onUpdateNode(withAddedStartInput(selectedNode))}
        onOpenEditor={setEditingIndex}
        onRemove={(targetIndex) => onUpdateNode(withRemovedStartInput(selectedNode, targetIndex))}
        onNameChange={(targetIndex, value) => setPendingNames((state) => ({ ...state, [targetIndex]: value }))}
        onNameBlur={commitPendingName}
        onNameCancel={(targetIndex) => setPendingNames((state) => withoutIndex(state, targetIndex))}
      />
      <WorkflowStartInputModal
        open={editingInput !== undefined}
        input={editingInput ?? EMPTY_INPUT}
        index={editingIndex ?? 0}
        onCancel={() => setEditingIndex(undefined)}
        onSave={(nextInput) => {
          if (editingIndex === undefined) {
            return;
          }
          onUpdateNode(withUpdatedStartInput(selectedNode, editingIndex, nextInput));
          setEditingIndex(undefined);
        }}
      />
    </>
  );
}

interface StartVariablesPanelProps {
  copy: ReturnType<typeof useWebLocale>['copy'];
  variables: WorkflowInputVariable[];
  pendingNames: Record<number, string>;
  onAdd: () => void;
  onOpenEditor: (index: number) => void;
  onRemove: (index: number) => void;
  onNameChange: (index: number, value: string) => void;
  onNameBlur: (index: number) => void;
  onNameCancel: (index: number) => void;
}

function StartVariablesPanel(props: StartVariablesPanelProps) {
  const { copy, variables, pendingNames, onAdd, onOpenEditor, onRemove, onNameChange, onNameBlur, onNameCancel } = props;
  return (
    <div className="workflow-arch-prop-group">
      <div className="workflow-arch-prop-group-head">
        <span>{copy.workflow.startVariables}</span>
        <button type="button" className="workflow-arch-inline-button" onClick={onAdd}>
          {copy.workflow.startAddItem}
        </button>
      </div>
      <div className="workflow-arch-variable-list">
        {variables.map((input, index) => (
          <StartVariableItem
            key={`${input.name}-${index}`}
            copy={copy}
            index={index}
            input={input}
            pendingName={pendingNames[index]}
            onOpenEditor={onOpenEditor}
            onRemove={onRemove}
            onNameChange={onNameChange}
            onNameBlur={onNameBlur}
            onNameCancel={onNameCancel}
          />
        ))}
      </div>
    </div>
  );
}

interface StartVariableItemProps {
  copy: ReturnType<typeof useWebLocale>['copy'];
  input: WorkflowInputVariable;
  index: number;
  pendingName?: string;
  onOpenEditor: (index: number) => void;
  onRemove: (index: number) => void;
  onNameChange: (index: number, value: string) => void;
  onNameBlur: (index: number) => void;
  onNameCancel: (index: number) => void;
}

function StartVariableItem(props: StartVariableItemProps) {
  const {
    copy,
    input,
    index,
    pendingName,
    onOpenEditor,
    onRemove,
    onNameChange,
    onNameBlur,
    onNameCancel,
  } = props;

  return (
    <div
      className="workflow-arch-variable-item"
      onClick={() => onOpenEditor(index)}
      onKeyDown={(event) => handleVariableItemKeyDown({ event, onOpen: () => onOpenEditor(index) })}
      role="button"
      tabIndex={0}
    >
      <div className="workflow-arch-variable-item-head">
        <span>{copy.workflow.startArgLabel(index)}</span>
        <button
          type="button"
          className="workflow-arch-remove-button"
          onClick={(event) => {
            event.stopPropagation();
            onRemove(index);
          }}
        >
          ✕
        </button>
      </div>
      <input
        type="text"
        value={pendingName ?? input.name}
        maxLength={INPUT_NAME_MAX_LENGTH}
        placeholder={copy.workflow.startVariableNamePlaceholder}
        onClick={(event) => event.stopPropagation()}
        onChange={(event) => onNameChange(index, event.target.value)}
        onBlur={() => onNameBlur(index)}
        onKeyDown={(event) => handleNameInputKeyDown({
          event,
          index,
          onCommit: onNameBlur,
          onCancel: onNameCancel,
        })}
      />
      <p className="workflow-arch-variable-item-meta">
        <strong>{copy.workflow.startValueLabel}</strong> {formatStartInputValuePreview(input, copy)}
      </p>
    </div>
  );
}

const EMPTY_INPUT: WorkflowInputVariable = {
  name: '',
  type: 'string',
};

function buildVariableNamesKey(inputs: WorkflowInputVariable[]): string {
  return inputs.map((input) => input.name).join('\u0000');
}

function withoutIndex(state: Record<number, string>, index: number): Record<number, string> {
  if (!(index in state)) {
    return state;
  }
  const nextState = { ...state };
  delete nextState[index];
  return nextState;
}

function handleNameInputKeyDown(
  options: {
    event: KeyboardEvent<HTMLInputElement>;
    index: number;
    onCommit: (index: number) => void;
    onCancel: (index: number) => void;
  },
) {
  const { event, index, onCommit, onCancel } = options;
  event.stopPropagation();
  if (event.nativeEvent.isComposing) {
    return;
  }
  if (event.key === 'Enter') {
    event.preventDefault();
    onCommit(index);
    event.currentTarget.blur();
    return;
  }
  if (event.key !== 'Escape') {
    return;
  }
  event.preventDefault();
  onCancel(index);
  event.currentTarget.blur();
}

function handleVariableItemKeyDown(options: {
  event: KeyboardEvent<HTMLDivElement>;
  onOpen: () => void;
}) {
  const { event, onOpen } = options;
  if (event.key !== 'Enter' && event.key !== ' ') {
    return;
  }
  event.preventDefault();
  onOpen();
}

const VALUE_PREVIEW_MAX_LENGTH = 72;

function formatStartInputValuePreview(
  input: WorkflowInputVariable,
  copy: ReturnType<typeof useWebLocale>['copy'],
): string {
  if (!Object.prototype.hasOwnProperty.call(input, 'default')) {
    return copy.workflow.startValueUnset;
  }
  const rawValue = formatInputValue(input.default, copy);
  if (rawValue.length <= VALUE_PREVIEW_MAX_LENGTH) {
    return rawValue;
  }
  return `${rawValue.slice(0, VALUE_PREVIEW_MAX_LENGTH)}...`;
}

function formatInputValue(value: unknown, copy: ReturnType<typeof useWebLocale>['copy']): string {
  if (typeof value === 'string') {
    return value;
  }
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }
  const serialized = JSON.stringify(value);
  return serialized ?? copy.workflow.startValueUndefined;
}
