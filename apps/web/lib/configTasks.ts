import type {
  TaskUpdateRequest as SharedTaskUpdateRequest,
} from '@/lib/envelope.generated';
import type {
  AgentMessageTaskPayload,
  TaskRuntimeOverrides,
  TextTaskCreateRequest,
  TextTaskUpdateRequest,
} from '@/lib/types';

const MIN_INTERVAL_SECONDS = 1;
const DECIMAL_RADIX = 10;
const TOOL_ALLOWLIST_SEPARATOR_PATTERN = /[\n,]/;

type TaskUpdateRequest = Pick<
  SharedTaskUpdateRequest,
  'message' | 'session_id' | 'runtime_overrides' | 'interval_seconds' | 'cron_expr' | 'task_kind'
> | TextTaskUpdateRequest;

export type TaskEditorMode = 'create' | 'edit';
export type TaskScheduleMode = 'interval' | 'cron';

export interface TaskEditorState {
  message: string;
  scheduleMode: TaskScheduleMode;
  intervalSeconds: string;
  cronExpr: string;
  sessionId: string;
  runtimeOverridesEnabled: boolean;
  runtimeModel: string;
  runtimeToolAllowlist: string;
}

export const emptyTaskEditorState: TaskEditorState = {
  message: '',
  scheduleMode: 'interval',
  intervalSeconds: '300',
  cronExpr: '',
  sessionId: '',
  runtimeOverridesEnabled: false,
  runtimeModel: '',
  runtimeToolAllowlist: '',
};

export function editorStateFromTask(task: AgentMessageTaskPayload): TaskEditorState {
  const runtimeOverrides = task.runtime_overrides;

  return {
    message: task.message,
    scheduleMode: task.schedule_type,
    intervalSeconds: task.interval_seconds === undefined ? '' : String(task.interval_seconds),
    cronExpr: task.cron_expr ?? '',
    sessionId: task.session_id ?? '',
    runtimeOverridesEnabled: runtimeOverrides !== undefined,
    runtimeModel: runtimeOverrides?.model ?? '',
    runtimeToolAllowlist: (runtimeOverrides?.tool_allowlist ?? []).join(', '),
  };
}

export function taskCreateRequestFromEditor(editor: TaskEditorState): TextTaskCreateRequest {
  const runtimeOverrides = runtimeOverridesFromEditor(editor);

  return {
    task_kind: 'agent_message',
    ...taskMutationFieldsFromEditor(editor),
    runtime_overrides: runtimeOverrides,
  };
}

export function taskUpdateRequestFromEditor(editor: TaskEditorState): TaskUpdateRequest {
  const runtimeOverrides = runtimeOverridesForUpdate(editor);

  return {
    task_kind: 'agent_message',
    ...taskMutationFieldsFromEditor(editor),
    runtime_overrides: runtimeOverrides,
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
