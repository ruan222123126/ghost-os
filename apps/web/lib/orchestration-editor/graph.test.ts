import {
  canCreateOrchestrationEdge,
  connectOrchestrationNodes,
  resolveOrchestrationEdgeKind,
} from '@/lib/orchestration-editor/graph';
import type { WorkflowCanvasDraft, WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

describe('lib/orchestration-editor/graph', () => {
  it('resolves supported orchestration edge kinds', () => {
    const agent = buildNode('agent-a', 'agent');
    const groupA = buildNode('group-a', 'group');
    const groupB = buildNode('group-b', 'group');

    expect(resolveOrchestrationEdgeKind(agent, groupA)).toBe('member');
    expect(resolveOrchestrationEdgeKind(groupA, groupB)).toBe('control');
    expect(resolveOrchestrationEdgeKind(groupA, agent)).toBeUndefined();
    expect(resolveOrchestrationEdgeKind(groupA, groupA)).toBeUndefined();
    expect(canCreateOrchestrationEdge(agent, groupA)).toBe(true);
  });

  it('connects supported orchestration nodes once', () => {
    const draft = buildDraft();
    const connected = connectOrchestrationNodes(draft, 'agent-a', 'group-a');
    const duplicate = connectOrchestrationNodes(connected, 'agent-a', 'group-a');

    expect(connected.edges).toEqual([
      {
        id: 'edge-1-agent-a-group-a-member',
        from_node_id: 'agent-a',
        to_node_id: 'group-a',
        kind: 'member',
      },
    ]);
    expect(duplicate).toBe(connected);
  });

  it('keeps draft unchanged for unsupported orchestration connections', () => {
    const draft = buildDraft();

    expect(connectOrchestrationNodes(draft, 'group-a', 'agent-a')).toBe(draft);
    expect(connectOrchestrationNodes(draft, 'agent-a', 'missing')).toBe(draft);
  });
});

function buildDraft(): WorkflowCanvasDraft {
  return {
    mode: 'create',
    schedule: { mode: 'interval', intervalSeconds: '', cronExpr: '' },
    nodes: [
      buildNode('agent-a', 'agent'),
      buildNode('group-a', 'group'),
      buildNode('group-b', 'group'),
    ],
    edges: [],
  };
}

function buildNode(
  id: string,
  type: WorkflowCanvasNodeDraft['type'],
): WorkflowCanvasNodeDraft {
  return {
    id,
    type,
    position: { x: 0, y: 0 },
    ui: { toolArgumentsMode: 'kv' },
  };
}
