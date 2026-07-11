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

type NodePayloadFactory = (
  source: Partial<WorkflowCanvasNodeDraft> | undefined,
) => Pick<
  WorkflowCanvasNodeDraft,
  'start' | 'tool' | 'llm' | 'agent' | 'group' | 'if' | 'loop'
>;

const NODE_PAYLOAD_FACTORIES = {
  start: (source) => ({ start: source?.start ?? { inputs: [] } }),
  end: () => ({}),
  tool: (source) => ({ tool: source?.tool ?? { tool_name: '', arguments: {} } }),
  llm: (source) => ({ llm: source?.llm ?? { prompt: '', system_prompt: '' } }),
  agent: (source) => ({ agent: source?.agent ?? { message: '', title: '' } }),
  group: (source) => ({ group: source?.group ?? buildDefaultGroupConfig() }),
  if: (source) => ({ if: source?.if ?? buildDefaultIfConfig() }),
  loop: (source) => ({ loop: source?.loop ?? buildDefaultLoopConfig() }),
} satisfies Record<WorkflowNodeType, NodePayloadFactory>;

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
    ...NODE_PAYLOAD_FACTORIES[type](source),
  };
}

function buildDefaultGroupConfig(): NonNullable<WorkflowCanvasNodeDraft['group']> {
  return {
    title: '',
    shared_context: '',
    speaking_mode: 'sequential',
    max_rounds: DEFAULT_ORCHESTRATION_GROUP_MAX_ROUNDS,
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

function buildDefaultLoopConfig(): NonNullable<WorkflowCanvasNodeDraft['loop']> {
  return {
    role: LOOP_ROLE_START,
    loop_id: '',
    max_iterations: DEFAULT_LOOP_MAX_ITERATIONS,
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
