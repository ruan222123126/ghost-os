'use client';

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type Dispatch,
  type SetStateAction,
} from 'react';
import { getMousePosition } from '@/lib/api/tools/mousePosition';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ScreenControlComposerStep } from '@/lib/workflow-editor';
import {
  CLICK_COORDINATE_SOURCE_FIND_ICON,
  CLICK_MOUSE_POLL_MS,
  applyMousePositionToState,
  buildInitialEditorState,
  buildSavedClickStep,
  buildStepTag,
  type ClickEditorState,
} from '@/components/workflow/workflowClickStepEditorHelpers';

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

const CLICK_EDITOR_TITLE_ID = 'workflow-screen-click-editor-title';

export function useWorkflowClickStepEditor(
  options: UseWorkflowClickStepEditorOptions,
): UseWorkflowClickStepEditorResult {
  const { locale, copy } = useWebLocale();
  const initial = useMemo(() => buildInitialEditorState(options.step), [options.step]);
  const [state, setState] = useState<ClickEditorState>(initial);
  const [saving, setSaving] = useState(false);
  const [errorText, setErrorText] = useState('');
  const [captureActive, setCaptureActive] = useState(false);
  const [captureLoading, setCaptureLoading] = useState(false);
  const captureSnapshotRef = useRef<ClickEditorState | null>(null);
  const capturePendingRef = useRef(false);

  useEffect(() => {
    setState(initial);
    setErrorText('');
    setCaptureActive(false);
    setCaptureLoading(false);
    captureSnapshotRef.current = null;
  }, [initial]);

  const stopCapture = useCallback(() => {
    setCaptureActive(false);
    setCaptureLoading(false);
    capturePendingRef.current = false;
  }, []);

  const restoreCaptureSnapshot = useCallback(() => {
    const snapshot = captureSnapshotRef.current;
    if (snapshot) {
      setState(snapshot);
    }
  }, []);

  const cancelCapture = useCallback(() => {
    restoreCaptureSnapshot();
    stopCapture();
  }, [restoreCaptureSnapshot, stopCapture]);

  const failCapture = useCallback((error: unknown) => {
    restoreCaptureSnapshot();
    stopCapture();
    setErrorText(toErrorMessage(error, copy.system.genericRequestFailed));
  }, [copy.system.genericRequestFailed, restoreCaptureSnapshot, stopCapture]);

  const pollMousePosition = useCallback(async () => {
    if (capturePendingRef.current) {
      return;
    }
    capturePendingRef.current = true;
    setCaptureLoading(true);
    try {
      const position = await getMousePosition();
      setState((current) => applyMousePositionToState(current, position));
      setErrorText('');
    } catch (error) {
      failCapture(error);
      return;
    } finally {
      capturePendingRef.current = false;
      setCaptureLoading(false);
    }
  }, [failCapture]);

  useCapturePolling(captureActive, pollMousePosition);
  useCaptureKeyboard(captureActive, stopCapture, cancelCapture);

  const startCapture = useCallback(() => {
    captureSnapshotRef.current = state;
    setErrorText('');
    setCaptureActive(true);
  }, [state]);

  const saveClick = useCallback(() => {
    handleSaveClick({
      step: options.step,
      stepIndex: options.stepIndex,
      state,
      canUseFindIconReference: options.canUseFindIconReference,
      onSave: options.onSave,
      setSaving,
      setErrorText,
      fallbackError: copy.system.genericRequestFailed,
    });
  }, [copy.system.genericRequestFailed, options, state]);

  return {
    canUseFindIconReference: options.canUseFindIconReference,
    captureActive,
    captureLoading,
    closeAria: copy.workflow.clickEditorCloseAria,
    errorText,
    locale,
    onSave: saveClick,
    onStartCapture: startCapture,
    onStateChange: setState,
    saving,
    state,
    stepTag: buildStepTag(options.stepIndex),
    title: copy.workflow.clickEditorTitle,
    titleID: CLICK_EDITOR_TITLE_ID,
  };
}

function useCapturePolling(
  captureActive: boolean,
  pollMousePosition: () => Promise<void>,
): void {
  useEffect(() => {
    if (!captureActive) {
      return undefined;
    }
    void pollMousePosition();
    const timer = window.setInterval(() => {
      void pollMousePosition();
    }, CLICK_MOUSE_POLL_MS);
    return () => window.clearInterval(timer);
  }, [captureActive, pollMousePosition]);
}

function useCaptureKeyboard(
  captureActive: boolean,
  stopCapture: () => void,
  cancelCapture: () => void,
): void {
  useEffect(() => {
    if (!captureActive) {
      return undefined;
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Enter') {
        event.preventDefault();
        stopCapture();
      }
      if (event.key === 'Escape') {
        event.preventDefault();
        cancelCapture();
      }
    };
    window.addEventListener('keydown', handleKeyDown, true);
    return () => window.removeEventListener('keydown', handleKeyDown, true);
  }, [cancelCapture, captureActive, stopCapture]);
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

