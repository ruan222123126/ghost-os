import type { WebLocale } from '@/lib/i18n/locale';

const DEFAULT_TASK_RUN_STATUS = 'running';

const ZH_TASK_RUN_STATUS_LABELS: Record<string, string> = {
  awaiting_human: '等待人工',
  cancelled: '已取消',
  error: '错误',
  incomplete: '未完成',
  running: '运行中',
  skipped: '已跳过',
  success: '成功',
};

export function formatTaskRunTimestamp(value: string | undefined): string {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toISOString().slice(0, 19).replace('T', ' ');
}

export function formatTaskRunStatus(status: string | undefined, locale: WebLocale): string {
  const normalized = status?.trim() || DEFAULT_TASK_RUN_STATUS;
  if (locale !== 'zh-CN') {
    return normalized;
  }
  return ZH_TASK_RUN_STATUS_LABELS[normalized] ?? normalized;
}
