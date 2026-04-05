'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { WorkflowPropertiesPanel } from '@/components/workflow/WorkflowPropertiesPanel';
import {
  addNode,
  applyEdgeChanges,
  applyNodeChanges,
  connectNodes,
  createDraftNode,
  removeNode,
  toFlowEdge,
  toFlowNode,
  updateNode,
} from '@/components/workflow/workflowCanvasState';
import { createTask, getTask, listTasks, updateTask } from '@/lib/api/tasks/api';
import { toErrorMessage } from '@/lib/errors';
import { buildHomeSettingsURL } from '@/lib/settingsQuery';
import {
  createEmptyWorkflowDraft,
  draftToWorkflowCreatePayload,
  draftToWorkflowUpdatePayload,
  importWorkflowFromSessionTasks,
  validateWorkflowDraft,
  withWorkflowContent,
} from '@/lib/workflow-editor';
import type { WorkflowCanvasDraft, WorkflowNodeType } from '@/lib/workflow-editor';
import { useRouter } from 'next/navigation';

interface WorkflowEditorClientProps {
  mode: 'create' | 'edit';
  taskID?: string;
}

export function WorkflowEditorClient(props: WorkflowEditorClientProps) {
  const { mode, taskID } = props;
  const router = useRouter();
  const [draft, setDraft] = useState<WorkflowCanvasDraft>(() => createEmptyWorkflowDraft(mode));
  const [loading, setLoading] = useState(mode === 'edit');
  const [saving, setSaving] = useState(false);
  const [actionError, setActionError] = useState('');
  const [importSessionID, setImportSessionID] = useState('');
  const [importLoading, setImportLoading] = useState(false);
  const validation = useMemo(() => validateWorkflowDraft(draft), [draft]);
  const selectedNode = useMemo(
    () => draft.nodes.find((node) => node.id === draft.selectedNodeId),
    [draft.nodes, draft.selectedNodeId],
  );
  const flowNodes = useMemo(() => draft.nodes.map((node) => toFlowNode(node, node.id === draft.selectedNodeId)), [draft.nodes, draft.selectedNodeId]);
  const flowEdges = useMemo(() => draft.edges.map(toFlowEdge), [draft.edges]);

  useEffect(() => {
    if (mode !== 'edit' || !taskID) {
      return;
    }

    let cancelled = false;
    setLoading(true);
    setActionError('');
    void getTask(taskID)
      .then((task) => {
        if (cancelled) {
          return;
        }
        if (task.task_kind !== 'workflow') {
          throw new Error('task is not a workflow');
        }

        const base = createEmptyWorkflowDraft('edit');
        const nodes = task.workflow.nodes.map((node, index) => createDraftNode(node.id, node.type, index, node));
        const edges = task.workflow.edges.map((edge, index) => ({
          id: `edge-${index}-${edge.from_node_id}-${edge.to_node_id}`,
          from_node_id: edge.from_node_id,
          to_node_id: edge.to_node_id,
        }));

        setDraft({
          ...base,
          mode: 'edit',
          taskId: task.id,
          schedule: task.schedule_type === 'interval'
            ? { mode: 'interval', intervalSeconds: String(task.interval_seconds ?? 300), cronExpr: '' }
            : { mode: 'cron', intervalSeconds: '300', cronExpr: task.cron_expr ?? '' },
          nodes,
          edges,
        });
      })
      .catch((error) => {
        if (cancelled) {
          return;
        }
        setActionError(toErrorMessage(error, 'failed to load workflow task'));
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [mode, taskID]);

  const handleCancel = useCallback(() => {
    router.push(buildHomeSettingsURL('tasks'));
  }, [router]);

  const handleSave = useCallback(async () => {
    setActionError('');
    if (!validation.valid) {
      setActionError('workflow validation failed');
      return;
    }

    setSaving(true);
    try {
      if (mode === 'edit' && taskID) {
        await updateTask(taskID, draftToWorkflowUpdatePayload(draft));
      } else {
        await createTask(draftToWorkflowCreatePayload(draft));
      }
      router.push(buildHomeSettingsURL('tasks'));
    } catch (error) {
      setActionError(toErrorMessage(error, 'failed to save workflow'));
    } finally {
      setSaving(false);
    }
  }, [draft, mode, router, taskID, validation.valid]);

  const handleImport = useCallback(async () => {
    setImportLoading(true);
    setActionError('');
    try {
      const tasks = await listTasks();
      const imported = importWorkflowFromSessionTasks(tasks, importSessionID);
      setDraft((state) => withWorkflowContent(state, imported.nodes, imported.edges));
    } catch (error) {
      setActionError(toErrorMessage(error, 'failed to import session tasks'));
    } finally {
      setImportLoading(false);
    }
  }, [importSessionID]);

  if (loading) {
    return <main className="workflow-loading">Loading workflow task...</main>;
  }

  return (
    <main className="workflow-page">
      <header className="workflow-header">
        <button type="button" onClick={handleCancel}>Back</button>
        <div className="workflow-header-title">
          <h1>{mode === 'edit' ? `Edit Workflow ${taskID}` : 'New Workflow'}</h1>
          <p>{validation.valid ? 'Validation: passed' : `Validation: failed (${validation.errors.length})`}</p>
        </div>
        <div className="workflow-header-actions">
          {(['start', 'tool', 'llm', 'agent', 'end'] as WorkflowNodeType[]).map((type) => (
            <button key={type} type="button" onClick={() => setDraft((state) => addNode(state, type))}>
              + {type}
            </button>
          ))}
          <button type="button" onClick={handleSave} disabled={saving}>
            {saving ? 'Saving...' : 'Save'}
          </button>
        </div>
      </header>

      <section className="workflow-body">
        <div className="workflow-canvas">
          <ReactFlow
            nodes={flowNodes}
            edges={flowEdges}
            fitView
            onNodesChange={(changes) => setDraft((state) => ({ ...state, nodes: applyNodeChanges(state.nodes, changes) }))}
            onEdgesChange={(changes) => setDraft((state) => ({ ...state, edges: applyEdgeChanges(state.edges, changes) }))}
            onConnect={(connection) => setDraft((state) => connectNodes(state, connection))}
            onNodeClick={(_, node) => setDraft((state) => ({ ...state, selectedNodeId: node.id }))}
            onPaneClick={() => setDraft((state) => ({ ...state, selectedNodeId: undefined }))}
          >
            <Background />
            <MiniMap />
            <Controls />
          </ReactFlow>
        </div>

        <WorkflowPropertiesPanel
          draft={draft}
          selectedNode={selectedNode}
          saving={saving}
          importSessionID={importSessionID}
          importLoading={importLoading}
          actionError={actionError}
          validationErrors={validation.errors}
          onChangeImportSessionID={setImportSessionID}
          onImportFromSession={handleImport}
          onScheduleChange={(patch) => setDraft((state) => ({ ...state, schedule: { ...state.schedule, ...patch } }))}
          onUpdateNode={(node) => setDraft((state) => updateNode(state, node))}
          onDeleteNode={(nodeID) => setDraft((state) => removeNode(state, nodeID))}
        />
      </section>
    </main>
  );
}
