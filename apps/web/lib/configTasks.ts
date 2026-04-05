import type {
  TaskCreateRequest as SharedTaskCreateRequest,
  TaskUpdateRequest as SharedTaskUpdateRequest,
} from '@/lib/envelope.generated';
import type {
  AgentMessageTaskPayload,
  TaskPayload,
  TaskRuntimeOverrides,
  TextTaskCreateRequest,
  TextTaskUpdateRequest,
} from '@/lib/types';

type WorkflowTaskCreateRequest = Extract<SharedTaskCreateRequest, { task_kind: 'workflow' }>;
type TaskCreateRequest = WorkflowTaskCreateRequest | TextTaskCreateRequest;
type TaskUpdateRequest = Pick<
  SharedTaskUpdateRequest,
  'message' | 'session_id' | 'runtime_overrides' | 'interval_seconds' | 'cron_expr' | 'enabled' | 'task_kind' | 'workflow'
> | TextTaskUpdateRequest;

const MIN_INTERVAL_SECONDS = 1;
const DECIMAL_RADIX = 10;
const TOOL_ALLOWLIST_SEPARATOR_PATTERN = /[\n,]/;
const WORKFLOW_START_NODE_ID = 'start-node';
const WORKFLOW_END_NODE_ID = 'end-node';
const WORKFLOW_AGENT_NODE_PREFIX = 'agent-node-';

export type TaskEditorMode = 'create' | 'edit';
export type TaskScheduleMode = 'interval' | 'cron';
export type TaskEditorKind = 'agent_message' | 'workflow';

export interface TaskEditorState {
  taskKind: TaskEditorKind;
  message: string;
  scheduleMode: TaskScheduleMode;
  intervalSeconds: string;
  cronExpr: string;
  sessionId: string;
  workflowSessionId: string;
  runtimeOverridesEnabled: boolean;
  runtimeModel: string;
  runtimeToolAllowlist: string;
}

export const emptyTaskEditorState: TaskEditorState = {
  taskKind: 'workflow',
  message: '',
  scheduleMode: 'interval',
  intervalSeconds: '300',
  cronExpr: '',
  sessionId: '',
  workflowSessionId: '',
  runtimeOverridesEnabled: false,
  runtimeModel: '',
  runtimeToolAllowlist: '',
};

export function editorStateFromTask(task: AgentMessageTaskPayload): TaskEditorState {
  const runtimeOverrides = task.runtime_overrides;

  return {
    taskKind: 'agent_message',
    message: task.message,
    scheduleMode: task.schedule_type,
    intervalSeconds: task.interval_seconds === undefined ? '' : String(task.interval_seconds),
    cronExpr: task.cron_expr ?? '',
    sessionId: task.session_id ?? '',
    workflowSessionId: task.session_id ?? '',
    runtimeOverridesEnabled: runtimeOverrides !== undefined,
    runtimeModel: runtimeOverrides?.model ?? '',
    runtimeToolAllowlist: (runtimeOverrides?.tool_allowlist ?? []).join(', '),
  };
}

export function taskCreateRequestFromEditor(
  editor: TaskEditorState,
  tasks: TaskPayload[],
): TaskCreateRequest {
  if (editor.taskKind === 'workflow') {
    return workflowTaskCreateRequestFromEditor(editor, tasks);
  }

  const runtimeOverrides = runtimeOverridesFromEditor(editor);

  return {
    task_kind: 'agent_message',
    ...taskMutationFieldsFromEditor(editor),
    runtime_overrides: runtimeOverrides,
  } satisfies TextTaskCreateRequest;
}

export function taskUpdateRequestFromEditor(editor: TaskEditorState): TaskUpdateRequest {
  if (editor.taskKind !== 'agent_message') {
    throw new Error('workflow editing is not supported yet');
  }
  const runtimeOverrides = runtimeOverridesForUpdate(editor);

  return {
    task_kind: 'agent_message',
    ...taskMutationFieldsFromEditor(editor),
    runtime_overrides: runtimeOverrides,
  };
}

