import type { WorkflowInputVariable } from '@/lib/types';

export type WorkflowInputType = WorkflowInputVariable['type'];
export type StartInputDefaultErrorCode =
  | 'START_INPUT_DEFAULT_NUMBER'
  | 'START_INPUT_DEFAULT_JSON'
  | 'START_INPUT_DEFAULT_OBJECT'
  | 'START_INPUT_DEFAULT_ARRAY';
const NUMBER_INPUT_TEXT_PATTERN = /^-?(?:\d+)?(?:\.\d*)?$/;

export function parseStartInputDefault(inputType: WorkflowInputType, rawValue: string): unknown {
  if (inputType === 'string') {
    return rawValue;
  }
  if (inputType === 'number') {
    const trimmed = rawValue.trim();
    if (trimmed.length === 0) {
      throw new StartInputDefaultError('START_INPUT_DEFAULT_NUMBER');
    }
    const parsedNumber = Number(trimmed);
    if (!Number.isFinite(parsedNumber)) {
      throw new StartInputDefaultError('START_INPUT_DEFAULT_NUMBER');
    }
    return parsedNumber;
  }
  if (inputType === 'boolean') {
    return rawValue === 'true';
  }

  const parsedJSON = parseJSONInput(rawValue);
  if (inputType === 'object') {
    if (!parsedJSON || typeof parsedJSON !== 'object' || Array.isArray(parsedJSON)) {
      throw new StartInputDefaultError('START_INPUT_DEFAULT_OBJECT');
    }
    return parsedJSON;
  }
  if (!Array.isArray(parsedJSON)) {
    throw new StartInputDefaultError('START_INPUT_DEFAULT_ARRAY');
  }
  return parsedJSON;
}

export function formatStartInputDefault(value: unknown, inputType: WorkflowInputType): string {
  if (value === undefined) {
    return inputType === 'boolean' ? 'false' : '';
  }
  if (inputType === 'object' || inputType === 'array') {
    return JSON.stringify(value, null, 2);
  }
  if (inputType === 'boolean') {
    return value ? 'true' : 'false';
  }
  return String(value);
}

export function validateStartInputDefault(inputType: WorkflowInputType, defaultText: string): string {
  try {
    parseStartInputDefault(inputType, defaultText);
    return '';
  } catch (error) {
    return messageForStartInputDefaultError(error);
  }
}

export function validateStartInputDefaultCode(
  inputType: WorkflowInputType,
  defaultText: string,
): StartInputDefaultErrorCode | undefined {
  try {
    parseStartInputDefault(inputType, defaultText);
    return undefined;
  } catch (error) {
    return parseStartInputDefaultErrorCode(error);
  }
}

export function parseStartInputDefaultErrorCode(error: unknown): StartInputDefaultErrorCode | undefined {
  if (error instanceof StartInputDefaultError) {
    return error.code;
  }
  return undefined;
}

export function formatStartInputJSON(inputType: WorkflowInputType, defaultText: string): string {
  if (inputType !== 'object' && inputType !== 'array') {
    return defaultText;
  }
  try {
    return formatStartInputDefault(parseStartInputDefault(inputType, defaultText), inputType);
  } catch {
    return defaultText;
  }
}

export function coerceStartInputDefaultText(inputType: WorkflowInputType, defaultText: string): string {
  if (inputType === 'boolean') {
    return defaultText === 'true' ? 'true' : 'false';
  }
  if (inputType === 'number' || inputType === 'string') {
    return defaultText;
  }
  return formatStartInputJSON(inputType, defaultText);
}

export function isNumberInputText(value: string): boolean {
  return NUMBER_INPUT_TEXT_PATTERN.test(value);
}

class StartInputDefaultError extends Error {
  readonly code: StartInputDefaultErrorCode;

  constructor(code: StartInputDefaultErrorCode) {
    super(messageFromStartInputDefaultErrorCode(code));
    this.code = code;
  }
}

function parseJSONInput(rawValue: string): unknown {
  try {
    return JSON.parse(rawValue) as unknown;
  } catch {
    throw new StartInputDefaultError('START_INPUT_DEFAULT_JSON');
  }
}

function messageForStartInputDefaultError(error: unknown): string {
  if (error instanceof StartInputDefaultError) {
    return messageFromStartInputDefaultErrorCode(error.code);
  }
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }
  return 'invalid start input default value';
}

function messageFromStartInputDefaultErrorCode(code: StartInputDefaultErrorCode): string {
  if (code === 'START_INPUT_DEFAULT_NUMBER') {
    return 'value must be a finite number';
  }
  if (code === 'START_INPUT_DEFAULT_JSON') {
    return 'value must be valid JSON';
  }
  if (code === 'START_INPUT_DEFAULT_OBJECT') {
    return 'value for object must be a JSON object';
  }
  return 'value for array must be a JSON array';
}
