import type {
  WorkflowDefinition,
  WorkflowInputVariable,
  WorkflowNode,
} from '@/lib/types';
import { cloneWorkflowTaskRuntimeOverrides } from '@/lib/workflow-editor/agentRuntime';
import {
  NODE_X_GAP,
  NODE_Y_BASE,
} from '@/lib/workflow-editor/constants';
import type { WorkflowCanvasNodeSource } from '@/lib/workflow-editor/draftNodeSource';
import { expandLoopPairsForDraft } from '@/lib/workflow-editor/draftLoopExpand';
import { normalizeScreenControlComposerAction } from '@/lib/workflow-editor/screenControlComposer';
import type {
  ScreenControlComposerStep,
  WorkflowCanvasEdgeDraft,
  WorkflowCanvasNodeDraft,
} from '@/lib/workflow-editor/types';

const DEFAULT_TOOL_ARGUMENTS_MODE = 'kv';
const SCREEN_CONTROL_TOOL_NAME = 'screen_control';
const SCREEN_CONTROL_WORKFLOW_STEPS_KEY = 'workflow_steps';
const SCREEN_CONTROL_ACTION_KEY = 'action';
const SCREEN_CONTROL_PARAMS_KEY = 'params';
const SCREEN_CONTROL_COMPOSER_ACTIONS = new Set(['screenshot', 'find_text', 'find_icon', 'click']);

export function buildCanvasGraphFromWorkflowDefinition(workflow: WorkflowDefinition): {
  nodes: WorkflowCanvasNodeDraft[];
  edges: WorkflowCanvasEdgeDraft[];
} {
  const expanded = expandLoopPairsForDraft(workflow, workflowNodeToCanvasSource);
  return {
    nodes: expanded.nodes.map((node, index) => createCanvasNode(node, index)),
    edges: expanded.edges.map((edge, index) => buildEdgeDraft(edge.from_node_id, edge.to_node_id, index)),
  };
}
function createCanvasNode(node: WorkflowCanvasNodeSource, index: number): WorkflowCanvasNodeDraft {
  const screenControlComposer = extractScreenControlComposer(node.tool);
  return {
    id: node.id,
    type: node.type,
    position: {
      x: index * NODE_X_GAP,
      y: NODE_Y_BASE,
    },
    ui: {
      toolArgumentsMode: DEFAULT_TOOL_ARGUMENTS_MODE,
      screenControlComposer,
    },
    start: cloneStartNodePayload(node.start),
    tool: cloneToolNodePayload(node.tool),
    llm: cloneLLMNodePayload(node.llm),
    agent: cloneAgentNodePayload(node.agent),
    if: cloneIfNodePayload(node.if),
    loop: cloneLoopNodePayload(node.loop),
  };
}

function extractScreenControlComposer(
  tool: WorkflowCanvasNodeSource['tool'],
): WorkflowCanvasNodeDraft['ui']['screenControlComposer'] {
  if (!tool || tool.tool_name.trim() !== SCREEN_CONTROL_TOOL_NAME) {
    return undefined;
  }
  const args = asRecord(tool.arguments);
  const workflowSteps = readScreenControlWorkflowSteps(args[SCREEN_CONTROL_WORKFLOW_STEPS_KEY]);
  if (workflowSteps.length > 0) {
    return { steps: workflowSteps };
  }
  const singleStep = readScreenControlSingleStep(args);
  if (!singleStep) {
    return undefined;
  }
  return { steps: [singleStep] };
}

function readScreenControlWorkflowSteps(input: unknown): ScreenControlComposerStep[] {
  if (!Array.isArray(input)) {
    return [];
  }
  const steps = input
    .map((item) => readScreenControlComposerStep(asRecord(item)))
    .filter((step): step is ScreenControlComposerStep => !!step);
  return steps;
}

function readScreenControlSingleStep(args: Record<string, unknown>): ScreenControlComposerStep | undefined {
  return readScreenControlComposerStep({
    action: args[SCREEN_CONTROL_ACTION_KEY],
    params: args[SCREEN_CONTROL_PARAMS_KEY],
  });
}

