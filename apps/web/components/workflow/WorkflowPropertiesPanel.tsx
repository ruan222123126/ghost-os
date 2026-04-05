'use client';

import { useState } from 'react';
import {
  WorkflowAgentNodeEditor,
  WorkflowLLMNodeEditor,
  WorkflowStartInputsEditor,
} from '@/components/workflow/WorkflowStartInputsEditor';
import { WorkflowToolNodeEditor } from '@/components/workflow/WorkflowToolNodeEditor';
import type {
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowNodeType,
} from '@/lib/workflow-editor/types';

interface WorkflowPropertiesPanelProps {
  draft: WorkflowCanvasDraft;
  selectedNode?: WorkflowCanvasNodeDraft;
  saving: boolean;
  importSessionID: string;
  importLoading: boolean;
  actionError: string;
  validationErrors: string[];
  onChangeImportSessionID: (value: string) => void;
  onImportFromSession: () => void;
  onScheduleChange: (patch: Partial<WorkflowCanvasDraft['schedule']>) => void;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
  onDeleteNode: (nodeID: string) => void;
}

const NODE_TYPE_OPTIONS: WorkflowNodeType[] = ['start', 'tool', 'llm', 'agent', 'end'];

export function WorkflowPropertiesPanel(props: WorkflowPropertiesPanelProps) {
  const {
    draft,
    selectedNode,
    saving,
    importSessionID,
    importLoading,
    actionError,
    validationErrors,
    onChangeImportSessionID,
    onImportFromSession,
    onScheduleChange,
    onUpdateNode,
    onDeleteNode,
  } = props;
  const [toolError, setToolError] = useState('');

  return (
    <aside className="workflow-panel">
      <section className="workflow-panel-section">
        <h2>Schedule</h2>
        <label>
          <span>Mode</span>
          <select value={draft.schedule.mode} onChange={(event) => onScheduleChange({ mode: event.target.value as WorkflowCanvasDraft['schedule']['mode'] })}>
            <option value="interval">Interval</option>
            <option value="cron">Cron</option>
          </select>
        </label>
        {draft.schedule.mode === 'interval' ? (
          <label>
            <span>Interval Seconds</span>
            <input value={draft.schedule.intervalSeconds} onChange={(event) => onScheduleChange({ intervalSeconds: event.target.value })} />
          </label>
        ) : (
          <label>
            <span>Cron Expression</span>
            <input value={draft.schedule.cronExpr} onChange={(event) => onScheduleChange({ cronExpr: event.target.value })} />
          </label>
        )}
      </section>

      <section className="workflow-panel-section">
        <h2>Import Session</h2>
        <label>
          <span>Session ID</span>
          <input value={importSessionID} onChange={(event) => onChangeImportSessionID(event.target.value)} />
        </label>
        <button type="button" disabled={importLoading} onClick={onImportFromSession}>
          {importLoading ? 'Importing...' : 'Import Text Tasks'}
        </button>
      </section>

      {actionError ? <p className="workflow-panel-error">{actionError}</p> : null}
      {toolError ? <p className="workflow-panel-error">{toolError}</p> : null}
      {validationErrors.length > 0 ? (
        <section className="workflow-panel-section workflow-errors">
          <h2>Validation</h2>
          <ul>
            {validationErrors.map((error) => (
              <li key={error}>{error}</li>
            ))}
          </ul>
        </section>
      ) : null}

      <section className="workflow-panel-section">
        <h2>Node Properties</h2>
        {!selectedNode ? <p className="workflow-muted">Select a node to edit.</p> : null}
        {selectedNode ? (
          <div className="workflow-node-form">
            <label>
              <span>Node ID</span>
              <input value={selectedNode.id} onChange={(event) => onUpdateNode({ ...selectedNode, id: event.target.value })} />
            </label>
            <label>
              <span>Type</span>
              <select value={selectedNode.type} onChange={(event) => onUpdateNode(nodeWithType(selectedNode, event.target.value as WorkflowNodeType))}>
                {NODE_TYPE_OPTIONS.map((type) => (
                  <option key={type} value={type}>
                    {type}
                  </option>
                ))}
              </select>
            </label>

            {selectedNode.type === 'start' ? <WorkflowStartInputsEditor node={selectedNode} onUpdateNode={onUpdateNode} /> : null}
            {selectedNode.type === 'tool' ? <WorkflowToolNodeEditor node={selectedNode} onUpdateNode={onUpdateNode} onErrorChange={setToolError} /> : null}
            {selectedNode.type === 'llm' ? <WorkflowLLMNodeEditor node={selectedNode} onUpdateNode={onUpdateNode} /> : null}
            {selectedNode.type === 'agent' ? <WorkflowAgentNodeEditor node={selectedNode} onUpdateNode={onUpdateNode} /> : null}

            <button type="button" className="workflow-danger" onClick={() => onDeleteNode(selectedNode.id)} disabled={saving}>
              Delete Node
            </button>
          </div>
        ) : null}
      </section>
    </aside>
  );
}

function nodeWithType(node: WorkflowCanvasNodeDraft, type: WorkflowNodeType): WorkflowCanvasNodeDraft {
  return {
    ...node,
    type,
    start: type === 'start' ? node.start ?? { inputs: [] } : undefined,
    tool: type === 'tool' ? node.tool ?? { tool_name: '', arguments: {} } : undefined,
    llm: type === 'llm' ? node.llm ?? { prompt: '', system_prompt: '' } : undefined,
    agent: type === 'agent' ? node.agent ?? { message: '' } : undefined,
  };
}
