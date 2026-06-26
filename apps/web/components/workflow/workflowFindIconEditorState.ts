'use client';

import { useEffect, type Dispatch, type RefObject, type SetStateAction } from 'react';
import { buildFindIconTemplatePreviewURL } from '@/lib/workflow-editor/findIconPreview';
import type {
  FindIconEditorPanelState,
  FindIconEditorPreview,
  FindIconTestResult,
} from '@/components/workflow/WorkflowFindIconStepEditorModalView';

interface FindIconEditorResetOptions {
  initial: FindIconEditorPanelState;
  setState: Dispatch<SetStateAction<FindIconEditorPanelState>>;
  setErrorText: (value: string) => void;
  setPreview: Dispatch<SetStateAction<FindIconEditorPreview | null>>;
  setTesting: (value: boolean) => void;
  setTestResult: (value: FindIconTestResult) => void;
}

interface ClearFindIconTemplateOptions {
  fileInputRef: RefObject<HTMLInputElement>;
  setState: Dispatch<SetStateAction<FindIconEditorPanelState>>;
  setPreview: Dispatch<SetStateAction<FindIconEditorPreview | null>>;
  setErrorText: (value: string) => void;
}

export function useFindIconEditorResetState(options: FindIconEditorResetOptions) {
  const { initial, setState, setErrorText, setPreview, setTesting, setTestResult } = options;

  useEffect(() => {
    setState(initial);
    setErrorText('');
    setTesting(false);
    setTestResult('idle');
    setPreview((current) => {
      revokeFindIconPreviewURL(current);
      return buildInitialFindIconPreview(initial.templatePath);
    });
  }, [initial, setErrorText, setPreview, setState, setTesting, setTestResult]);
}

export function useFindIconEditorPreviewCleanup(preview: FindIconEditorPreview | null) {
  useEffect(() => () => revokeFindIconPreviewURL(preview), [preview]);
}

export function clearFindIconTemplateSelection(options: ClearFindIconTemplateOptions) {
  const { fileInputRef, setState, setPreview, setErrorText } = options;

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
  if (preview?.revocable && preview.url) {
    URL.revokeObjectURL(preview.url);
  }
}

function buildInitialFindIconPreview(templatePath: string): FindIconEditorPreview | null {
  const previewURL = buildFindIconTemplatePreviewURL(templatePath);
  if (!previewURL) {
    return null;
  }
  return {
    url: previewURL,
    revocable: false,
  };
}
