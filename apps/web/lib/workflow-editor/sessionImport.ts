import type { TaskPayload } from '@/lib/types';
import {
  AGENT_NODE_ID_PREFIX,
  DEFAULT_END_NODE_ID,
  DEFAULT_START_NODE_ID,
  NODE_X_GAP,
  NODE_Y_BASE,
} from '@/lib/workflow-editor/constants';
import type { SessionImportResult, WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

type AgentMessageTaskPayload = Extract<TaskPayload, { task_kind: 'agent_message' }>;

interface SessionMessageSource {
  createdAt: string;
  message: string;
}

export function importWorkflowFromSessionTasks(tasks: TaskPayload[], sessionID: string): SessionImportResult {
  const messages = buildSortedSessionMessages(tasks, sessionID);
  if (messages.length === 0) {
    throw new Error('no frontend text tasks found for this session id');
  }

  const nodes = buildNodes(messages);
  const edges = nodes.slice(0, -1).map((node, index) => ({
    id: `edge-${index}-${node.id}-${nodes[index + 1].id}`,
    from_node_id: node.id,
    to_node_id: nodes[index + 1].id,
  }));

  return { nodes, edges };
}

function buildSortedSessionMessages(tasks: TaskPayload[], sessionID: string): SessionMessageSource[] {
  const normalizedSessionID = sessionID.trim();

  return tasks
    .map((task) => sessionMessageSource(task, normalizedSessionID))
    .filter((message): message is SessionMessageSource => message !== undefined)
    .sort(compareSessionMessages);
}

function sessionMessageSource(
  task: TaskPayload,
  sessionID: string,
): SessionMessageSource | undefined {
  if (!isMatchingSessionMessageTask(task, sessionID)) {
    return undefined;
  }
  const message = task.message.trim();
  if (message.length === 0) {
    return undefined;
  }
  return {
    createdAt: task.created_at,
    message,
  };
}

function isMatchingSessionMessageTask(
  task: TaskPayload,
  sessionID: string,
): task is AgentMessageTaskPayload {
  return task.task_kind === 'agent_message' && (task.session_id ?? '').trim() === sessionID;
}

function compareSessionMessages(left: SessionMessageSource, right: SessionMessageSource): number {
  return left.createdAt.localeCompare(right.createdAt);
}

function buildNodes(messages: SessionMessageSource[]): WorkflowCanvasNodeDraft[] {
  const nodes: WorkflowCanvasNodeDraft[] = [
    {
      id: DEFAULT_START_NODE_ID,
      type: 'start',
      position: { x: 0, y: NODE_Y_BASE },
      ui: { toolArgumentsMode: 'kv' },
      start: { inputs: [] },
    },
  ];

  for (let index = 0; index < messages.length; index += 1) {
    nodes.push({
      id: `${AGENT_NODE_ID_PREFIX}-${index + 1}`,
      type: 'agent',
      position: { x: NODE_X_GAP * (index + 1), y: NODE_Y_BASE },
      ui: { toolArgumentsMode: 'kv' },
      agent: { message: messages[index].message },
    });
  }

  nodes.push({
    id: DEFAULT_END_NODE_ID,
    type: 'end',
    position: { x: NODE_X_GAP * (messages.length + 1), y: NODE_Y_BASE },
    ui: { toolArgumentsMode: 'kv' },
  });

  return nodes;
}
