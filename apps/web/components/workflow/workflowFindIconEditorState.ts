'use client';

import { useEffect, type Dispatch, type RefObject, type SetStateAction } from 'react';
import type {
  FindIconEditorPanelState,
  FindIconEditorPreview,
  FindIconTestResult,
} from '@/components/workflow/WorkflowFindIconStepEditorModalView';

export function useFindIconEditorResetState(
  initial: FindIconEditorPanelState,
  setState: Dispatch<SetStateAction<FindIconEditorPanelState>>,
  setErrorText: (value: string) => void,
  setPreview: Dispatch<SetStateAction<FindIconEditorPreview | null>>,
  setTesting: (value: boolean) => void,
  setTestResult: (value: FindIconTestResult) => void,
) {
  useEffect(() => {
    setState(initial);
    setErrorText('');
    setTesting(false);
    setTestResult('idle');
    setPreview((current) => {
      revokeFindIconPreviewURL(current);
      return null;
    });
  }, [initial, setErrorText, setPreview, setState, setTesting, setTestResult]);
}

export function useFindIconEditorPreviewCleanup(preview: FindIconEditorPreview | null) {
  useEffect(() => () => revokeFindIconPreviewURL(preview), [preview]);
}

export function clearFindIconTemplateSelection(
  fileInputRef: RefObject<HTMLInputElement>,
  setState: Dispatch<SetStateAction<FindIconEditorPanelState>>,
  setPreview: Dispatch<SetStateAction<FindIconEditorPreview | null>>,
  setErrorText: (value: string) => void,
) {
  setState((current) => ({ ...current, templatePath: '', templateName: '' }));
  setErrorText('');
  setPreview((current) => {
    revokeFindIconPreviewURL(current);
    return null;
  });
  if (fileInputRef.current) {
    fileInputRef.current.value = '';
  }
}

export function revokeFindIconPreviewURL(preview: FindIconEditorPreview | null) {
  if (preview?.objectURL) {
    URL.revokeObjectURL(preview.objectURL);
  }
}
