import type { WebCopy } from '@/lib/i18n/messages';
import type { WorkflowInputVariable } from '@/lib/types';
import { INPUT_NAME_MAX_LENGTH, INPUT_NAME_PATTERN } from '@/lib/workflow-editor/constants';
import {
  parseStartInputDefault,
  parseStartInputDefaultErrorCode,
  type StartInputDefaultErrorCode,
  type WorkflowInputType,
} from '@/lib/workflow-editor/startInputDefaults';

interface SaveWorkflowStartInputOptions {
  copy: WebCopy;
  name: string;
  inputType: WorkflowInputType;
  required: boolean;
  hasValue: boolean;
  valueText: string;
  description: string;
  onSave: (input: WorkflowInputVariable) => void;
  onSetError: (value: string) => void;
}

export function saveWorkflowStartInput(options: SaveWorkflowStartInputOptions) {
  const {
    copy,
    name,
    inputType,
    required,
    hasValue,
    valueText,
    description,
    onSave,
    onSetError,
  } = options;
  const trimmedName = name.trim();
  const nameError = validateStartInputName(trimmedName, copy);
  if (nameError) {
    onSetError(nameError);
    return;
  }

  try {
    const nextInput: WorkflowInputVariable = {
      name: trimmedName,
      type: inputType,
      required,
      description: description.trim().length > 0 ? description : undefined,
    };
    if (hasValue) {
      nextInput.default = parseStartInputDefault(inputType, valueText);
    }
    onSave(nextInput);
  } catch (error) {
    const code = parseStartInputDefaultErrorCode(error);
    onSetError(code ? localizeStartInputValueError(code, copy) : (error as Error).message);
  }
}

export function validateStartInputName(name: string, copy: WebCopy): string {
  const trimmed = name.trim();
  if (trimmed.length === 0) {
    return copy.workflow.inputNameEmpty;
  }
  if (!INPUT_NAME_PATTERN.test(trimmed)) {
    return copy.workflow.inputNamePattern;
  }
  if (trimmed.length > INPUT_NAME_MAX_LENGTH) {
    return copy.workflow.inputNameTooLong(INPUT_NAME_MAX_LENGTH);
  }
  return '';
}

export function localizeStartInputValueError(code: StartInputDefaultErrorCode, copy: WebCopy): string {
  if (code === 'START_INPUT_DEFAULT_NUMBER') {
    return copy.workflow.inputValueNumberError;
  }
  if (code === 'START_INPUT_DEFAULT_JSON') {
    return copy.workflow.inputValueJSONError;
  }
  if (code === 'START_INPUT_DEFAULT_OBJECT') {
    return copy.workflow.inputValueObjectError;
  }
  return copy.workflow.inputValueArrayError;
}