function readScreenControlComposerStep(input: Record<string, unknown>): ScreenControlComposerStep | undefined {
  const action = readScreenControlComposerAction(input.action);
  if (!action) {
    return undefined;
  }
  const params = cloneObject(asRecord(input.params));
  if (!params || Object.keys(params).length === 0) {
    return { action };
  }
  return { action, params };
}

function readScreenControlComposerAction(input: unknown): ScreenControlComposerStep['action'] | undefined {
  if (typeof input !== 'string') {
    return undefined;
  }
  const normalized = normalizeScreenControlComposerAction(input.trim() as ScreenControlComposerStep['action']);
  if (!SCREEN_CONTROL_COMPOSER_ACTIONS.has(normalized)) {
    return undefined;
  }
  return normalized;
}
function workflowNodeToCanvasSource(node: WorkflowNode): WorkflowCanvasNodeSource {
  return {
    id: node.id,
    type: node.type,
    start: cloneStartNodePayload(node.start),
    tool: cloneToolNodePayload(node.tool),
    llm: cloneLLMNodePayload(node.llm),
    agent: cloneAgentNodePayload(node.agent),
    if: cloneIfNodePayload(node.if),
    loop: undefined,
  };
}
function buildEdgeDraft(fromNodeID: string, toNodeID: string, index: number): WorkflowCanvasEdgeDraft {
  return {
    id: `edge-${index}-${fromNodeID}-${toNodeID}`,
    from_node_id: fromNodeID,
    to_node_id: toNodeID,
  };
}
function cloneInputs(inputs?: WorkflowInputVariable[]): WorkflowInputVariable[] | undefined {
  if (!inputs) {
    return undefined;
  }
  return inputs.map((input) => ({ ...input }));
}
function cloneObject(input?: Record<string, unknown>): Record<string, unknown> | undefined {
  if (!input) {
    return undefined;
  }
  return { ...input };
}

function cloneStartNodePayload(
  start: WorkflowCanvasNodeSource['start'],
): WorkflowCanvasNodeSource['start'] {
  return start ? { inputs: cloneInputs(start.inputs) } : undefined;
}

function cloneToolNodePayload(tool: WorkflowCanvasNodeSource['tool']): WorkflowCanvasNodeSource['tool'] {
  if (!tool) {
    return undefined;
  }
  return {
    tool_name: tool.tool_name,
    arguments: cloneObject(tool.arguments),
  };
}

function cloneLLMNodePayload(llm: WorkflowCanvasNodeSource['llm']): WorkflowCanvasNodeSource['llm'] {
  if (!llm) {
    return undefined;
  }
  return {
    prompt: llm.prompt,
    system_prompt: llm.system_prompt,
  };
}

function cloneAgentNodePayload(agent: WorkflowCanvasNodeSource['agent']): WorkflowCanvasNodeSource['agent'] {
  if (!agent) {
    return undefined;
  }
  return {
    message: agent.message,
    runtime_overrides: cloneWorkflowTaskRuntimeOverrides(agent.runtime_overrides),
  };
}

function cloneIfNodePayload(ifConfig: WorkflowCanvasNodeSource['if']): WorkflowCanvasNodeSource['if'] {
  if (!ifConfig) {
    return undefined;
  }
  return {
    source_node_id: ifConfig.source_node_id,
    operator: ifConfig.operator,
    value: ifConfig.value,
    true_node_id: ifConfig.true_node_id,
    false_node_id: ifConfig.false_node_id,
  };
}

function cloneLoopNodePayload(loop: WorkflowCanvasNodeSource['loop']): WorkflowCanvasNodeSource['loop'] {
  if (!loop) {
    return undefined;
  }
  return {
    role: loop.role,
    loop_id: loop.loop_id,
    max_iterations: loop.max_iterations,
    body_node_id: loop.body_node_id,
    exit_node_id: loop.exit_node_id,
  };
}

function asRecord(value: unknown): Record<string, unknown> {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return {};
  }
  return value as Record<string, unknown>;
}
