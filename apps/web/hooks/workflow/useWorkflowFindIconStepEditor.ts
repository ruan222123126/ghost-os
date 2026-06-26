'use client';

import {
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type Dispatch,
  type RefObject,
  type SetStateAction,
} from 'react';
import { previewFindIcon, uploadFindIconTemplate } from '@/lib/api/tools/findIcon';
import { createChatImageDrafts } from '@/lib/chatImageDrafts';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import {
  normalizeFindIconComposerParams,
  withFindIconComposerParams,
  type ScreenControlComposerStep,
} from '@/lib/workflow-editor';
import type {
  FindIconEditorPanelState,
  FindIconEditorPreview,
  FindIconTestResult,
} from '@/components/workflow/WorkflowFindIconStepEditorModalView';
import {
  buildFindIconInitialEditorState,
  buildFindIconStepTag,
  mergeFindIconHoverAction,
} from '@/components/workflow/workflowFindIconEditorHelpers';
import {
  clearFindIconTemplateSelection,
  revokeFindIconPreviewURL,
  useFindIconEditorPreviewCleanup,
  useFindIconEditorResetState,
} from '@/components/workflow/workflowFindIconEditorState';

interface UseWorkflowFindIconStepEditorOptions {
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
  step: ScreenControlComposerStep;
  stepIndex: number;
}

interface UseWorkflowFindIconStepEditorResult {
  closeAria: string;
  errorText: string;
  fileInputRef: RefObject<HTMLInputElement>;
  onPickAction: (hover: boolean) => void;
  onRemoveImage: () => void;
  onSave: () => void;
  onTest: () => Promise<void>;
  onUpload: (event: ChangeEvent<HTMLInputElement>) => void;
  preview: FindIconEditorPreview | null;
  saving: boolean;
  state: FindIconEditorPanelState;
  stepTag: string;
  testing: boolean;
  testResult: FindIconTestResult;
  titleID: string;
  uploading: boolean;
}

interface FindIconEditorHandlerOptions {
  fallbackError: string;
  fileInputRef: RefObject<HTMLInputElement>;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
  setErrorText: (value: string) => void;
  setPreview: Dispatch<SetStateAction<FindIconEditorPreview | null>>;
  setSaving: (value: boolean) => void;
  setState: Dispatch<SetStateAction<FindIconEditorPanelState>>;
  setTesting: (value: boolean) => void;
  setTestResult: (value: FindIconTestResult) => void;
  setUploading: (value: boolean) => void;
  state: FindIconEditorPanelState;
  step: ScreenControlComposerStep;
  stepIndex: number;
}

const FIND_ICON_PREVIEW_THRESHOLD = 0.96;
const FIND_ICON_EDITOR_TITLE_ID = 'workflow-screen-find-icon-editor-title';

export function useWorkflowFindIconStepEditor(
  options: UseWorkflowFindIconStepEditorOptions,
): UseWorkflowFindIconStepEditorResult {
  const { copy } = useWebLocale();
  const initial = useMemo(() => buildFindIconInitialEditorState(options.step), [options.step]);
  const [state, setState] = useState<FindIconEditorPanelState>(initial);
  const [preview, setPreview] = useState<FindIconEditorPreview | null>(null);
  const [uploading, setUploading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<FindIconTestResult>('idle');
  const [errorText, setErrorText] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);
  const resetActions = { setState, setErrorText, setPreview, setTesting, setTestResult };

  useFindIconEditorResetState({ initial, ...resetActions });
  useFindIconEditorPreviewCleanup(preview);

  const handlers = useFindIconEditorHandlers({
    step: options.step,
    stepIndex: options.stepIndex,
    state,
    onSave: options.onSave,
    fallbackError: copy.system.genericRequestFailed,
    fileInputRef,
    setState,
    setPreview,
    setUploading,
    setSaving,
    setTesting,
    setTestResult,
    setErrorText,
  });

  return {
    closeAria: copy.workflow.findIconEditorCloseAria,
    errorText,
    fileInputRef,
    preview,
    saving,
    state,
    stepTag: buildFindIconStepTag(options.stepIndex),
    testing,
    testResult,
    titleID: FIND_ICON_EDITOR_TITLE_ID,
    uploading,
    ...handlers,
  };
}

