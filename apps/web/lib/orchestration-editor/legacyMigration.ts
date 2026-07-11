import { createOrchestration } from '@/lib/api/orchestrations/api';
import { deleteLegacyOrchestration, listLegacyOrchestrationRecords } from '@/lib/orchestrationStore';
import { cloneOrchestrationTaskRuntimeOverrides } from '@/lib/workflow-editor/agentRuntime';
import { DEFAULT_ORCHESTRATION_AGENT_TITLE_PREFIX, DEFAULT_ORCHESTRATION_GROUP_TITLE_PREFIX } from '@/lib/workflow-editor/constants';
import type { WorkflowCanvasDraft, WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';
import { draftToOrchestrationCreatePayload } from './draft';

const MIGRATION_GROUP_MAX_ROUNDS = 1;
const MIGRATION_GROUP_Y = 220;
const MIGRATION_GROUP_X_GAP = 320;
const MIGRATION_MEMBER_Y = 60;

interface MigratedAgentNodeOptions {
  sourceNode: WorkflowCanvasNodeDraft;
  agentID: string;
  sequence: number;
  x: number;
}

export async function migrateLegacyOrchestrations(): Promise<boolean> {
  const records = listLegacyOrchestrationRecords();
  if (records.length === 0) {
    return false;
  }
  for (const record of records) {
    const migratedDraft = migrateLegacyDraft(record.draft);
    await createOrchestration(draftToOrchestrationCreatePayload(migratedDraft, record.name));
    deleteLegacyOrchestration(record.id);
  }
  return true;
}

function migrateLegacyDraft(legacyDraft: WorkflowCanvasDraft): WorkflowCanvasDraft {
  const agents = legacyDraft.nodes.filter((node) => node.type === 'agent');
  const nodes: WorkflowCanvasDraft['nodes'] = [];
  const edges: WorkflowCanvasDraft['edges'] = [];
  let previousControlID = '';
  agents.forEach((agentNode, index) => {
    const sequence = index + 1;
    const groupID = `migrated-group-${sequence}`;
    const agentID = `migrated-agent-${sequence}`;
    const x = sequence * MIGRATION_GROUP_X_GAP;
    nodes.push(buildMigratedGroupNode(groupID, sequence, x));
    nodes.push(buildMigratedAgentNode({ sourceNode: agentNode, agentID, sequence, x }));
    if (previousControlID) {
      edges.push({ id: `edge-control-${previousControlID}-${groupID}`, from_node_id: previousControlID, to_node_id: groupID, kind: 'control' });
    }
    edges.push({ id: `edge-member-${agentID}-${groupID}`, from_node_id: agentID, to_node_id: groupID, kind: 'member' });
    previousControlID = groupID;
  });
  return { ...legacyDraft, nodes, edges, selectedNodeId: undefined };
}

function buildMigratedGroupNode(groupID: string, sequence: number, x: number): WorkflowCanvasNodeDraft {
  return {
    id: groupID,
    type: 'group',
    position: { x, y: MIGRATION_GROUP_Y },
    ui: { toolArgumentsMode: 'kv' },
    group: {
      title: `${DEFAULT_ORCHESTRATION_GROUP_TITLE_PREFIX} ${sequence}`,
      shared_context: '',
      speaking_mode: 'sequential',
      max_rounds: MIGRATION_GROUP_MAX_ROUNDS,
    },
  };
}

function buildMigratedAgentNode(options: MigratedAgentNodeOptions): WorkflowCanvasNodeDraft {
  const { sourceNode, agentID, sequence, x } = options;
  return {
    id: agentID,
    type: 'agent',
    position: { x, y: MIGRATION_MEMBER_Y },
    ui: { toolArgumentsMode: 'kv' },
    agent: {
      title: `${DEFAULT_ORCHESTRATION_AGENT_TITLE_PREFIX} ${sequence}`,
      message: sourceNode.agent?.message ?? '',
      runtime_overrides: cloneOrchestrationTaskRuntimeOverrides(sourceNode.agent?.runtime_overrides),
    },
  };
}
