import type { WebLocale } from '@/lib/i18n/locale';
import {
  DEFAULT_ORCHESTRATION_GROUP_MAX_ROUNDS,
  DEFAULT_ORCHESTRATION_GROUP_TITLE_PREFIX,
  DEFAULT_ORCHESTRATION_SPEAKING_MODE,
} from '@/lib/workflow-editor/constants';
import type { WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

const DEFAULT_GROUP_SHARED_CONTEXT: Record<WebLocale, string> = {
  'en-US': 'You are now in a group facing other members. You may communicate with others and discuss the problem. You have the right to remain silent, and you also have the right to say anything.',
  'zh-CN': '你们现在在一个群组中，面对着其他人，你们可以跟其他人交流，探讨问题，你们有权保持沉默，也有权说任何话。',
};

export function defaultOrchestrationGroupSharedContext(locale: WebLocale): string {
  return DEFAULT_GROUP_SHARED_CONTEXT[locale];
}

export function buildDefaultOrchestrationGroupNode(
  index: number,
  locale: WebLocale,
): NonNullable<WorkflowCanvasNodeDraft['group']> {
  return {
    title: `${DEFAULT_ORCHESTRATION_GROUP_TITLE_PREFIX} ${index}`,
    shared_context: defaultOrchestrationGroupSharedContext(locale),
    speaking_mode: DEFAULT_ORCHESTRATION_SPEAKING_MODE,
    owner_agent_id: '',
    max_rounds: DEFAULT_ORCHESTRATION_GROUP_MAX_ROUNDS,
  };
}
