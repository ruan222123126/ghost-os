import { withToolArguments } from '@/lib/workflow-editor/nodePatches';
import { normalizeClickComposerParams, syncClickStepToToolArguments } from '@/lib/workflow-editor/screenControlClick';
import { normalizeScreenControlComposerAction } from '@/lib/workflow-editor/screenControlComposer';
import { normalizeFindIconComposerParams, syncFindIconStepToToolArguments } from '@/lib/workflow-editor/screenControlFindIcon';
import type {
  ScreenControlComposerStep,
  WorkflowCanvasNodeDraft,
} from '@/lib/workflow-editor/types';

const ATOMIC_MODE = 'atomic';
const WORKFLOW_STEPS_KEY = 'workflow_steps';

export function syncScreenControlComposerStepsToToolArguments(
  node: WorkflowCanvasNodeDraft,
  steps: ScreenControlComposerStep[],
): WorkflowCanvasNodeDraft {
  const normalizedSteps = steps.map(normalizeComposerStep);
  if (normalizedSteps.length === 0) {
    return clearScreenControlComposerAction(node);
  }
  if (normalizedSteps.length === 1) {
    return syncScreenControlComposerSingleStep(node, normalizedSteps[0]);
  }
  return syncScreenControlComposerWorkflowSteps(node, normalizedSteps);
}

function syncScreenControlComposerSingleStep(
  node: WorkflowCanvasNodeDraft,
  step: ScreenControlComposerStep,
): WorkflowCanvasNodeDraft {
  const sanitizedNode = withCleanComposerArguments(node);
  switch (step.action) {
    case 'screenshot':
      return withAtomicComposerAction(sanitizedNode, 'screenshot', step.params);
    case 'find_text':
      return withAtomicComposerAction(sanitizedNode, 'find_text', step.params);
    case 'find_icon':
      if (!normalizeFindIconComposerParams(step.params)) {
        return clearScreenControlComposerAction(sanitizedNode);
      }
      return syncFindIconStepToToolArguments(sanitizedNode, step);
    case 'click':
      if (!normalizeClickComposerParams(step.params)) {
        return clearScreenControlComposerAction(sanitizedNode);
      }
      return syncClickStepToToolArguments(sanitizedNode, step);
    default:
      return clearScreenControlComposerAction(sanitizedNode);
  }
}

function syncScreenControlComposerWorkflowSteps(
  node: WorkflowCanvasNodeDraft,
  steps: ScreenControlComposerStep[],
): WorkflowCanvasNodeDraft {
  const sourceArgs = withoutComposerArguments(asRecord(node.tool?.arguments));
  return withToolArguments(node, {
    ...sourceArgs,
    [WORKFLOW_STEPS_KEY]: steps.map((step) => ({
      action: step.action,
      params: cloneParams(step.params),
    })),
  });
}

function withAtomicComposerAction(
  node: WorkflowCanvasNodeDraft,
  action: 'screenshot' | 'find_text',
  params: Record<string, unknown> | undefined,
): WorkflowCanvasNodeDraft {
  const sourceArgs = withoutComposerArguments(asRecord(node.tool?.arguments));
  return withToolArguments(node, {
    ...sourceArgs,
    mode: ATOMIC_MODE,
    action,
    params: cloneParams(params),
  });
}

function clearScreenControlComposerAction(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft {
  const sourceArgs = asRecord(node.tool?.arguments);
  const rest = withoutComposerArguments(sourceArgs);
  return withToolArguments(node, rest);
}

function withCleanComposerArguments(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft {
  return withToolArguments(node, withoutComposerArguments(asRecord(node.tool?.arguments)));
}

function withoutComposerArguments(source: Record<string, unknown>): Record<string, unknown> {
  const {
    action: _action,
    params: _params,
    [WORKFLOW_STEPS_KEY]: _workflowSteps,
    ...rest
  } = source;
  return rest;
}

function normalizeComposerStep(step: ScreenControlComposerStep): ScreenControlComposerStep {
  return {
    action: normalizeScreenControlComposerAction(step.action),
    params: cloneParams(step.params),
  };
}

function cloneParams(params: Record<string, unknown> | undefined): Record<string, unknown> {
  return { ...asRecord(params) };
}

function asRecord(input: unknown): Record<string, unknown> {
  if (typeof input !== 'object' || input === null || Array.isArray(input)) {
    return {};
  }
  return input as Record<string, unknown>;
}
