import type {
  ScreenControlClickParams,
  ScreenControlComposerStep,
  WorkflowCanvasNodeDraft,
} from '@/lib/workflow-editor/types';
import { withToolArguments } from '@/lib/workflow-editor/nodePatches';

const ATOMIC_MODE = 'atomic';
const CLICK_ACTION = 'click';
const CLICK_ICON_ACTION = 'click_icon';
const SCREEN_CONTROL_PARAMS_KEY = 'params';

export function normalizeClickComposerParams(
  params: Record<string, unknown> | undefined,
): ScreenControlClickParams | undefined {
  const source = asRecord(params);
  const x = readCoordinate(source.x);
  const y = readCoordinate(source.y);
  if (x === undefined || y === undefined) {
    return undefined;
  }
  return { x, y };
}

export function withClickComposerParams(
  step: ScreenControlComposerStep,
  params: ScreenControlClickParams,
): ScreenControlComposerStep {
  const source = asRecord(step.params);
  const nextParams: Record<string, unknown> = { ...source, x: params.x, y: params.y };
  return {
    ...step,
    action: CLICK_ACTION,
    params: nextParams,
  };
}

export function syncClickStepToToolArguments(
  node: WorkflowCanvasNodeDraft,
  step: ScreenControlComposerStep,
): WorkflowCanvasNodeDraft {
  const clickParams = normalizeClickComposerParams(step.params);
  if (step.action !== CLICK_ACTION || !clickParams) {
    return node;
  }

  const sourceArgs = asRecord(node.tool?.arguments);
  const sourceParams = asRecord(sourceArgs[SCREEN_CONTROL_PARAMS_KEY]);
  const nextParams: Record<string, unknown> = {
    ...sourceParams,
    x: clickParams.x,
    y: clickParams.y,
  };

  return withToolArguments(node, {
    ...sourceArgs,
    mode: ATOMIC_MODE,
    action: CLICK_ICON_ACTION,
    params: nextParams,
  });
}

function asRecord(input: unknown): Record<string, unknown> {
  if (typeof input !== 'object' || input === null || Array.isArray(input)) {
    return {};
  }
  return input as Record<string, unknown>;
}

function readCoordinate(input: unknown): number | undefined {
  if (typeof input !== 'number' || !Number.isFinite(input)) {
    return undefined;
  }
  return input;
}
