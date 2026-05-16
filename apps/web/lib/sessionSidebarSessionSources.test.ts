import {
  collectSessionSourceAssignments,
  collectSessionSourceResolution,
  isSystemSessionPartitionID,
  mergeSessionSourcePartitionViews,
  type SessionSourceAssignment,
  type SessionSourceKind,
} from './sessionSidebarSessionSources';
import { UNCLASSIFIED_PARTITION_ID } from './sessionSidebarPartitions';
import type { ChatCopy } from '@/lib/i18n/messages/chat';
import type { SessionMetadata, TaskRunLog } from '@/lib/types';

describe('lib/sessionSidebarSessionSources', () => {
  it('extracts only run-level workflow and orchestration transcript sessions from task run logs', () => {
    const runs = [
      runLog({
        task_kind: 'workflow',
        session_id_output: 'workflow-display',
        node_results: [{
          node_id: 'agent',
          node_type: 'agent',
          status: 'success',
          output: { session_id_output: 'workflow-agent' },
        }],
      }),
      runLog({
        task_kind: 'orchestration',
        session_id_output: 'orchestration-display',
        node_results: [{
          node_id: 'group',
          node_type: 'group',
          status: 'success',
          output: {
            owner_session_id: 'owner-session',
            member_session_ids: { agent1: 'member-session' },
            dispatch_results: [{ member_results: [{ session_id: 'dispatch-member' }] }],
          },
        }],
      }),
    ];
    const assignments = collectSessionSourceAssignments(runs);
    const resolution = collectSessionSourceResolution(runs);

    expect(assignmentKinds(assignments)).toEqual({
      'workflow-display': 'workflow',
      'orchestration-display': 'orchestration',
    });
    expect(assignments['orchestration-display']).toMatchObject({
      ownerID: 'task-1',
      ownerName: 'task-1',
    });
    expect(resolution.hiddenSessionIDs.sort()).toEqual([
      'dispatch-member',
      'member-session',
      'owner-session',
      'workflow-agent',
    ]);
  });

  it('keeps resumed agent task sessions in manual groups', () => {
    const assignments = collectSessionSourceAssignments([
      runLog({
        task_kind: 'agent_message',
        session_id_input: 'manual-session',
        session_id_output: 'manual-session',
      }),
      runLog({
        task_kind: 'agent_message',
        session_id_output: 'task-session',
      }),
    ]);

    expect(assignmentKinds(assignments)).toEqual({ 'task-session': 'task' });
  });

  it('assigns relay task sessions to the loop partition', () => {
    const assignments = collectSessionSourceAssignments([
      runLog({
        task_id: 'loop-task',
        task_kind: 'agent_message',
        session_id_output: 'loop-session',
      }),
      runLog({
        task_id: 'standard-task',
        task_kind: 'agent_message',
        session_id_output: 'task-session',
      }),
    ], {
      loopTaskIDs: ['loop-task'],
    });

    expect(assignmentKinds(assignments)).toEqual({
      'loop-session': 'loop',
      'task-session': 'task',
    });
  });

  it('keeps unclassified first and removes sourced or hidden execution sessions from manual views', () => {
    const sourceAndChatSessions = sessions();
    const hiddenExecutionSession = session('orchestration-member-session', '2026-05-09T00:45:00Z');
    const customSession = session('custom-session', '2026-05-09T00:30:00Z');
    const allSessions = [...sourceAndChatSessions, hiddenExecutionSession, customSession];
    const views = mergeSessionSourcePartitionViews({
      manualViews: [
        { id: UNCLASSIFIED_PARTITION_ID, name: 'Unclassified', sessions: [...sourceAndChatSessions, hiddenExecutionSession] },
        { id: 'custom', name: 'Custom', sessions: [customSession] },
      ],
      sessions: allSessions,
      sourceAssignments: {
        'workflow-session': sourceAssignment('workflow', 'workflow-1', 'Workflow 1'),
        'orchestration-session': sourceAssignment('orchestration', 'orchestration-1', 'Orchestration 1'),
      },
      hiddenSessionIDs: [hiddenExecutionSession.id],
      searchQuery: '',
      copy: copy(),
    });

    expect(views.map((view) => [view.name, view.readOnly])).toEqual([
      ['Unclassified', undefined],
      ['Workflows', true],
      ['Orchestration', true],
      ['Custom', undefined],
    ]);
    expect(views[0].sessions.map((session) => session.id)).toEqual(['chat-session']);
    expect(views[1].sessions.map((session) => session.id)).toEqual(['workflow-session']);
    expect(views[2].sessions.map((session) => session.id)).toEqual(['orchestration-session']);
    expect(views[3].sessions.map((session) => session.id)).toEqual(['custom-session']);
    expect(views[1].childPartitions?.map((partition) => partition.name)).toEqual(['Workflow 1']);
    expect(views[2].childPartitions?.map((partition) => partition.name)).toEqual(['Orchestration 1']);
  });

  it('creates nested readonly source partitions by task and orchestration', () => {
    const views = mergeSessionSourcePartitionViews({
      manualViews: [{ id: UNCLASSIFIED_PARTITION_ID, name: 'Unclassified', sessions: [] }],
      sessions: [
        session('loop-session', '2026-05-09T05:30:00Z'),
        session('task-session-a', '2026-05-09T05:00:00Z'),
        session('task-session-b', '2026-05-09T04:00:00Z'),
        session('orchestration-session-a', '2026-05-09T03:00:00Z'),
        session('orchestration-session-b', '2026-05-09T02:00:00Z'),
      ],
      sourceAssignments: {
        'loop-session': sourceAssignment('loop', 'loop-a', 'Daily loop'),
        'task-session-a': sourceAssignment('task', 'task-a', 'Daily task'),
        'task-session-b': sourceAssignment('task', 'task-b', 'Audit task'),
        'orchestration-session-a': sourceAssignment('orchestration', 'orch-a', 'Morning orchestration'),
        'orchestration-session-b': sourceAssignment('orchestration', 'orch-b', 'Evening orchestration'),
      },
      searchQuery: '',
      copy: copy(),
    });

    const loops = findPartition(views, 'Loops');
    const orchestration = findPartition(views, 'Orchestration');
    const tasks = findPartition(views, 'Tasks');
    expect(loops?.childPartitions?.map((partition) => partition.name)).toEqual(['Daily loop']);
    expect(orchestration?.childPartitions?.map((partition) => partition.name)).toEqual([
      'Morning orchestration',
      'Evening orchestration',
    ]);
    expect(tasks?.childPartitions?.map((partition) => partition.name)).toEqual([
      'Daily task',
      'Audit task',
    ]);
    expect(isSystemSessionPartitionID(orchestration?.childPartitions?.[0]?.id ?? '')).toBe(true);
  });
});

