import type { WorkflowAgentRuntimeCatalog } from '@/lib/workflow-editor';
import type {
  AutosaveState,
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowCanvasPosition,
  WorkflowNodeType,
} from '@/lib/workflow-editor';
import type { WebCopy } from '@/lib/i18n/messages';

export type WorkflowEditorMode = 'create' | 'edit';
export type WorkflowEditorPhase = 'loading' | 'ready';

export interface UseWorkflowEditorControllerOptions {
  mode: WorkflowEditorMode;
  taskID?: string;
  copy: WebCopy;
}

export interface UseWorkflowEditorControllerResult {
  phase: WorkflowEditorPhase;
  draft: WorkflowCanvasDraft;
  autosaveState: AutosaveState;
  actionError: string;
  importSessionID: string;
  importLoading: boolean;
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  agentRuntimeLoading: boolean;
  agentRuntimeError: string;
  validationErrors: string[];
  onChangeImportSessionID: (value: string) => void;
  onImportFromSession: () => Promise<void>;
  onScheduleChange: (patch: Partial<WorkflowCanvasDraft['schedule']>) => void;
  onAddNode: (type: WorkflowNodeType, position: WorkflowCanvasPosition) => void;
  onSelectNode: (nodeID?: string) => void;
  onMoveNode: (nodeID: string, position: WorkflowCanvasPosition) => void;
  onConnectNodes: (sourceNodeID: string, targetNodeID: string) => void;
  onDeleteEdge: (edgeID: string) => void;
  onDuplicateNode: (nodeID: string) => void;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
  onDeleteNode: (nodeID: string) => void;
  onSave: () => Promise<void>;
  onBack: () => void;
}

