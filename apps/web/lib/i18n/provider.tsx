'use client';

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import {
  htmlLangOfLocale,
  resolveInitialLocale,
  WEB_LOCALE_STORAGE_KEY,
  type WebLocale,
} from '@/lib/i18n/locale';
import { copyForLocale, type WebCopy } from '@/lib/i18n/messages';

interface WebLocaleContextValue {
  locale: WebLocale;
  setLocale: (locale: WebLocale) => void;
  copy: WebCopy;
}

const WebLocaleContext = createContext<WebLocaleContextValue | null>(null);
const DEFAULT_LOCALE: WebLocale = 'en-US';

interface WebLocaleProviderProps {
  children: ReactNode;
  initialLocale?: WebLocale;
}

function resolveLocaleFromWindow(): WebLocale {
  if (typeof window === 'undefined') {
    return DEFAULT_LOCALE;
  }
  try {
    return resolveInitialLocale({
      storedLocale: window.localStorage.getItem(WEB_LOCALE_STORAGE_KEY),
      browserLocale: window.navigator.language,
    });
  } catch {
    return DEFAULT_LOCALE;
  }
}

function isLocaleDifferent(current: WebLocale, next: WebLocale): boolean {
  return current !== next;
}

export function WebLocaleProvider(props: WebLocaleProviderProps) {
  const { children, initialLocale = DEFAULT_LOCALE } = props;
  const [locale, setLocaleState] = useState<WebLocale>(initialLocale);

  useEffect(() => {
    const nextLocale = resolveLocaleFromWindow();
    setLocaleState((current) => isLocaleDifferent(current, nextLocale) ? nextLocale : current);
  }, []);

  useEffect(() => {
    if (typeof document === 'undefined') {
      return;
    }
    document.documentElement.lang = htmlLangOfLocale(locale);
  }, [locale]);

  const setLocale = useCallback((nextLocale: WebLocale) => {
    setLocaleState(nextLocale);
    if (typeof window === 'undefined') {
      return;
    }
    window.localStorage.setItem(WEB_LOCALE_STORAGE_KEY, nextLocale);
  }, []);

  const copy = useMemo(() => copyForLocale(locale), [locale]);

  const value = useMemo<WebLocaleContextValue>(
    () => ({ locale, setLocale, copy }),
    [copy, locale, setLocale],
  );

  return (
    <WebLocaleContext.Provider value={value}>
      {children}
    </WebLocaleContext.Provider>
  );
}

export function useWebLocale(): WebLocaleContextValue {
  const context = useContext(WebLocaleContext);
  if (!context) {
    throw new Error('useWebLocale must be used within WebLocaleProvider');
  }

  return context;
}
