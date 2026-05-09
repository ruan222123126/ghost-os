import { localizeWorkflowValidationError } from '@/lib/i18n/workflowValidation';
import type { WebLocale } from '@/lib/i18n/locale';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';

export function buildOrchestrationWorkflowCopy(
  copy: WorkflowCopy,
  locale: WebLocale,
): WorkflowCopy {
  if (locale === 'zh-CN') {
    return {
      ...copy,
      loadingTask: '正在加载编排…',
      watermark: '编排',
      nodeLabels: {
        ...copy.nodeLabels,
        agent: '角色',
        group: '群组',
      },
      closeWorkflowSettingsAria: '关闭编排设置',
      sidebarOpenSettingsAria: '打开编排设置',
      sidebarTitleSub: '编排',
      sidebarReturn: '返回编排设置',
      sidebarSaveAria: (label: string) => `保存编排。${label}`,
      modalTitle: '编排设置',
      modalDescription: '用于编排调度和会话导入的基础画布设置。',
    } as unknown as WorkflowCopy;
  }

  return {
    ...copy,
    loadingTask: 'Loading orchestration...',
    watermark: 'Orchestration',
    nodeLabels: {
      ...copy.nodeLabels,
      agent: 'ROLE',
      group: 'GROUP',
    },
    closeWorkflowSettingsAria: 'Close orchestration settings',
    sidebarOpenSettingsAria: 'Open orchestration settings',
    sidebarTitleSub: 'Orchestration',
    sidebarReturn: 'Back to Orchestration Settings',
    sidebarSaveAria: (label: string) => `Save orchestration. ${label}`,
    modalTitle: 'Orchestration Settings',
    modalDescription: 'Basic canvas controls for orchestration scheduling and session import.',
  } as unknown as WorkflowCopy;
}

export function localizeOrchestrationValidationError(
  message: string,
  locale: WebLocale,
): string {
  const normalizedMessage = message.replaceAll('orchestration', 'workflow');
  const localized = localizeWorkflowValidationError(normalizedMessage, locale);
  if (locale === 'zh-CN') {
    return localized
      .replaceAll('工作流', '编排')
      .replaceAll('orchestration', '编排');
  }
  return localized.replaceAll('workflow', 'orchestration');
}
