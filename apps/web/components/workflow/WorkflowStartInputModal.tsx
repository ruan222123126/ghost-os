'use client';

import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { useWebLocale } from '@/lib/i18n/provider';
import type { WorkflowInputVariable } from '@/lib/types';
import { INPUT_NAME_MAX_LENGTH } from '@/lib/workflow-editor/constants';
import {
  coerceStartInputDefaultText,
  formatStartInputDefault,
  formatStartInputJSON,
  isNumberInputText,
  type WorkflowInputType,
  validateStartInputDefaultCode,
} from '@/lib/workflow-editor/startInputDefaults';
import {
  localizeStartInputValueError,
  saveWorkflowStartInput,
  validateStartInputName,
} from './workflowStartInputModalHelpers';

interface WorkflowStartInputModalProps {
  open: boolean;
  input: WorkflowInputVariable;
  index: number;
  onCancel: () => void;
  onSave: (input: WorkflowInputVariable) => void;
}

export function WorkflowStartInputModal(props: WorkflowStartInputModalProps) {
  const { copy } = useWebLocale();
  const { open, input, index, onCancel, onSave } = props;
  const [mounted, setMounted] = useState(false);
  const [name, setName] = useState('');
  const [inputType, setInputType] = useState<WorkflowInputType>('string');
  const [required, setRequired] = useState(false);
  const [hasValue, setHasValue] = useState(false);
  const [valueText, setValueText] = useState('');
  const [description, setDescription] = useState('');
  const [error, setError] = useState('');
  const nameError = validateStartInputName(name, copy);
  const valueErrorCode = hasValue ? validateStartInputDefaultCode(inputType, valueText) : undefined;
  const valueError = valueErrorCode ? localizeStartInputValueError(valueErrorCode, copy) : '';
  const visibleError = error || nameError || valueError;

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (!open) {
      return;
    }
    setName(input.name);
    setInputType(input.type);
    setRequired(Boolean(input.required));
    const shouldUseValue = Object.prototype.hasOwnProperty.call(input, 'default');
    setHasValue(shouldUseValue);
    setValueText(formatStartInputDefault(input.default, input.type));
    setDescription(input.description ?? '');
    setError('');
  }, [input, open]);

  if (!open || !mounted) {
    return null;
  }

  return createPortal(
    <div className="workflow-arch-input-modal" role="dialog" aria-modal="true" aria-labelledby="workflow-input-editor-title">
      <button type="button" className="workflow-arch-input-modal-backdrop" onClick={onCancel} aria-label={copy.workflow.closeVariableEditorAria} />
      <section className="workflow-arch-input-modal-panel">
        <header className="workflow-arch-input-modal-head">
          <div>
            <h3 id="workflow-input-editor-title">{copy.workflow.startInputEditorTitle}</h3>
            <p>{copy.workflow.startArgLabel(index)}</p>
          </div>
          <button type="button" className="workflow-arch-input-modal-close" onClick={onCancel} aria-label={copy.workflow.closeVariableEditorAria}>
            ✕
          </button>
        </header>
        <div className="workflow-arch-input-modal-body">
          <label>
            <span>{copy.workflow.inputName}</span>
            <input
              value={name}
              maxLength={INPUT_NAME_MAX_LENGTH}
              onChange={(event) => {
                setName(event.target.value);
                setError('');
              }}
              placeholder={copy.workflow.startVariableNamePlaceholder}
            />
          </label>
          <label>
            <span>{copy.workflow.inputType}</span>
            <select
              value={inputType}
              onChange={(event) => {
                const nextType = event.target.value as WorkflowInputType;
                setInputType(nextType);
                setValueText((current) => coerceStartInputDefaultText(nextType, current));
                setError('');
              }}
            >
              <option value="string">string</option>
              <option value="number">number</option>
              <option value="boolean">boolean</option>
              <option value="object">object</option>
              <option value="array">array</option>
            </select>
          </label>
          <div className="workflow-arch-input-modal-switches">
            <label className="workflow-arch-input-modal-check">
              <input type="checkbox" checked={required} onChange={(event) => setRequired(event.target.checked)} />
              <span>{copy.workflow.inputRequired}</span>
            </label>
            <label className="workflow-arch-input-modal-check">
              <input
                type="checkbox"
                checked={hasValue}
                onChange={(event) => {
                  setHasValue(event.target.checked);
                  setError('');
                }}
              />
              <span>{copy.workflow.inputSetValue}</span>
            </label>
          </div>
          {hasValue ? (
            <ValueInputEditor
              copy={copy}
              inputType={inputType}
              valueText={valueText}
              onChangeValueText={(value) => {
                setValueText(value);
                setError('');
              }}
            />
          ) : null}
          <label>
            <span>{copy.workflow.inputDescription}</span>
            <textarea
              rows={3}
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              placeholder={copy.workflow.inputDescriptionPlaceholder}
            />
          </label>
          {visibleError ? <p className="workflow-arch-input-modal-error">{visibleError}</p> : null}
          <div className="workflow-arch-input-modal-actions">
            <button type="button" className="workflow-arch-input-modal-cancel" onClick={onCancel}>{copy.workflow.inputCancel}</button>
            <button
              type="button"
              className="workflow-arch-input-modal-save"
              disabled={Boolean(nameError || valueError)}
              onClick={() =>
                saveWorkflowStartInput({
                  copy,
                  name,
                  inputType,
                  required,
                  hasValue,
                  valueText,
                  description,
                  onSave,
                  onSetError: setError,
                })}
            >
              {copy.workflow.inputSaveVariable}
            </button>
          </div>
        </div>
      </section>
    </div>,
    document.body,
  );
}

interface ValueInputEditorProps {
  copy: ReturnType<typeof useWebLocale>['copy'];
  inputType: WorkflowInputType;
  valueText: string;
  onChangeValueText: (value: string) => void;
}

function ValueInputEditor(props: ValueInputEditorProps) {
  const { copy, inputType, valueText, onChangeValueText } = props;
  if (inputType === 'boolean') {
    return (
      <label>
        <span>{copy.workflow.inputValue}</span>
        <select value={valueText || 'false'} onChange={(event) => onChangeValueText(event.target.value)}>
          <option value="false">false</option>
          <option value="true">true</option>
        </select>
      </label>
    );
  }
  if (inputType === 'object' || inputType === 'array') {
    return (
      <label>
        <span>{copy.workflow.inputValueJSON}</span>
        <textarea
          rows={4}
          value={valueText}
          onChange={(event) => onChangeValueText(event.target.value)}
          onBlur={() => onChangeValueText(formatStartInputJSON(inputType, valueText))}
          placeholder={inputType === 'object' ? '{"key":"value"}' : '["item"]'}
        />
      </label>
    );
  }
  if (inputType === 'number') {
    return (
      <label>
        <span>{copy.workflow.inputValue}</span>
        <input
          type="text"
          inputMode="decimal"
          value={valueText}
          onChange={(event) => {
            const nextValue = event.target.value;
            if (!isNumberInputText(nextValue)) {
              return;
            }
            onChangeValueText(nextValue);
          }}
          placeholder="0"
        />
      </label>
    );
  }
  return (
    <label>
      <span>{copy.workflow.inputValue}</span>
      <input value={valueText} onChange={(event) => onChangeValueText(event.target.value)} />
    </label>
  );
}
