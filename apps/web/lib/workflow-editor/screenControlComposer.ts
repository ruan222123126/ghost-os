import type {
  ScreenControlAtomicAction,
  ScreenControlComposerStep,
} from '@/lib/workflow-editor/types';

const FIND_TEXT_ACTION = 'find_text';
const CLICK_ACTION = 'click';
const LEGACY_CLICK_TEXT_ACTION = 'click_text';
const LEGACY_CLICK_ICON_ACTION = 'click_icon';

export const SCREEN_CONTROL_COMPOSER_ACTIONS: ScreenControlAtomicAction[] = [
  'screenshot',
  FIND_TEXT_ACTION,
  'find_icon',
  CLICK_ACTION,
];

export function appendScreenControlComposerStep(
  steps: ScreenControlComposerStep[],
  action: ScreenControlAtomicAction,
): ScreenControlComposerStep[] {
  return [...steps, { action: normalizeScreenControlComposerAction(action) }];
}

export function removeScreenControlComposerStep(
  steps: ScreenControlComposerStep[],
  index: number,
): ScreenControlComposerStep[] {
  if (index < 0 || index >= steps.length) {
    return steps;
  }
  return steps.filter((_, stepIndex) => stepIndex !== index);
}

export function moveScreenControlComposerStep(
  steps: ScreenControlComposerStep[],
  index: number,
  direction: 'up' | 'down',
): ScreenControlComposerStep[] {
  const targetIndex = direction === 'up' ? index - 1 : index + 1;
  if (index < 0 || index >= steps.length || targetIndex < 0 || targetIndex >= steps.length) {
    return steps;
  }
  const next = [...steps];
  const temp = next[index];
  next[index] = next[targetIndex];
  next[targetIndex] = temp;
  return next;
}

export function updateScreenControlComposerStep(
  steps: ScreenControlComposerStep[],
  index: number,
  step: ScreenControlComposerStep,
): ScreenControlComposerStep[] {
  if (index < 0 || index >= steps.length) {
    return steps;
  }
  return steps.map((item, stepIndex) => (stepIndex === index ? cloneComposerStep(step) : item));
}

export function normalizeScreenControlComposerAction(
  action: ScreenControlAtomicAction,
): ScreenControlAtomicAction {
  if (action === LEGACY_CLICK_ICON_ACTION) {
    return CLICK_ACTION;
  }
  if (action === LEGACY_CLICK_TEXT_ACTION) {
    return FIND_TEXT_ACTION;
  }
  return action;
}

function cloneComposerStep(step: ScreenControlComposerStep): ScreenControlComposerStep {
  return {
    action: normalizeScreenControlComposerAction(step.action),
    params: step.params ? { ...step.params } : undefined,
  };
}
