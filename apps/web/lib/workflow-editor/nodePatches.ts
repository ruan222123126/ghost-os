import type { WorkflowInputVariable } from '@/lib/types';
import { normalizeScreenControlComposerAction } from '@/lib/workflow-editor/screenControlComposer';
import type {
  ScreenControlComposerStep,
  WorkflowCanvasNodeDraft,
} from '@/lib/workflow-editor/types';

const DEFAULT_NEW_INPUT: WorkflowInputVariable = {
  name: 'new_variable',
  type: 'string',
  required: false,
};

const EMPTY_SYSTEM_PROMPT = '';
const DEFAULT_TOOL_ARGUMENTS_MODE = 'kv' as const;
const EMPTY_TOOL_NAME = '';

export function withAddedStartInput(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft {
  const inputs = [...(node.start?.inputs ?? []), { ...DEFAULT_NEW_INPUT }];
  return { ...node, start: { inputs } };
}

export function withRemovedStartInput(node: WorkflowCanvasNodeDraft, index: number): WorkflowCanvasNodeDraft {
  const inputs = (node.start?.inputs ?? []).filter((_, inputIndex) => inputIndex !== index);
  return { ...node, start: { inputs } };
}

export function withUpdatedStartInput(
  node: WorkflowCanvasNodeDraft,
  index: number,
  nextInput: WorkflowInputVariable,
): WorkflowCanvasNodeDraft {
  const inputs = (node.start?.inputs ?? []).map((input, inputIndex) => (
    inputIndex === index ? nextInput : input
  ));
  return { ...node, start: { inputs } };
}

export function withRenamedStartInput(
  node: WorkflowCanvasNodeDraft,
  index: number,
  name: string,
): WorkflowCanvasNodeDraft {
  const inputs = (node.start?.inputs ?? []).map((input, inputIndex) => (
    inputIndex === index ? { ...input, name } : input
  ));
  return { ...node, start: { inputs } };
}

export function withLLMPrompt(node: WorkflowCanvasNodeDraft, prompt: string): WorkflowCanvasNodeDraft {
  return { ...node, llm: { prompt, system_prompt: node.llm?.system_prompt ?? EMPTY_SYSTEM_PROMPT } };
}

export function withLLMSystemPrompt(node: WorkflowCanvasNodeDraft, systemPrompt: string): WorkflowCanvasNodeDraft {
  return { ...node, llm: { prompt: node.llm?.prompt ?? '', system_prompt: systemPrompt } };
}

export function withAgentMessage(node: WorkflowCanvasNodeDraft, message: string): WorkflowCanvasNodeDraft {
  return { ...node, agent: { message } };
}

export function withToolName(node: WorkflowCanvasNodeDraft, toolName: string): WorkflowCanvasNodeDraft {
  return {
    ...node,
    tool: {
      tool_name: toolName,
      arguments: node.tool?.arguments ?? {},
    },
  };
}

export function withToolArguments(
  node: WorkflowCanvasNodeDraft,
  argumentsValue: Record<string, unknown>,
): WorkflowCanvasNodeDraft {
  return {
    ...node,
    tool: {
      tool_name: node.tool?.tool_name ?? EMPTY_TOOL_NAME,
      arguments: argumentsValue,
    },
  };
}

export function withToolArgumentsMode(
  node: WorkflowCanvasNodeDraft,
  mode: WorkflowCanvasNodeDraft['ui']['toolArgumentsMode'],
): WorkflowCanvasNodeDraft {
  return {
    ...node,
    ui: {
      ...node.ui,
      toolArgumentsMode: mode ?? DEFAULT_TOOL_ARGUMENTS_MODE,
    },
  };
}

export function withScreenControlComposerSteps(
  node: WorkflowCanvasNodeDraft,
  steps: ScreenControlComposerStep[],
): WorkflowCanvasNodeDraft {
  if (steps.length === 0) {
    return {
      ...node,
      ui: {
        ...node.ui,
        screenControlComposer: undefined,
      },
    };
  }
  return {
    ...node,
    ui: {
      ...node.ui,
      screenControlComposer: {
        steps: cloneScreenControlComposerSteps(steps),
      },
    },
  };
}

function cloneScreenControlComposerSteps(steps: ScreenControlComposerStep[]): ScreenControlComposerStep[] {
  return steps.map((step) => ({
    action: normalizeScreenControlComposerAction(step.action),
    params: step.params ? { ...step.params } : undefined,
  }));
}
