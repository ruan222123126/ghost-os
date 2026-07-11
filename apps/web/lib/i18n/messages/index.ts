import type { WebLocale } from '@/lib/i18n/locale';
import { copyForChat } from '@/lib/i18n/messages/chat';
import { copyForSettings } from '@/lib/i18n/messages/settings';
import { copyForSystem } from '@/lib/i18n/messages/system';
import { copyForWorkflow } from '@/lib/i18n/messages/workflow';

const copyCatalog = {
  'en-US': {
    chat: copyForChat('en-US'),
    settings: copyForSettings('en-US'),
    system: copyForSystem('en-US'),
    workflow: copyForWorkflow('en-US'),
  },
  'zh-CN': {
    chat: copyForChat('zh-CN'),
    settings: copyForSettings('zh-CN'),
    system: copyForSystem('zh-CN'),
    workflow: copyForWorkflow('zh-CN'),
  },
} as const;

export type WebCopy = (typeof copyCatalog)['en-US'];

export function copyForLocale(locale: WebLocale): WebCopy {
  return copyCatalog[locale];
}
