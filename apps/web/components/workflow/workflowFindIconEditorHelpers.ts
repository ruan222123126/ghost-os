import {
  normalizeFindIconComposerParams,
  type ScreenControlComposerStep,
} from '@/lib/workflow-editor';
import type { FindIconEditorPanelState } from '@/components/workflow/WorkflowFindIconStepEditorModalView';

const HOVER_AFTER_MATCH_KEY = 'hover_after_match';

export function buildFindIconInitialEditorState(step: ScreenControlComposerStep): FindIconEditorPanelState {
  const params = normalizeFindIconComposerParams(step.params);
  const source = asRecord(step.params);
  return {
    templatePath: params?.template_path ?? '',
    templateName: params?.template_name ?? '',
    hoverAfterMatch: source[HOVER_AFTER_MATCH_KEY] === true,
  };
}

export function buildFindIconStepTag(stepIndex: number): string {
  const id = String(stepIndex + 1).padStart(4, '0');
  return `节点 ID: IMG-TRG-${id}`;
}

export function mergeFindIconHoverAction(
  step: ScreenControlComposerStep,
  hoverAfterMatch: boolean,
): ScreenControlComposerStep {
  return {
    ...step,
    params: {
      ...asRecord(step.params),
      [HOVER_AFTER_MATCH_KEY]: hoverAfterMatch,
    },
  };
}

function asRecord(input: unknown): Record<string, unknown> {
  if (typeof input !== 'object' || input === null || Array.isArray(input)) {
    return {};
  }
  return input as Record<string, unknown>;
}
