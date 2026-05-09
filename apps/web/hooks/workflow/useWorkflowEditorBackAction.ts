'use client';

import { useCallback } from 'react';
import { buildHomeSettingsURL } from '@/lib/settingsQuery';

interface WorkflowEditorRouter {
  push(path: string): void;
}

export function useWorkflowEditorBackAction(router: WorkflowEditorRouter) {
  return useCallback(() => {
    router.push(buildHomeSettingsURL('tasks'));
  }, [router]);
}
