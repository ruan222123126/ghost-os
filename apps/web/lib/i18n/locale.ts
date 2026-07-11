export type WebLocale = 'zh-CN' | 'en-US';

export const WEB_LOCALE_STORAGE_KEY = 'ghost.web.locale';

const ZH_CN: WebLocale = 'zh-CN';
const EN_US: WebLocale = 'en-US';
const ZH_PREFIX = 'zh';

export function isWebLocale(value: string): value is WebLocale {
  return value === ZH_CN || value === EN_US;
}

export function resolveStoredLocale(value: string | null | undefined): WebLocale | undefined {
  if (!value) {
    return undefined;
  }
  return isWebLocale(value) ? value : undefined;
}

export function resolveBrowserLocale(browserLocale: string | undefined): WebLocale {
  const normalized = (browserLocale ?? '').trim().toLowerCase();
  if (normalized.startsWith(ZH_PREFIX)) {
    return ZH_CN;
  }
  return EN_US;
}

export function resolveInitialLocale(input: {
  storedLocale?: string | null;
  browserLocale?: string | undefined;
}): WebLocale {
  const stored = resolveStoredLocale(input.storedLocale);
  if (stored) {
    return stored;
  }
  return resolveBrowserLocale(input.browserLocale);
}

export function htmlLangOfLocale(locale: WebLocale): string {
  return locale === ZH_CN ? 'zh-CN' : 'en-US';
}
