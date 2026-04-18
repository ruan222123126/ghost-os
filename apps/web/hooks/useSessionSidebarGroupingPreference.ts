'use client';

import { useCallback, useEffect, useState } from 'react';

export const SESSION_SIDEBAR_GROUPING_STORAGE_KEY = 'ghost.web.session.sidebar.grouping.enabled';

const GROUPING_CHANGE_EVENT = 'ghost:web:session-sidebar-grouping-change';
const ENABLED_VALUE = '1';
const DISABLED_VALUE = '0';

interface GroupingChangeDetail {
  enabled: boolean;
}

interface GroupingPreference {
  enabled: boolean;
  setEnabled: (enabled: boolean) => void;
}

export function useSessionSidebarGroupingPreference(): GroupingPreference {
  const [enabled, setEnabledState] = useState(true);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    setEnabledState(parseStoredValue(window.localStorage.getItem(SESSION_SIDEBAR_GROUPING_STORAGE_KEY)));
  }, []);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const onStorage = (event: StorageEvent) => {
      if (event.key !== SESSION_SIDEBAR_GROUPING_STORAGE_KEY) {
        return;
      }
      setEnabledState(parseStoredValue(event.newValue));
    };

    const onGroupingChange = (event: Event) => {
      const customEvent = event as CustomEvent<GroupingChangeDetail>;
      if (typeof customEvent.detail?.enabled !== 'boolean') {
        return;
      }
      setEnabledState(customEvent.detail.enabled);
    };

    window.addEventListener('storage', onStorage);
    window.addEventListener(GROUPING_CHANGE_EVENT, onGroupingChange);
    return () => {
      window.removeEventListener('storage', onStorage);
      window.removeEventListener(GROUPING_CHANGE_EVENT, onGroupingChange);
    };
  }, []);

  const setEnabled = useCallback((nextEnabled: boolean) => {
    setEnabledState(nextEnabled);
    if (typeof window === 'undefined') {
      return;
    }

    const value = nextEnabled ? ENABLED_VALUE : DISABLED_VALUE;
    window.localStorage.setItem(SESSION_SIDEBAR_GROUPING_STORAGE_KEY, value);
    window.dispatchEvent(new CustomEvent<GroupingChangeDetail>(GROUPING_CHANGE_EVENT, {
      detail: { enabled: nextEnabled },
    }));
  }, []);

  return {
    enabled,
    setEnabled,
  };
}

function parseStoredValue(raw: string | null): boolean {
  if (raw === null) {
    return true;
  }
  if (raw === ENABLED_VALUE || raw.toLowerCase() === 'true') {
    return true;
  }
  if (raw === DISABLED_VALUE || raw.toLowerCase() === 'false') {
    return false;
  }
  return true;
}
