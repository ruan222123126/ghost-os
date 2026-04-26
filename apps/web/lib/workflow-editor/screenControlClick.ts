import type {
  ScreenControlClickParams,
  ScreenControlComposerStep,
  WorkflowCanvasNodeDraft,
} from '@/lib/workflow-editor/types';
import { withToolArguments } from '@/lib/workflow-editor/nodePatches';

const ATOMIC_MODE = 'atomic';
const CLICK_ACTION = 'click';
const CLICK_ICON_ACTION = 'click_icon';
const DISPLAY_ID_KEY = 'display_id';
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
  const stepParams = asRecord(step.params);
  const displayIDDirective = readDisplayIDDirective(stepParams);
  const nextParams: Record<string, unknown> = {
    ...sourceParams,
    ...stepParams,
    x: clickParams.x,
    y: clickParams.y,
  };
  if (displayIDDirective.kind === 'set') {
    nextParams[DISPLAY_ID_KEY] = displayIDDirective.value;
  }
  if (displayIDDirective.kind === 'clear') {
    delete nextParams[DISPLAY_ID_KEY];
  }

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

function readDisplayIDDirective(
  params: Record<string, unknown>,
): { kind: 'keep' } | { kind: 'clear' } | { kind: 'set'; value: number } {
  if (!Object.prototype.hasOwnProperty.call(params, DISPLAY_ID_KEY)) {
    return { kind: 'keep' };
  }
  const raw = params[DISPLAY_ID_KEY];
  if (raw === null) {
    return { kind: 'clear' };
  }
  if (typeof raw === 'number' && Number.isInteger(raw) && raw >= 0) {
    return { kind: 'set', value: raw };
  }
  return { kind: 'keep' };
}
