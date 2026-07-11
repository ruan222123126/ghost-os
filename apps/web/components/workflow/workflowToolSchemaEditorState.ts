import { useEffect, useMemo, useState } from 'react';
import {
  rowsToToolArguments,
  type WorkflowCanvasNodeDraft,
  type WorkflowToolArgumentRow,
  withToolArguments,
} from '@/lib/workflow-editor';

export interface ToolSchemaField {
  key: string;
  required: boolean;
  valueType: WorkflowToolArgumentRow['valueType'];
  description: string;
  schemaType: string;
}

interface ExtractSchemaFieldsOptions {
  toolName?: string;
}

const SCREEN_CONTROL_TOOL_NAME = 'screen_control';
const SCREEN_CONTROL_HIDDEN_SCHEMA_KEYS = new Set(['mode', 'display_id']);

export function useToolSchemaRowsState(options: {
  selectedNode: WorkflowCanvasNodeDraft;
  schemaFields: ToolSchemaField[];
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}) {
  const { selectedNode, schemaFields, onUpdateNode } = options;
  const [rows, setRows] = useState<WorkflowToolArgumentRow[]>([]);
  const [errorText, setErrorText] = useState('');
  const schemaFingerprint = useMemo(
    () => schemaFields.map((field) => `${field.key}:${field.valueType}:${field.required}`).join('|'),
    [schemaFields],
  );

  useEffect(() => {
    setRows(buildSchemaRows(schemaFields, selectedNode.tool?.arguments));
    setErrorText('');
  }, [schemaFingerprint, selectedNode.id, selectedNode.tool?.arguments]);

  return {
    rows,
    errorText,
    onRowValueChange: (index: number, value: string) => {
      const nextRows = rows.map((row, rowIndex) => (rowIndex === index ? { ...row, value } : row));
      setRows(nextRows);
      syncSchemaRowsToNode({
        node: selectedNode,
        rows: nextRows,
        schemaFields,
        onUpdateNode,
        setErrorText,
      });
    },
  };
}

function syncSchemaRowsToNode(options: {
  node: WorkflowCanvasNodeDraft;
  rows: WorkflowToolArgumentRow[];
  schemaFields: ToolSchemaField[];
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
  setErrorText: (value: string) => void;
}) {
  const { node, rows, schemaFields, onUpdateNode, setErrorText } = options;
  try {
    const parsed = parseSchemaRows(rows, schemaFields);
    const merged = mergeSchemaArguments(node.tool?.arguments, parsed, schemaFields);
    onUpdateNode(withToolArguments(node, merged));
    setErrorText('');
  } catch (error) {
    setErrorText((error as Error).message);
  }
}

function parseSchemaRows(rows: WorkflowToolArgumentRow[], schemaFields: ToolSchemaField[]): Record<string, unknown> {
  const required = new Set(schemaFields.filter((field) => field.required).map((field) => field.key));
  const parsed: Record<string, unknown> = {};

  for (const row of rows) {
    if (row.valueType !== 'null' && row.value.trim().length === 0) {
      if (required.has(row.key)) {
        throw new Error(`tool argument "${row.key}" is required`);
      }
      continue;
    }
    parsed[row.key] = rowsToToolArguments([row])[row.key];
  }

  return parsed;
}

function mergeSchemaArguments(
  existingArguments: Record<string, unknown> | undefined,
  parsedSchemaArguments: Record<string, unknown>,
  schemaFields: ToolSchemaField[],
): Record<string, unknown> {
  const editableKeys = new Set(schemaFields.map((field) => field.key));
  const merged: Record<string, unknown> = {};
  const existing = asRecord(existingArguments);

  for (const [key, value] of Object.entries(existing)) {
    if (!editableKeys.has(key)) {
      merged[key] = value;
    }
  }
  for (const [key, value] of Object.entries(parsedSchemaArguments)) {
    merged[key] = value;
  }
  return merged;
}

function buildSchemaRows(
  fields: ToolSchemaField[],
  args: Record<string, unknown> | undefined,
): WorkflowToolArgumentRow[] {
  const source = asRecord(args);
  return fields.map((field) => ({
    key: field.key,
    valueType: field.valueType,
    value: serializeSchemaValue(source[field.key], field.valueType),
  }));
}

function serializeSchemaValue(value: unknown, valueType: WorkflowToolArgumentRow['valueType']): string {
  if (value === undefined || valueType === 'null') {
    return '';
  }
  if (typeof value === 'string' && isFindIconReference(value)) {
    return value;
  }
  if (valueType === 'object' || valueType === 'array') {
    return JSON.stringify(value, null, 2);
  }
  return String(value);
}

function isFindIconReference(value: string): boolean {
  return /^\$\{find_icon(?:\.[A-Za-z0-9_-]+)*\}$/.test(value.trim());
}

export function extractSchemaFields(
  inputSchema?: Record<string, unknown>,
  options: ExtractSchemaFieldsOptions = {},
): ToolSchemaField[] {
  const schema = asRecord(inputSchema);
  const properties = asRecord(schema.properties);
  const keys = Object.keys(properties)
    .filter((key) => !isHiddenSchemaKey(options.toolName, key))
    .sort();
  if (keys.length === 0) {
    return [];
  }
  const required = new Set(asStringArray(schema.required));

  return keys.map((key) => {
    const propertySchema = asRecord(properties[key]);
    const schemaType = resolveSchemaType(propertySchema);
    return {
      key,
      required: required.has(key),
      valueType: mapSchemaTypeToValueType(schemaType),
      description: asString(propertySchema.description),
      schemaType,
    };
  });
}

function isHiddenSchemaKey(toolName: string | undefined, key: string): boolean {
  if (toolName?.trim() !== SCREEN_CONTROL_TOOL_NAME) {
    return false;
  }
  return SCREEN_CONTROL_HIDDEN_SCHEMA_KEYS.has(key);
}

function resolveSchemaType(schema: Record<string, unknown>): string {
  const typeValue = schema.type;
  if (typeof typeValue === 'string' && typeValue.trim().length > 0) {
    return typeValue.trim();
  }
  if (Array.isArray(typeValue)) {
    const candidates = typeValue.filter((value): value is string => typeof value === 'string' && value.trim().length > 0);
    const preferred = candidates.find((value) => value !== 'null');
    if (preferred) {
      return preferred;
    }
  }
  if (hasObjectProperties(schema)) {
    return 'object';
  }
  if (schema.items !== undefined) {
    return 'array';
  }
  return 'string';
}

function hasObjectProperties(schema: Record<string, unknown>): boolean {
  const properties = asRecord(schema.properties);
  return Object.keys(properties).length > 0;
}

function mapSchemaTypeToValueType(schemaType: string): WorkflowToolArgumentRow['valueType'] {
  switch (schemaType) {
    case 'integer':
    case 'number':
      return 'number';
    case 'boolean':
      return 'boolean';
    case 'object':
      return 'object';
    case 'array':
      return 'array';
    case 'null':
      return 'null';
    default:
      return 'string';
  }
}

function asString(value: unknown): string {
  if (typeof value !== 'string') {
    return '';
  }
  return value.trim();
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value
    .filter((entry): entry is string => typeof entry === 'string')
    .map((entry) => entry.trim())
    .filter((entry) => entry.length > 0);
}

const EMPTY_RECORD: Record<string, unknown> = {};

function asRecord(value: unknown): Record<string, unknown> {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return EMPTY_RECORD;
  }
  return value as Record<string, unknown>;
}