function useFindIconEditorHandlers(options: FindIconEditorHandlerOptions) {
  const {
    step,
    stepIndex,
    state,
    onSave,
    fallbackError,
    fileInputRef,
    setState,
    setPreview,
    setUploading,
    setSaving,
    setTesting,
    setTestResult,
    setErrorText,
  } = options;
  return {
    onPickAction: (hover: boolean) => setState((current) => ({ ...current, hoverAfterMatch: hover })),
    onSave: () => handleSaveFindIcon({ step, stepIndex, state, onSave, setSaving, setErrorText, fallbackError }),
    onUpload: (event: ChangeEvent<HTMLInputElement>) => {
      setTesting(false);
      setTestResult('idle');
      void handleUploadTemplateFile({
        files: event.target.files,
        setState,
        setPreview,
        setUploading,
        setErrorText,
        fallbackError,
      });
    },
    onRemoveImage: () => {
      clearFindIconTemplateSelection({ fileInputRef, setState, setPreview, setErrorText });
      setTesting(false);
      setTestResult('idle');
    },
    onTest: () =>
      handleTestFindIcon({
        templatePath: state.templatePath,
        hoverAfterMatch: state.hoverAfterMatch,
        setTesting,
        setTestResult,
        setErrorText,
        fallbackError,
      }),
  };
}

async function handleUploadTemplateFile(options: {
  fallbackError: string;
  files: FileList | null;
  setErrorText: (value: string) => void;
  setPreview: Dispatch<SetStateAction<FindIconEditorPreview | null>>;
  setState: Dispatch<SetStateAction<FindIconEditorPanelState>>;
  setUploading: (value: boolean) => void;
}): Promise<void> {
  const { files, setState, setPreview, setUploading, setErrorText, fallbackError } = options;
  const file = files?.[0];
  if (!file) {
    return;
  }
  setUploading(true);
  setErrorText('');
  try {
    const drafts = await createChatImageDrafts([file]);
    const draft = drafts[0];
    if (!draft?.content.url) {
      throw new Error('template image url is missing');
    }
    const uploaded = await uploadFindIconTemplate({
      filename: file.name,
      mime_type: file.type,
      data_url: draft.content.url,
    });
    setState((current) => ({
      ...current,
      templatePath: uploaded.template_path,
      templateName: uploaded.template_name,
    }));
    const objectURL = URL.createObjectURL(file);
    setPreview((current) => {
      revokeFindIconPreviewURL(current);
      return { url: objectURL, revocable: true };
    });
  } catch (error) {
    setErrorText(toErrorMessage(error, fallbackError));
  } finally {
    setUploading(false);
  }
}

async function handleTestFindIcon(options: {
  fallbackError: string;
  hoverAfterMatch: boolean;
  setErrorText: (value: string) => void;
  setTesting: (value: boolean) => void;
  setTestResult: (value: FindIconTestResult) => void;
  templatePath: string;
}): Promise<void> {
  const { templatePath, hoverAfterMatch, setTesting, setTestResult, setErrorText, fallbackError } = options;
  setTesting(true);
  setErrorText('');
  try {
    const path = templatePath.trim();
    if (!path) {
      throw new Error('template_path is required');
    }
    const result = await previewFindIcon({
      template_path: path,
      max_results: 1,
      threshold: FIND_ICON_PREVIEW_THRESHOLD,
      hover_after_match: hoverAfterMatch,
    });
    setTestResult(result.exists ? 'success' : 'failure');
  } catch (error) {
    setTestResult('failure');
    setErrorText(toErrorMessage(error, fallbackError));
  } finally {
    setTesting(false);
  }
}

async function handleSaveFindIcon(options: {
  fallbackError: string;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
  setErrorText: (value: string) => void;
  setSaving: (value: boolean) => void;
  state: FindIconEditorPanelState;
  step: ScreenControlComposerStep;
  stepIndex: number;
}): Promise<void> {
  const { step, stepIndex, state, onSave, setSaving, setErrorText, fallbackError } = options;
  setSaving(true);
  setErrorText('');
  try {
    const templatePath = state.templatePath.trim();
    if (!templatePath) {
      throw new Error('template_path is required');
    }
    const current = normalizeFindIconComposerParams(step.params);
    const baseStep = withFindIconComposerParams(step, {
      template_path: templatePath,
      template_name: state.templateName.trim() || undefined,
      threshold: current?.threshold,
      max_results: current?.max_results,
    });
    onSave(stepIndex, mergeFindIconHoverAction(baseStep, state.hoverAfterMatch));
  } catch (error) {
    setErrorText(toErrorMessage(error, fallbackError));
  } finally {
    setSaving(false);
  }
}
