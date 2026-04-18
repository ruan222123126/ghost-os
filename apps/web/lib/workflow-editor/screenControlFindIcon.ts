import type {
  ScreenControlComposerStep,
  ScreenControlFindIconParams,
  WorkflowCanvasNodeDraft,
} from '@/lib/workflow-editor/types';
import { withToolArguments } from '@/lib/workflow-editor/nodePatches';

const ATOMIC_MODE = 'atomic';
const FIND_ICON_ACTION = 'find_icon';
const SCREEN_CONTROL_PARAMS_KEY = 'params';
const HOVER_AFTER_MATCH_KEY = 'hover_after_match';

export function normalizeFindIconComposerParams(
  params: Record<string, unknown> | undefined,
): ScreenControlFindIconParams | undefined {
  const source = asRecord(params);
  const templatePath = readNonEmptyString(source.template_path);
  if (!templatePath) {
    return undefined;
  }

  const normalized: ScreenControlFindIconParams = { template_path: templatePath };
  const templateName = readNonEmptyString(source.template_name);
  const threshold = readOptionalNumber(source.threshold);
  const maxResults = readOptionalInteger(source.max_results);

  if (templateName) {
    normalized.template_name = templateName;
  }
  if (threshold !== undefined) {
    normalized.threshold = threshold;
  }
  if (maxResults !== undefined) {
    normalized.max_results = maxResults;
  }
  return normalized;
}

export function withFindIconComposerParams(
  step: ScreenControlComposerStep,
  params: ScreenControlFindIconParams,
): ScreenControlComposerStep {
  const source = asRecord(step.params);
  const nextParams: Record<string, unknown> = { ...source };
  nextParams.template_path = params.template_path;
  setOptionalString(nextParams, 'template_name', params.template_name);
  setOptionalNumber(nextParams, 'threshold', params.threshold);
  setOptionalNumber(nextParams, 'max_results', params.max_results);
  return {
    ...step,
    action: FIND_ICON_ACTION,
    params: nextParams,
  };
}

export function syncFindIconStepToToolArguments(
  node: WorkflowCanvasNodeDraft,
  step: ScreenControlComposerStep,
): WorkflowCanvasNodeDraft {
  const findIconParams = normalizeFindIconComposerParams(step.params);
  if (step.action !== FIND_ICON_ACTION || !findIconParams) {
    return node;
  }

  const sourceArgs = asRecord(node.tool?.arguments);
  const sourceParams = asRecord(sourceArgs[SCREEN_CONTROL_PARAMS_KEY]);
  const nextParams: Record<string, unknown> = {
    ...sourceParams,
    template_path: findIconParams.template_path,
  };
  setOptionalNumber(nextParams, 'threshold', findIconParams.threshold);
  setOptionalNumber(nextParams, 'max_results', findIconParams.max_results);
  setHoverAfterMatch(nextParams, asRecord(step.params)[HOVER_AFTER_MATCH_KEY] === true);

  return withToolArguments(node, {
    ...sourceArgs,
    mode: ATOMIC_MODE,
    action: FIND_ICON_ACTION,
    params: nextParams,
  });
}

function asRecord(input: unknown): Record<string, unknown> {
  if (typeof input !== 'object' || input === null || Array.isArray(input)) {
    return {};
  }
  return input as Record<string, unknown>;
}

function readNonEmptyString(input: unknown): string | undefined {
  if (typeof input !== 'string') {
    return undefined;
  }
  const value = input.trim();
  return value.length > 0 ? value : undefined;
}

function readOptionalNumber(input: unknown): number | undefined {
  if (typeof input !== 'number' || !Number.isFinite(input)) {
    return undefined;
  }
  if (input < 0 || input > 1) {
    return undefined;
  }
  return input;
}

function readOptionalInteger(input: unknown): number | undefined {
  if (typeof input !== 'number' || !Number.isFinite(input)) {
    return undefined;
  }
  const value = Math.floor(input);
  if (value < 1) {
    return undefined;
  }
  return value;
}

function setOptionalString(target: Record<string, unknown>, key: string, value?: string) {
  if (!value || value.trim().length === 0) {
    delete target[key];
    return;
  }
  target[key] = value.trim();
}

function setOptionalNumber(target: Record<string, unknown>, key: string, value?: number) {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    delete target[key];
    return;
  }
  target[key] = value;
}

function setHoverAfterMatch(target: Record<string, unknown>, hoverAfterMatch: boolean) {
  if (!hoverAfterMatch) {
    delete target[HOVER_AFTER_MATCH_KEY];
    return;
  }
  target[HOVER_AFTER_MATCH_KEY] = true;
}
