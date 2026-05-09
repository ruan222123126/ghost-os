import { cloneWorkflowTaskRuntimeOverrides } from '@/lib/workflow-editor/agentRuntime';
import { DEFAULT_ORCHESTRATION_GROUP_MAX_ROUNDS } from '@/lib/workflow-editor/constants';
import { normalizeScreenControlComposerAction } from '@/lib/workflow-editor/screenControlComposer';
import type {
  ScreenControlComposerStep,
  WorkflowCanvasNodeDraft,
} from '@/lib/workflow-editor/types';

const EMPTY_SYSTEM_PROMPT = '';
const DEFAULT_TOOL_ARGUMENTS_MODE = 'kv' as const;
const EMPTY_TOOL_NAME = '';
const DEFAULT_GROUP_SPEAKING_MODE = 'sequential' as const;
const DEFAULT_GROUP_MAX_ROUNDS = DEFAULT_ORCHESTRATION_GROUP_MAX_ROUNDS;

export function withLLMPrompt(node: WorkflowCanvasNodeDraft, prompt: string): WorkflowCanvasNodeDraft {
  return { ...node, llm: { prompt, system_prompt: node.llm?.system_prompt ?? EMPTY_SYSTEM_PROMPT } };
}

export function withLLMSystemPrompt(node: WorkflowCanvasNodeDraft, systemPrompt: string): WorkflowCanvasNodeDraft {
  return { ...node, llm: { prompt: node.llm?.prompt ?? '', system_prompt: systemPrompt } };
}

export function withAgentMessage(node: WorkflowCanvasNodeDraft, message: string): WorkflowCanvasNodeDraft {
  return {
    ...node,
    agent: {
      title: node.agent?.title,
      message,
      runtime_overrides: cloneWorkflowTaskRuntimeOverrides(node.agent?.runtime_overrides),
    },
  };
}

export function withAgentTitle(node: WorkflowCanvasNodeDraft, title: string): WorkflowCanvasNodeDraft {
  return {
    ...node,
    agent: {
      title,
      message: node.agent?.message ?? '',
      runtime_overrides: cloneWorkflowTaskRuntimeOverrides(node.agent?.runtime_overrides),
    },
  };
}

export function withAgentRuntimeOverrides(
  node: WorkflowCanvasNodeDraft,
  runtimeOverrides?: NonNullable<WorkflowCanvasNodeDraft['agent']>['runtime_overrides'],
): WorkflowCanvasNodeDraft {
  return {
    ...node,
    agent: {
      title: node.agent?.title,
      message: node.agent?.message ?? '',
      runtime_overrides: cloneWorkflowTaskRuntimeOverrides(runtimeOverrides),
    },
  };
}

export function withGroupNode(
  node: WorkflowCanvasNodeDraft,
  patch: Partial<NonNullable<WorkflowCanvasNodeDraft['group']>>,
): WorkflowCanvasNodeDraft {
  return {
    ...node,
    group: {
      title: patch.title ?? node.group?.title ?? '',
      shared_context: patch.shared_context ?? node.group?.shared_context ?? '',
      speaking_mode: patch.speaking_mode ?? node.group?.speaking_mode ?? DEFAULT_GROUP_SPEAKING_MODE,
      owner_agent_id: patch.owner_agent_id ?? node.group?.owner_agent_id ?? '',
      max_rounds: patch.max_rounds ?? node.group?.max_rounds ?? DEFAULT_GROUP_MAX_ROUNDS,
    },
  };
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
