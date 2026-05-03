import type { MousePositionResponse } from '@/lib/api/tools/mousePosition';
import {
  normalizeClickComposerParams,
  withClickComposerParams,
  type ScreenControlComposerStep,
} from '@/lib/workflow-editor';

export const CLICK_POSITION_TYPE_KEY = 'position_type';
export const CLICK_DISPLAY_ID_KEY = 'display_id';
export const CLICK_COORDINATE_REF_KEY = 'coordinate_ref';
export const POSITION_TYPE_RELATIVE = 'relative';
export const POSITION_TYPE_ABSOLUTE = 'absolute';
export const CLICK_MOUSE_POLL_MS = 120;
export const CLICK_COORDINATE_SOURCE_MANUAL = 'manual';
export const CLICK_COORDINATE_SOURCE_FIND_ICON = 'find_icon';
export const SCREEN_CONTROL_FIND_ICON_COORDINATE_REF = '${find_icon}';

export type ClickPositionType =
  | typeof POSITION_TYPE_RELATIVE
  | typeof POSITION_TYPE_ABSOLUTE;

export type ClickCoordinateSource =
  | typeof CLICK_COORDINATE_SOURCE_MANUAL
  | typeof CLICK_COORDINATE_SOURCE_FIND_ICON;

export interface ClickEditorState {
  x: string;
  y: string;
  coordinateSource: ClickCoordinateSource;
  positionType: ClickPositionType;
  displayID?: number | null;
}

export function buildInitialEditorState(
  step: ScreenControlComposerStep,
): ClickEditorState {
  const params = normalizeClickComposerParams(step.params);
  const source = asRecord(step.params);
  return {
    x: params === undefined ? '' : String(params.x),
    y: params === undefined ? '' : String(params.y),
    coordinateSource: parseCoordinateSource(source[CLICK_COORDINATE_REF_KEY]),
    positionType: parsePositionType(source[CLICK_POSITION_TYPE_KEY]),
    displayID: parseOptionalDisplayID(source[CLICK_DISPLAY_ID_KEY]),
  };
}

export function buildSavedClickStep(
  step: ScreenControlComposerStep,
  state: ClickEditorState,
): ScreenControlComposerStep {
  if (state.coordinateSource === CLICK_COORDINATE_SOURCE_FIND_ICON) {
    return buildFindIconReferenceClickStep(step);
  }
  const baseStep = withClickComposerParams(step, {
    x: parseCoordinate(state.x, 'x'),
    y: parseCoordinate(state.y, 'y'),
  });
  const nextParams = {
    ...asRecord(baseStep.params),
    [CLICK_POSITION_TYPE_KEY]: state.positionType,
  } as Record<string, unknown>;
  const nextDisplayID = state.positionType === POSITION_TYPE_RELATIVE
    ? null
    : state.displayID;
  if (nextDisplayID === undefined) {
    delete nextParams[CLICK_DISPLAY_ID_KEY];
  } else {
    nextParams[CLICK_DISPLAY_ID_KEY] = nextDisplayID;
  }
  return {
    ...baseStep,
    params: nextParams,
  };
}

export function applyMousePositionToState(
  state: ClickEditorState,
  position: MousePositionResponse,
): ClickEditorState {
  return {
    ...state,
    x: String(position.x),
    y: String(position.y),
    coordinateSource: CLICK_COORDINATE_SOURCE_MANUAL,
    positionType: POSITION_TYPE_ABSOLUTE,
    displayID: position.display_id ?? null,
  };
}

export function buildStepTag(stepIndex: number): string {
  const id = String(stepIndex + 1).padStart(4, '0');
  return `STEP ID: CLICK-${id}`;
}

export function parseCoordinate(raw: string, field: 'x' | 'y'): number {
  const trimmed = raw.trim();
  if (!trimmed) {
    throw new Error(`${field} is required`);
  }
  const value = Number(trimmed);
  if (!Number.isFinite(value)) {
    throw new Error(`${field} must be a finite number`);
  }
  return value;
}

export function parsePositionType(raw: unknown): ClickPositionType {
  if (raw === POSITION_TYPE_ABSOLUTE) {
    return POSITION_TYPE_ABSOLUTE;
  }
  return POSITION_TYPE_RELATIVE;
}

function parseCoordinateSource(raw: unknown): ClickCoordinateSource {
  if (raw === SCREEN_CONTROL_FIND_ICON_COORDINATE_REF) {
    return CLICK_COORDINATE_SOURCE_FIND_ICON;
  }
  return CLICK_COORDINATE_SOURCE_MANUAL;
}

function buildFindIconReferenceClickStep(
  step: ScreenControlComposerStep,
): ScreenControlComposerStep {
  const nextParams = { ...asRecord(step.params) };
  nextParams[CLICK_COORDINATE_REF_KEY] = SCREEN_CONTROL_FIND_ICON_COORDINATE_REF;
  delete nextParams.x;
  delete nextParams.y;
  delete nextParams[CLICK_POSITION_TYPE_KEY];
  delete nextParams[CLICK_DISPLAY_ID_KEY];
  return {
    ...step,
    action: 'click',
    params: nextParams,
  };
}

function parseOptionalDisplayID(raw: unknown): number | null | undefined {
  if (raw === null) {
    return null;
  }
  if (typeof raw !== 'number' || !Number.isInteger(raw) || raw < 0) {
    return undefined;
  }
  return raw;
}

function asRecord(input: unknown): Record<string, unknown> {
  if (typeof input !== 'object' || input === null || Array.isArray(input)) {
    return {};
  }
  return input as Record<string, unknown>;
}
