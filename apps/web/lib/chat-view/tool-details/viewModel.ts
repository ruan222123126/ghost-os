import type { ToolChatMessage } from '@/lib/types';
import { normalizeToolName } from '@/lib/toolNames';
import type { ToolCardViewModel, ToolCardViewModelOptions, ToolTone } from '../types';
import { type FormattedToolAction, formatToolAction, formatToolCardDetails } from './format';

const RUNNING_STATUSES = new Set(['running', 'pending', 'in_progress']);
const ERROR_STATUSES = new Set(['error', 'failed']);
const TOOL_FALLBACK_TITLE = 'Tool';
const FILE_ACTION_LABELS = {
  edit: '编辑文件',
  read: '阅读文件',
  write: '创建文件',
} as const;
const SKILL_ACTION_LABELS = {
  load: '加载技能',
  unload: '卸载技能',
} as const;
const SCREEN_ACTION_LABELS = {
  screen: '截取屏幕',
} as const;

export function buildToolCardViewModel(
  tool: ToolChatMessage,
  options: ToolCardViewModelOptions = {},
): ToolCardViewModel {
  const action = formatToolAction(tool);
  const tone = getToolTone(tool.toolStatus);
  const details = formatToolCardDetails(tool, {
    compactOutputEnabled: options.compactOutputEnabled,
  });
  const plainTitle = buildPlainActionTitle(action, tool);

  return {
    title: buildToolCardTitle(action, tool, options),
    tone,
    statusLabel: getStatusLabel(tone, tool.toolStatus),
    details: details || options.preparingDetails || '',
    titleMode: plainTitle ? 'plain' : 'status',
    showTerminalIcon: !plainTitle,
  };
}

export function getToolTone(status?: string): ToolTone {
  const normalized = normalizeStatus(status);
  if (!normalized || RUNNING_STATUSES.has(normalized)) {
    return 'running';
  }
  if (ERROR_STATUSES.has(normalized)) {
    return 'error';
  }
  return 'success';
}

export function getStatusLabel(tone: ToolTone, status?: string): string {
  if (tone === 'running') {
    return 'RUNNING';
  }
  if (tone === 'error') {
    return 'FAILED';
  }
  const normalized = normalizeStatus(status);
  if (!normalized || normalized === 'ok' || normalized === 'done' || normalized === 'completed') {
    return 'SUCCESS';
  }
  return normalized.replaceAll('_', ' ').toUpperCase();
}

function normalizeStatus(status?: string): string {
  return status?.trim().toLowerCase() || '';
}

function buildToolCardTitle(
  action: FormattedToolAction,
  tool: ToolChatMessage,
  options: ToolCardViewModelOptions,
): string {
  const plainTitle = buildPlainActionTitle(action, tool);
  if (plainTitle) {
    return plainTitle;
  }
  if (action.text === TOOL_FALLBACK_TITLE) {
    return options.fallbackTitle ?? TOOL_FALLBACK_TITLE;
  }
  return action.text;
}

function buildPlainActionTitle(action: FormattedToolAction, tool: ToolChatMessage): string {
  const fileActionTitle = buildFileActionTitle(action);
  if (fileActionTitle) {
    return fileActionTitle;
  }

  const toolName = normalizeToolName(tool.toolName);
  if ((toolName === 'web_search' || toolName === 'fetch_webpage') && action.actionKind === 'web') {
    return joinLabelAndTarget('网络搜索', action.actionText);
  }
  if (toolName === 'codex_cli' && action.actionKind === 'codex') {
    return joinLabelAndTarget('调用codex', action.actionText);
  }
  if (toolName === 'search_files' && action.actionKind === 'search') {
    return buildKeywordSearchTitle(action.actionText);
  }
  if (toolName === 'sfind' && isSkillToolAction(action)) {
    return joinLabelAndTarget(SKILL_ACTION_LABELS[action.actionKind], action.actionText);
  }
  if ((toolName === 'screen_action' || toolName === 'screen_control') && isScreenToolAction(action)) {
    return joinLabelAndTarget(SCREEN_ACTION_LABELS[action.actionKind], action.actionText);
  }
  return '';
}

function buildFileActionTitle(action: FormattedToolAction): string {
  if (!isFileToolAction(action)) {
    return '';
  }
  const label = FILE_ACTION_LABELS[action.actionKind];
  return joinLabelAndTarget(label, action.actionText);
}

function buildKeywordSearchTitle(rawTarget?: string): string {
  const target = rawTarget?.trim() || '';
  return target ? `搜索 ${target}（关键词）` : '搜索（关键词）';
}

function joinLabelAndTarget(label: string, rawTarget?: string): string {
  const target = rawTarget?.trim() || '';
  return target ? `${label} ${target}` : label;
}

function isFileToolAction(
  action: FormattedToolAction,
): action is FormattedToolAction & { actionKind: keyof typeof FILE_ACTION_LABELS } {
  return action.actionKind === 'read'
    || action.actionKind === 'write'
    || action.actionKind === 'edit';
}

function isSkillToolAction(
  action: FormattedToolAction,
): action is FormattedToolAction & { actionKind: keyof typeof SKILL_ACTION_LABELS } {
  return action.actionKind === 'load' || action.actionKind === 'unload';
}

function isScreenToolAction(
  action: FormattedToolAction,
): action is FormattedToolAction & { actionKind: keyof typeof SCREEN_ACTION_LABELS } {
  return action.actionKind === 'screen';
}