function workflowTaskCreateRequestFromEditor(
  editor: TaskEditorState,
  tasks: TaskPayload[],
): TaskCreateRequest {
  const sessionID = editor.workflowSessionId.trim();
  const messages = workflowMessagesFromTasks(tasks, sessionID);

  return {
    task_kind: 'workflow',
    workflow: buildWorkflowDefinition(messages),
    ...scheduleFieldsFromEditor(editor),
  };
}

function workflowMessagesFromTasks(tasks: TaskPayload[], sessionID: string): string[] {
  const messages = tasks
    .filter((task): task is AgentMessageTaskPayload => task.task_kind === 'agent_message')
    .filter((task) => (task.session_id ?? '').trim() === sessionID)
    .sort((left, right) => left.created_at.localeCompare(right.created_at))
    .map((task) => task.message.trim())
    .filter((message) => message.length > 0);

  if (messages.length === 0) {
    throw new Error('no frontend tasks found for this session id');
  }

  return messages;
}

function buildWorkflowDefinition(messages: string[]) {
  const nodes = [
    { id: WORKFLOW_START_NODE_ID, type: 'start' as const },
    ...messages.map((message, index) => ({
      id: `${WORKFLOW_AGENT_NODE_PREFIX}${index + 1}`,
      type: 'agent' as const,
      agent: { message },
    })),
    { id: WORKFLOW_END_NODE_ID, type: 'end' as const },
  ];

  return {
    nodes,
    edges: nodes.slice(0, -1).map((node, index) => ({
      from_node_id: node.id,
      to_node_id: nodes[index + 1].id,
    })),
  };
}

function taskMutationFieldsFromEditor(editor: TaskEditorState): {
  message: string;
  session_id: string;
  interval_seconds?: number;
  cron_expr?: string;
} {
  const message = editor.message.trim();
  if (message.length === 0) {
    throw new Error('message is required');
  }

  return {
    message,
    session_id: editor.sessionId.trim(),
    ...scheduleFieldsFromEditor(editor),
  };
}

function scheduleFieldsFromEditor(editor: TaskEditorState): {
  interval_seconds?: number;
  cron_expr?: string;
} {
  if (editor.scheduleMode === 'interval') {
    return {
      interval_seconds: parseIntervalSeconds(editor.intervalSeconds),
    };
  }

  const cronExpr = editor.cronExpr.trim();
  if (!cronExpr) {
    throw new Error('cron expression is required');
  }

  return {
    cron_expr: cronExpr,
  };
}

function runtimeOverridesForUpdate(
  editor: TaskEditorState,
): TaskRuntimeOverrides | Record<string, never> {
  if (!editor.runtimeOverridesEnabled) {
    return {};
  }

  return runtimeOverridesFromEditor(editor) ?? {};
}

function runtimeOverridesFromEditor(editor: TaskEditorState): TaskRuntimeOverrides | undefined {
  if (!editor.runtimeOverridesEnabled) {
    return undefined;
  }
  const model = editor.runtimeModel.trim();
  const toolAllowlist = parseToolAllowlist(editor.runtimeToolAllowlist);
  if (model === '' && toolAllowlist.length === 0) {
    return undefined;
  }

  return {
    model: model === '' ? undefined : model,
    tool_allowlist: toolAllowlist.length === 0 ? undefined : toolAllowlist,
  };
}

function parseToolAllowlist(raw: string): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const chunk of raw.split(TOOL_ALLOWLIST_SEPARATOR_PATTERN)) {
    const name = chunk.trim();
    if (name === '' || seen.has(name)) {
      continue;
    }
    seen.add(name);
    out.push(name);
  }

  return out;
}

function parseIntervalSeconds(value: string): number {
  const parsed = Number.parseInt(value.trim(), DECIMAL_RADIX);
  if (!Number.isFinite(parsed) || parsed < MIN_INTERVAL_SECONDS) {
    throw new Error('interval seconds must be a positive integer');
  }

  return parsed;
}
