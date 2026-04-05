'use client';

import { useEffect, useMemo, useState } from 'react';
import {
  jsonTextToToolArguments,
  rowsToToolArguments,
  toolArgumentsToJSONText,
  toolArgumentsToRows,
  type WorkflowToolArgumentRow,
} from '@/lib/workflow-editor/toolArguments';
import type { WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

interface WorkflowToolNodeEditorProps {
  node: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
  onErrorChange: (error: string) => void;
}

export function WorkflowToolNodeEditor(props: WorkflowToolNodeEditorProps) {
  const { node, onUpdateNode, onErrorChange } = props;
  const [jsonText, setJSONText] = useState('{}');
  const [rows, setRows] = useState<WorkflowToolArgumentRow[]>([]);

  useEffect(() => {
    const args = node.tool?.arguments;
    setJSONText(toolArgumentsToJSONText(args));
    setRows(toolArgumentsToRows(args));
    onErrorChange('');
  }, [node.id, node.tool?.arguments, onErrorChange]);

  const mode = node.ui.toolArgumentsMode;
  const effectiveRows = useMemo(
    () => (rows.length === 0 ? [{ key: '', valueType: 'string' as const, value: '' }] : rows),
    [rows],
  );

  return (
    <div className="workflow-fieldset">
      <label>
        <span>Tool Name</span>
        <input value={node.tool?.tool_name ?? ''} onChange={(event) => onUpdateNode(withTool(node, { tool_name: event.target.value }))} />
      </label>
      <label>
        <span>Arguments Mode</span>
        <select value={mode} onChange={(event) => onUpdateNode({ ...node, ui: { ...node.ui, toolArgumentsMode: event.target.value as 'json' | 'kv' } })}>
          <option value="json">JSON</option>
          <option value="kv">Key-Value</option>
        </select>
      </label>
      {mode === 'json'
        ? renderJSONEditor(node, jsonText, setJSONText, onUpdateNode, onErrorChange)
        : renderRowsEditor(node, effectiveRows, setRows, onUpdateNode, onErrorChange)}
    </div>
  );
}

function renderJSONEditor(
  node: WorkflowCanvasNodeDraft,
  jsonText: string,
  setJSONText: (value: string) => void,
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void,
  onErrorChange: (error: string) => void,
) {
  return (
    <label>
      <span>arguments (JSON object)</span>
      <textarea
        rows={8}
        value={jsonText}
        onChange={(event) => {
          const value = event.target.value;
          setJSONText(value);
          try {
            const argumentsObject = jsonTextToToolArguments(value);
            onUpdateNode(withTool(node, { arguments: argumentsObject }));
            onErrorChange('');
          } catch (error) {
            onErrorChange((error as Error).message);
          }
        }}
      />
    </label>
  );
}

function renderRowsEditor(
  node: WorkflowCanvasNodeDraft,
  rows: WorkflowToolArgumentRow[],
  setRows: (rows: WorkflowToolArgumentRow[]) => void,
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void,
  onErrorChange: (error: string) => void,
) {
  return (
    <div className="workflow-kv-table">
      {rows.map((row, index) => (
        <div className="workflow-kv-row" key={`kv-${index}`}>
          <input
            placeholder="key"
            value={row.key}
            onChange={(event) => applyRows(node, rows, index, { key: event.target.value }, setRows, onUpdateNode, onErrorChange)}
          />
          <select
            value={row.valueType}
            onChange={(event) => applyRows(node, rows, index, { valueType: event.target.value as WorkflowToolArgumentRow['valueType'] }, setRows, onUpdateNode, onErrorChange)}
          >
            <option value="string">string</option>
            <option value="number">number</option>
            <option value="boolean">boolean</option>
            <option value="object">object</option>
            <option value="array">array</option>
            <option value="null">null</option>
          </select>
          <input
            placeholder="value"
            value={row.value}
            disabled={row.valueType === 'null'}
            onChange={(event) => applyRows(node, rows, index, { value: event.target.value }, setRows, onUpdateNode, onErrorChange)}
          />
          <button type="button" onClick={() => removeRow(node, rows, index, setRows, onUpdateNode, onErrorChange)}>
            x
          </button>
        </div>
      ))}
      <button type="button" onClick={() => setRows([...rows, { key: '', valueType: 'string', value: '' }])}>
        Add argument
      </button>
    </div>
  );
}

function applyRows(
  node: WorkflowCanvasNodeDraft,
  rows: WorkflowToolArgumentRow[],
  index: number,
  patch: Partial<WorkflowToolArgumentRow>,
  setRows: (rows: WorkflowToolArgumentRow[]) => void,
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void,
  onErrorChange: (error: string) => void,
) {
  const nextRows = rows.map((row, rowIndex) => (rowIndex === index ? { ...row, ...patch } : row));
  setRows(nextRows);
  syncRowsToNode(node, nextRows, onUpdateNode, onErrorChange);
}

function removeRow(
  node: WorkflowCanvasNodeDraft,
  rows: WorkflowToolArgumentRow[],
  index: number,
  setRows: (rows: WorkflowToolArgumentRow[]) => void,
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void,
  onErrorChange: (error: string) => void,
) {
  const nextRows = rows.filter((_, rowIndex) => rowIndex !== index);
  setRows(nextRows);
  syncRowsToNode(node, nextRows, onUpdateNode, onErrorChange);
}

function syncRowsToNode(
  node: WorkflowCanvasNodeDraft,
  rows: WorkflowToolArgumentRow[],
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void,
  onErrorChange: (error: string) => void,
) {
  const nonEmptyRows = rows.filter((row) => row.key.trim().length > 0);

  try {
    const parsed = rowsToToolArguments(nonEmptyRows);
    onUpdateNode(withTool(node, { arguments: parsed }));
    onErrorChange('');
  } catch (error) {
    onErrorChange((error as Error).message);
  }
}

function withTool(
  node: WorkflowCanvasNodeDraft,
  patch: Partial<NonNullable<WorkflowCanvasNodeDraft['tool']>>,
): WorkflowCanvasNodeDraft {
  return {
    ...node,
    tool: {
      tool_name: node.tool?.tool_name ?? '',
      arguments: node.tool?.arguments,
      ...patch,
    },
  };
}