function runLog(patch: Partial<TaskRunLog>): TaskRunLog {
  return {
    task_id: 'task-1',
    run_id: 'run-1',
    trace_id: 'trace-1',
    scheduled_at: '2026-05-09T00:00:00Z',
    status: 'success',
    ...patch,
  };
}

function sessions(): SessionMetadata[] {
  return [
    session('workflow-session', '2026-05-09T03:00:00Z'),
    session('orchestration-session', '2026-05-09T02:00:00Z'),
    session('chat-session', '2026-05-09T01:00:00Z'),
  ];
}

function session(id: string, updatedAt: string): SessionMetadata {
  return {
    id,
    title: '',
    created_at: updatedAt,
    updated_at: updatedAt,
    message_count: 1,
    token_count: 1,
  };
}

function sourceAssignment(
  kind: SessionSourceKind,
  ownerID: string,
  ownerName: string,
): SessionSourceAssignment {
  return { kind, ownerID, ownerName };
}

function assignmentKinds(assignments: Record<string, SessionSourceAssignment>): Record<string, SessionSourceKind> {
  return Object.fromEntries(Object.entries(assignments).map(([sessionID, assignment]) => [sessionID, assignment.kind]));
}

function findPartition(
  views: ReturnType<typeof mergeSessionSourcePartitionViews>,
  name: string,
) {
  return views.find((view) => view.name === name);
}

function copy(): ChatCopy {
  return {
    sidebarPartitionWorkflow: 'Workflows',
    sidebarPartitionOrchestration: 'Orchestration',
    sidebarPartitionLoop: 'Loops',
    sidebarPartitionTask: 'Tasks',
  } as ChatCopy;
}
