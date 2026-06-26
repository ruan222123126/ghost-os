'use client';

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type Dispatch,
  type SetStateAction,
} from 'react';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ScreenControlComposerStep } from '@/lib/workflow-editor';
import {
  CLICK_COORDINATE_SOURCE_FIND_ICON,
  buildInitialEditorState,
  buildSavedClickStep,
  buildStepTag,
  type ClickEditorState,
} from '@/components/workflow/workflowClickStepEditorHelpers';
import { useWorkflowClickCaptureControls } from '@/hooks/workflow/useWorkflowClickCaptureControls';

interface UseWorkflowClickStepEditorOptions {
  canUseFindIconReference: boolean;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
  step: ScreenControlComposerStep;
  stepIndex: number;
}

interface UseWorkflowClickStepEditorResult {
  canUseFindIconReference: boolean;
  captureActive: boolean;
  captureLoading: boolean;
  closeAria: string;
  errorText: string;
  locale: ReturnType<typeof useWebLocale>['locale'];
  onSave: () => void;
  onStartCapture: () => void;
  onStateChange: Dispatch<SetStateAction<ClickEditorState>>;
  saving: boolean;
  state: ClickEditorState;
  stepTag: string;
  title: string;
  titleID: string;
}

interface ClickEditorFormState {
  captureActive: boolean;
  captureLoading: boolean;
  errorText: string;
  saving: boolean;
  setCaptureActive: Dispatch<SetStateAction<boolean>>;
  setCaptureLoading: Dispatch<SetStateAction<boolean>>;
  setErrorText: Dispatch<SetStateAction<string>>;
  setSaving: Dispatch<SetStateAction<boolean>>;
  setState: Dispatch<SetStateAction<ClickEditorState>>;
  state: ClickEditorState;
}

interface SaveClickActionOptions {
  canUseFindIconReference: boolean;
  fallbackError: string;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
  setErrorText: (value: string) => void;
  setSaving: (value: boolean) => void;
  state: ClickEditorState;
  step: ScreenControlComposerStep;
  stepIndex: number;
}

const CLICK_EDITOR_TITLE_ID = 'workflow-screen-click-editor-title';

export function useWorkflowClickStepEditor(
  options: UseWorkflowClickStepEditorOptions,
): UseWorkflowClickStepEditorResult {
  const { locale, copy } = useWebLocale();
  const initial = useMemo(() => buildInitialEditorState(options.step), [options.step]);
  const formState = useClickEditorFormState(initial);
  const startCapture = useWorkflowClickCaptureControls({
    captureActive: formState.captureActive,
    fallbackError: copy.system.genericRequestFailed,
    resetSignal: initial,
    setCaptureActive: formState.setCaptureActive,
    setCaptureLoading: formState.setCaptureLoading,
    setErrorText: formState.setErrorText,
    setState: formState.setState,
    state: formState.state,
  });
  const saveClick = useSaveClickAction({
    step: options.step,
    stepIndex: options.stepIndex,
    state: formState.state,
    canUseFindIconReference: options.canUseFindIconReference,
    onSave: options.onSave,
    setSaving: formState.setSaving,
    setErrorText: formState.setErrorText,
    fallbackError: copy.system.genericRequestFailed,
  });

  return {
    canUseFindIconReference: options.canUseFindIconReference,
    captureActive: formState.captureActive,
    captureLoading: formState.captureLoading,
    closeAria: copy.workflow.clickEditorCloseAria,
    errorText: formState.errorText,
    locale,
    onSave: saveClick,
    onStartCapture: startCapture,
    onStateChange: formState.setState,
    saving: formState.saving,
    state: formState.state,
    stepTag: buildStepTag(options.stepIndex),
    title: copy.workflow.clickEditorTitle,
    titleID: CLICK_EDITOR_TITLE_ID,
  };
}

function useClickEditorFormState(initial: ClickEditorState): ClickEditorFormState {
  const [state, setState] = useState<ClickEditorState>(initial);
  const [saving, setSaving] = useState(false);
  const [errorText, setErrorText] = useState('');
  const [captureActive, setCaptureActive] = useState(false);
  const [captureLoading, setCaptureLoading] = useState(false);

  useEffect(() => {
    setState(initial);
    setErrorText('');
    setCaptureActive(false);
    setCaptureLoading(false);
  }, [initial]);

  return {
    captureActive,
    captureLoading,
    errorText,
    saving,
    setCaptureActive,
    setCaptureLoading,
    setErrorText,
    setSaving,
    setState,
    state,
  };
}

function handleSaveClick(options: {
  canUseFindIconReference: boolean;
  fallbackError: string;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
  setErrorText: (value: string) => void;
  setSaving: (value: boolean) => void;
  state: ClickEditorState;
  step: ScreenControlComposerStep;
  stepIndex: number;
}): void {
  const { step, stepIndex, state, canUseFindIconReference, onSave, setSaving, setErrorText, fallbackError } = options;
  setSaving(true);
  setErrorText('');

  try {
    if (state.coordinateSource === CLICK_COORDINATE_SOURCE_FIND_ICON && !canUseFindIconReference) {
      throw new Error('find_icon reference is not available');
    }
    onSave(stepIndex, buildSavedClickStep(step, state));
  } catch (error) {
    setErrorText(toErrorMessage(error, fallbackError));
  } finally {
    setSaving(false);
  }
}

function useSaveClickAction(options: SaveClickActionOptions): () => void {
  return useCallback(() => {
    handleSaveClick(options);
  }, [options]);
}
