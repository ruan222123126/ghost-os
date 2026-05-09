import type { WebLocale } from '@/lib/i18n/locale';
import type { WebCopy } from '@/lib/i18n/messages';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import type { PresetPayload } from '@/lib/types';
import type {
  AutosaveState,
  WorkflowAgentRuntimeCatalog,
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowCanvasPosition,
  WorkflowNodeType,
} from '@/lib/workflow-editor';

export type OrchestrationEditorPhase = 'loading' | 'ready' | 'missing';

export interface UseOrchestrationEditorControllerOptions {
  orchestrationID: string;
  copy: WebCopy;
  locale: WebLocale;
}

export interface UseOrchestrationEditorControllerResult {
  phase: OrchestrationEditorPhase;
  draft: WorkflowCanvasDraft;
  autosaveState: AutosaveState;
  actionError: string;
  validationErrors: string[];
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  agentRuntimeLoading: boolean;
  agentRuntimeError: string;
  presets: PresetPayload[];
  presetLoading: boolean;
  presetError: string;
  workflowCopy: WorkflowCopy;
  localizeValidationError: (message: string, locale: WebLocale) => string;
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
