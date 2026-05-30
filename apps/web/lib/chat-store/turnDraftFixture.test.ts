import fs from 'node:fs';
import path from 'node:path';
import { chatStateReducer, createInitialChatState, type ChatStateStore } from './reducer';
import { buildChatStateView } from './runtimeReducer';
import { projectAgentEvent } from '@/lib/chatRuntime/eventProjector';
import { createChatRuntimeState, resolveEventSessionId } from '@/lib/chatRuntime/runtimeState';
import type { AgentStreamEvent, SessionTurnDraft } from '@/lib/types';

interface TurnDraftFixture {
  events: AgentStreamEvent[];
  expected_turn_draft: SessionTurnDraft;
}

describe('chat-store/turnDraft fixtures', () => {
  for (const fixture of loadTurnDraftFixtures()) {
    it(fixture.name, () => {
      const runtime = createChatRuntimeState(fixture.data.expected_turn_draft.trace_id);
      let state = createInitialChatState();
      let status: SessionTurnDraft['status'] = 'streaming';
      let error: string | undefined;
      let turn = 0;

      for (const event of fixture.data.events) {
        const sessionId = resolveEventSessionId(event);
        if (sessionId) {
          runtime.sessionId = sessionId;
        }
        state = chatStateReducer(state, {
          type: 'apply_runtime_actions',
          actions: projectAgentEvent({ event, runtime }),
        });
        turn = event.turn;

        switch (event.type) {
          case 'run_started':
            status = 'streaming';
            error = undefined;
            break;
          case 'awaiting_human':
            status = 'awaiting_human';
            error = undefined;
            break;
          case 'error':
            status = 'error';
            error = readErrorMessage(event.payload);
            break;
          default:
            break;
        }
      }

      expect(normalizeTurnDraftState(state, runtime.traceId, turn, status, error)).toEqual(
        fixture.data.expected_turn_draft,
      );
    });
  }
});

function normalizeTurnDraftState(
  state: ChatStateStore,
  traceId: string,
  turn: number,
  status: SessionTurnDraft['status'],
  error?: string,
): SessionTurnDraft {
  const view = buildChatStateView(state);
  return {
    trace_id: traceId,
    turn,
    status,
    ...(status === 'error' && error ? { error } : {}),
    pending_questions: view.pendingQuestions.map((question) => ({
      question_id: question.questionId,
      prompt: question.content,
      ...(question.selectionMode ? { selection_mode: question.selectionMode } : {}),
      ...(question.options?.length ? { options: question.options } : {}),
    })),
    assistant_segments: view.streamingAssistantSegments.map((segment) => ({
      id: segment.id,
      content: segment.content,
    })),
    thinking_segments: view.streamingThinkingSegments.map((segment) => ({
      id: segment.id,
      content: segment.content,
    })),
    tools: view.streamingTools.map((tool) => ({
      id: tool.id,
      content: tool.content,
      ...(tool.toolInput ? { tool_input: tool.toolInput } : {}),
      ...(tool.toolName ? { tool_name: tool.toolName } : {}),
      ...(tool.toolStatus ? { tool_status: tool.toolStatus } : {}),
      ...(tool.toolCallId ? { tool_call_id: tool.toolCallId } : {}),
      ...(tool.traceId ? { trace_id: tool.traceId } : {}),
    })),
    item_order: [...state.streamingItemOrder],
  };
}

function readErrorMessage(payload: Record<string, unknown>): string | undefined {
  return typeof payload.message === 'string' && payload.message ? payload.message : undefined;
}

function loadTurnDraftFixtures(): Array<{ name: string; data: TurnDraftFixture }> {
  const fixtureDir = path.resolve(__dirname, '../../../../core/shared/fixtures/turn_draft');
  return fs.readdirSync(fixtureDir)
    .filter((entry) => entry.endsWith('.json'))
    .sort((left, right) => left.localeCompare(right))
    .map((entry) => ({
      name: entry,
      data: JSON.parse(fs.readFileSync(path.join(fixtureDir, entry), 'utf8')) as TurnDraftFixture,
    }));
}
