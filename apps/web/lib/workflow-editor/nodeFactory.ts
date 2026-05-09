import type {
  ScreenControlComposerStep,
  WorkflowCanvasNodeDraft,
  WorkflowCanvasPosition,
  WorkflowNodeType,
} from '@/lib/workflow-editor/types';
import { normalizeScreenControlComposerAction } from '@/lib/workflow-editor/screenControlComposer';
import {
  DEFAULT_LOOP_MAX_ITERATIONS,
  DEFAULT_ORCHESTRATION_GROUP_MAX_ROUNDS,
  LOOP_ROLE_START,
} from '@/lib/workflow-editor/constants';

const NEW_NODE_X_GAP = 260;
const NEW_NODE_Y = 250;

export interface CreateDraftNodeInput {
  id: string;
  type: WorkflowNodeType;
  index: number;
  source?: Partial<WorkflowCanvasNodeDraft>;
}

export function createDefaultNodePosition(index: number): WorkflowCanvasPosition {
  return { x: index * NEW_NODE_X_GAP, y: NEW_NODE_Y };
}

export function createNextNodePosition(position: WorkflowCanvasPosition): WorkflowCanvasPosition {
  return { x: position.x + NEW_NODE_X_GAP, y: position.y };
}

export function createDraftNode(input: CreateDraftNodeInput): WorkflowCanvasNodeDraft {
  const { id, type, index, source } = input;
  return {
    id,
    type,
    position: source?.position ?? createDefaultNodePosition(index),
    ui: buildNodeUI(source?.ui),
    start: type === 'start' ? source?.start ?? { inputs: [] } : undefined,
    tool: type === 'tool' ? source?.tool ?? { tool_name: '', arguments: {} } : undefined,
    llm: type === 'llm' ? source?.llm ?? { prompt: '', system_prompt: '' } : undefined,
    agent: type === 'agent' ? source?.agent ?? { message: '', title: '' } : undefined,
    group: type === 'group'
      ? source?.group ?? {
        title: '',
        shared_context: '',
        speaking_mode: 'sequential',
        max_rounds: DEFAULT_ORCHESTRATION_GROUP_MAX_ROUNDS,
      }
      : undefined,
    if: type === 'if' ? source?.if ?? buildDefaultIfConfig() : undefined,
    loop: type === 'loop'
      ? source?.loop ?? {
        role: LOOP_ROLE_START,
        loop_id: '',
        max_iterations: DEFAULT_LOOP_MAX_ITERATIONS,
      }
      : undefined,
  };
}

function buildDefaultIfConfig(): NonNullable<WorkflowCanvasNodeDraft['if']> {
  return {
    source_node_id: '',
    operator: 'equals',
    value: '',
    true_node_id: '',
    false_node_id: '',
  };
}

function buildNodeUI(sourceUI?: WorkflowCanvasNodeDraft['ui']): WorkflowCanvasNodeDraft['ui'] {
  return {
    toolArgumentsMode: sourceUI?.toolArgumentsMode ?? 'kv',
    screenControlComposer: cloneScreenControlComposer(sourceUI?.screenControlComposer),
  };
}

function cloneScreenControlComposer(
  composer?: WorkflowCanvasNodeDraft['ui']['screenControlComposer'],
): WorkflowCanvasNodeDraft['ui']['screenControlComposer'] {
  if (!composer || composer.steps.length === 0) {
    return undefined;
  }
  return {
    steps: cloneScreenControlComposerSteps(composer.steps),
  };
}

function cloneScreenControlComposerSteps(
  steps: ScreenControlComposerStep[],
): ScreenControlComposerStep[] {
  return steps.map((step) => ({
    action: normalizeScreenControlComposerAction(step.action),
    params: step.params ? { ...step.params } : undefined,
  }));
}
