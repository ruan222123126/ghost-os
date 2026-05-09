import { createEmptyOrchestrationDraft } from '@/lib/orchestration-editor/draft';
import {
  addOrchestrationNode,
  buildDefaultOrchestrationNodeSource,
  connectOrchestrationDraftNodes,
  duplicateOrchestrationNode,
} from '@/lib/orchestration-editor/draftReducer';
import type { WorkflowCanvasDraft, WorkflowCanvasPosition, WorkflowNodeType } from '@/lib/workflow-editor/types';

const NODE_POSITION: WorkflowCanvasPosition = { x: 120, y: 240 };
const TOOL_NAMES = ['bash_exec', 'screen_control'];

describe('lib/orchestration-editor/draftReducer', () => {
  it('adds orchestration nodes with orchestration defaults', () => {
    const draft = createEmptyOrchestrationDraft();
    const withGroup = addNodeWithDefaultSource(draft, 'group');
    const withAgent = addNodeWithDefaultSource(withGroup, 'agent');

    expect(withAgent.nodes).toHaveLength(2);
    expect(withAgent.nodes[0]).toMatchObject({
      type: 'group',
      group: { title: '群组 1' },
    });
    expect(withAgent.nodes[1]).toMatchObject({
      type: 'agent',
      agent: {
        title: '角色 1',
        runtime_overrides: {
          tool_allowlist_only: true,
          tool_allowlist: TOOL_NAMES,
        },
      },
    });
  });

  it('connects agent membership and duplicates orchestration nodes through the facade', () => {
    const withGroup = addNodeWithDefaultSource(createEmptyOrchestrationDraft(), 'group');
    const withAgent = addNodeWithDefaultSource(withGroup, 'agent');
    const [groupNode, agentNode] = withAgent.nodes;
    const connected = connectOrchestrationDraftNodes(withAgent, agentNode.id, groupNode.id);
    const duplicated = duplicateOrchestrationNode(connected, agentNode.id);

    expect(connected.edges).toEqual([
      expect.objectContaining({
        from_node_id: agentNode.id,
        to_node_id: groupNode.id,
        kind: 'member',
      }),
    ]);
    expect(duplicated.nodes).toHaveLength(3);
    expect(duplicated.nodes[2].id).not.toBe(agentNode.id);
    expect(duplicated.nodes[2].agent).toEqual(agentNode.agent);
  });

  it('rejects workflow-only node types', () => {
    expect(() => addOrchestrationNode(createEmptyOrchestrationDraft(), {
      type: 'loop',
      position: NODE_POSITION,
    })).toThrow('unsupported orchestration node type "loop"');
  });
});

function addNodeWithDefaultSource(
  draft: WorkflowCanvasDraft,
  type: WorkflowNodeType,
): WorkflowCanvasDraft {
  return addOrchestrationNode(draft, {
    type,
    position: NODE_POSITION,
    source: buildDefaultOrchestrationNodeSource({
      draft,
      type,
      toolNames: TOOL_NAMES,
      locale: 'zh-CN',
    }),
  });
}
