'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { getMousePosition } from '@/lib/api/tools/mousePosition';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ScreenControlComposerStep } from '@/lib/workflow-editor';
import { WorkflowClickStepEditorModalView } from '@/components/workflow/WorkflowClickStepEditorModalView';
import {
  CLICK_COORDINATE_SOURCE_FIND_ICON,
  CLICK_MOUSE_POLL_MS,
  applyMousePositionToState,
  buildInitialEditorState,
  buildSavedClickStep,
  buildStepTag,
  type ClickEditorState,
} from '@/components/workflow/workflowClickStepEditorHelpers';

interface WorkflowClickStepEditorModalProps {
  open: boolean;
  stepIndex: number;
  step: ScreenControlComposerStep;
  canUseFindIconReference: boolean;
  onClose: () => void;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
}

export function WorkflowClickStepEditorModal(
  props: WorkflowClickStepEditorModalProps,
) {
  const { open } = props;
  const [mounted, setMounted] = useState(false);

  useEffect(() => setMounted(true), []);
  if (!mounted || !open) {
    return null;
  }
  return createPortal(
    <WorkflowClickStepEditorModalContent {...props} />,
    document.body,
  );
}

function WorkflowClickStepEditorModalContent(
  props: WorkflowClickStepEditorModalProps,
) {
  const { locale, copy } = useWebLocale();
  const { step, stepIndex, canUseFindIconReference, onClose, onSave } = props;
  const initial = useMemo(() => buildInitialEditorState(step), [step]);
  const [state, setState] = useState<ClickEditorState>(initial);
  const [saving, setSaving] = useState(false);
  const [errorText, setErrorText] = useState('');
  const [captureActive, setCaptureActive] = useState(false);
  const [captureLoading, setCaptureLoading] = useState(false);
  const captureSnapshotRef = useRef<ClickEditorState | null>(null);
  const capturePendingRef = useRef(false);
  const titleID = 'workflow-screen-click-editor-title';

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

  useEffect(() => {
    if (!captureActive) {
      return;
    }
    void pollMousePosition();
    const timer = window.setInterval(() => {
      void pollMousePosition();
    }, CLICK_MOUSE_POLL_MS);
    return () => window.clearInterval(timer);
  }, [captureActive, pollMousePosition]);

  useEffect(() => {
    if (!captureActive) {
      return;
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

  const startCapture = useCallback(() => {
    captureSnapshotRef.current = state;
    setErrorText('');
    setCaptureActive(true);
  }, [state]);

  const saveClick = () =>
    handleSaveClick({
      step,
      stepIndex,
      state,
      canUseFindIconReference,
      onSave,
      setSaving,
      setErrorText,
      fallbackError: copy.system.genericRequestFailed,
    });

  return (
    <WorkflowClickStepEditorModalView
      locale={locale}
      closeAria={copy.workflow.clickEditorCloseAria}
      titleID={titleID}
      title={copy.workflow.clickEditorTitle}
      stepTag={buildStepTag(stepIndex)}
      errorText={errorText}
      state={state}
      canUseFindIconReference={canUseFindIconReference}
      saving={saving}
      captureActive={captureActive}
      captureLoading={captureLoading}
      onClose={onClose}
      onSave={saveClick}
      onStartCapture={startCapture}
      onStateChange={setState}
    />
  );
}

function handleSaveClick(options: {
  step: ScreenControlComposerStep;
  stepIndex: number;
  state: ClickEditorState;
  canUseFindIconReference: boolean;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
  setSaving: (value: boolean) => void;
  setErrorText: (value: string) => void;
  fallbackError: string;
}) {
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
