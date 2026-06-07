import type {
  OrchestrationDefinition,
  OrchestrationEdge,
  OrchestrationGroupNode,
  OrchestrationNode,
  TaskPatchRequest,
  TaskRuntimeOverrides,
  WorkflowDefinition,
  WorkflowInputVariable,
  WorkflowNode,
  OrchestrationTaskCreateRequest,
  WorkflowTaskCreateRequest,
  WorkflowTaskPayload,
  OrchestrationTaskPayload,
} from '@/lib/types';

export type WorkflowEditorMode = 'create' | 'edit';
export type WorkflowEditorKind = 'workflow' | 'orchestration';
export type WorkflowToolArgumentsMode = 'json' | 'kv';
export type WorkflowScheduleMode = 'interval' | 'cron';
export type WorkflowNodeType = WorkflowNode['type'] | OrchestrationNode['type'];
export type WorkflowEdgeKind = OrchestrationEdge['kind'];
export type ScreenControlAtomicAction =
  | 'screenshot'
  | 'find_text'
  | 'find_icon'
  | 'click'
  | 'click_text'
  | 'click_icon';

export interface ScreenControlFindIconParams {
  template_path: string;
  template_name?: string;
  threshold?: number;
  max_results?: number;
}

export interface ScreenControlClickParams {
  x: number;
  y: number;
}

export interface ScreenControlComposerStep {
  action: ScreenControlAtomicAction;
  params?: Record<string, unknown>;
}

export interface WorkflowCanvasPosition {
  x: number;
  y: number;
}

export interface WorkflowCanvasNodeUIState {
  toolArgumentsMode: WorkflowToolArgumentsMode;
  screenControlComposer?: {
    steps: ScreenControlComposerStep[];
  };
}

export interface WorkflowCanvasNodeDraft {
  id: string;
  type: WorkflowNodeType;
  position: WorkflowCanvasPosition;
  ui: WorkflowCanvasNodeUIState;
  start?: {
    inputs?: WorkflowInputVariable[];
  };
  tool?: {
    tool_name: string;
    arguments?: Record<string, unknown>;
  };
  llm?: {
    prompt: string;
    system_prompt?: string;
  };
  agent?: {
    title?: string;
    message: string;
    runtime_overrides?: TaskRuntimeOverrides;
  };
  group?: {
    title: string;
    shared_context: string;
    speaking_mode: OrchestrationGroupNode['speaking_mode'];
    owner_agent_id?: string;
    max_rounds: number;
  };
  if?: {
    source_node_id?: string;
    operator: 'equals' | 'not_equals' | 'contains' | 'not_contains' | 'is_empty' | 'not_empty';
    value?: string;
    true_node_id?: string;
    false_node_id?: string;
  };
  loop?: {
    role: 'start' | 'end';
    loop_id: string;
    max_iterations?: number;
    body_node_id?: string;
    exit_node_id?: string;
  };
}

export interface WorkflowCanvasEdgeDraft {
  id: string;
  from_node_id: string;
  to_node_id: string;
  kind?: WorkflowEdgeKind;
}

export interface WorkflowScheduleDraft {
  mode: WorkflowScheduleMode;
  intervalSeconds: string;
  cronExpr: string;
}

export interface WorkflowCanvasDraft {
  mode: WorkflowEditorMode;
  taskId?: string;
  selectedNodeId?: string;
  schedule: WorkflowScheduleDraft;
  nodes: WorkflowCanvasNodeDraft[];
  edges: WorkflowCanvasEdgeDraft[];
}

export interface WorkflowValidationResult {
  valid: boolean;
  errors: string[];
}

export interface SessionImportResult {
  nodes: WorkflowCanvasNodeDraft[];
  edges: WorkflowCanvasEdgeDraft[];
}

export type WorkflowUpdatePayload = Pick<
  TaskPatchRequest,
  'task_kind' | 'name' | 'workflow' | 'orchestration' | 'interval_seconds' | 'cron_expr'
>;

export type WorkflowCreatePayload = WorkflowTaskCreateRequest;
export type OrchestrationCreatePayload = OrchestrationTaskCreateRequest;

export interface WorkflowDefinitionImport {
  workflow: WorkflowDefinition;
  scheduleType: WorkflowTaskPayload['schedule_type'];
  intervalSeconds?: number;
  cronExpr?: string;
}

export interface OrchestrationDefinitionImport {
  orchestration: OrchestrationDefinition;
  scheduleType: OrchestrationTaskPayload['schedule_type'];
  intervalSeconds?: number;
  cronExpr?: string;
}
