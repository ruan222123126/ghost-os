'use client';

import {
  useCallback,
  useEffect,
  useRef,
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
} from 'react';
import { getMousePosition } from '@/lib/api/tools/mousePosition';
import { toErrorMessage } from '@/lib/errors';
import {
  CLICK_MOUSE_POLL_MS,
  applyMousePositionToState,
  type ClickEditorState,
} from '@/components/workflow/workflowClickStepEditorHelpers';

export interface WorkflowClickCaptureControlsOptions {
  captureActive: boolean;
  fallbackError: string;
  resetSignal: ClickEditorState;
  setCaptureActive: Dispatch<SetStateAction<boolean>>;
  setCaptureLoading: Dispatch<SetStateAction<boolean>>;
  setErrorText: Dispatch<SetStateAction<string>>;
  setState: Dispatch<SetStateAction<ClickEditorState>>;
  state: ClickEditorState;
}

type CaptureSnapshotRef = MutableRefObject<ClickEditorState | null>;
type CapturePendingRef = MutableRefObject<boolean>;

interface CaptureControlRefs {
  pendingRef: CapturePendingRef;
  snapshotRef: CaptureSnapshotRef;
}

interface ClickCaptureSessionControls {
  cancelCapture: () => void;
  pollMousePosition: () => Promise<void>;
  stopCapture: () => void;
}

export function useWorkflowClickCaptureControls(
  options: WorkflowClickCaptureControlsOptions,
): () => void {
  const refs = useCaptureControlRefs(options.resetSignal);
  const controls = useClickCaptureSessionControls(options, refs);
  useClickCaptureEffects(options.captureActive, controls);

  return useStartClickCapture({
    setCaptureActive: options.setCaptureActive,
    setErrorText: options.setErrorText,
    snapshotRef: refs.snapshotRef,
    state: options.state,
  });
}

function useClickCaptureSessionControls(
  options: WorkflowClickCaptureControlsOptions,
  refs: CaptureControlRefs,
): ClickCaptureSessionControls {
  const {
    fallbackError,
    setCaptureActive,
    setCaptureLoading,
    setErrorText,
    setState,
  } = options;
  const stopCapture = useStopClickCapture({ refs, setCaptureActive, setCaptureLoading });
  const restoreCaptureSnapshot = useRestoreCaptureSnapshot(refs.snapshotRef, setState);
  const cancelCapture = useCancelClickCapture(restoreCaptureSnapshot, stopCapture);
  const failCapture = useFailClickCapture({
    fallbackError,
    restoreCaptureSnapshot,
    setErrorText,
    stopCapture,
  });
  const pollMousePosition = usePollMousePosition({
    failCapture,
    refs,
    setCaptureLoading,
    setErrorText,
    setState,
  });

  return { cancelCapture, pollMousePosition, stopCapture };
}

function useClickCaptureEffects(
  captureActive: boolean,
  controls: ClickCaptureSessionControls,
): void {
  useCapturePolling(captureActive, controls.pollMousePosition);
  useCaptureKeyboard(captureActive, controls.stopCapture, controls.cancelCapture);
}

function useCaptureControlRefs(resetSignal: ClickEditorState): CaptureControlRefs {
  const snapshotRef = useRef<ClickEditorState | null>(null);
  const pendingRef = useRef(false);

  useEffect(() => {
    snapshotRef.current = null;
  }, [resetSignal]);

  return { pendingRef, snapshotRef };
}

function useStopClickCapture(options: {
  refs: CaptureControlRefs;
  setCaptureActive: Dispatch<SetStateAction<boolean>>;
  setCaptureLoading: Dispatch<SetStateAction<boolean>>;
}): () => void {
  const { refs, setCaptureActive, setCaptureLoading } = options;
  return useCallback(() => {
    setCaptureActive(false);
    setCaptureLoading(false);
    refs.pendingRef.current = false;
  }, [refs.pendingRef, setCaptureActive, setCaptureLoading]);
}

function useRestoreCaptureSnapshot(
  snapshotRef: CaptureSnapshotRef,
  setState: Dispatch<SetStateAction<ClickEditorState>>,
): () => void {
  return useCallback(() => {
    const snapshot = snapshotRef.current;
    if (snapshot) {
      setState(snapshot);
    }
  }, [setState, snapshotRef]);
}

function useCancelClickCapture(
  restoreCaptureSnapshot: () => void,
  stopCapture: () => void,
): () => void {
  return useCallback(() => {
    restoreCaptureSnapshot();
    stopCapture();
  }, [restoreCaptureSnapshot, stopCapture]);
}

function useFailClickCapture(options: {
  fallbackError: string;
  restoreCaptureSnapshot: () => void;
  setErrorText: Dispatch<SetStateAction<string>>;
  stopCapture: () => void;
}): (error: unknown) => void {
  const { fallbackError, restoreCaptureSnapshot, setErrorText, stopCapture } = options;
  return useCallback((error) => {
    restoreCaptureSnapshot();
    stopCapture();
    setErrorText(toErrorMessage(error, fallbackError));
  }, [fallbackError, restoreCaptureSnapshot, setErrorText, stopCapture]);
}

function usePollMousePosition(options: {
  failCapture: (error: unknown) => void;
  refs: CaptureControlRefs;
  setCaptureLoading: Dispatch<SetStateAction<boolean>>;
  setErrorText: Dispatch<SetStateAction<string>>;
  setState: Dispatch<SetStateAction<ClickEditorState>>;
}): () => Promise<void> {
  const { failCapture, refs, setCaptureLoading, setErrorText, setState } = options;
  return useCallback(async () => {
    if (refs.pendingRef.current) {
      return;
    }
    refs.pendingRef.current = true;
    setCaptureLoading(true);
    try {
      const position = await getMousePosition();
      setState((current) => applyMousePositionToState(current, position));
      setErrorText('');
    } catch (error) {
      failCapture(error);
      return;
    } finally {
      refs.pendingRef.current = false;
      setCaptureLoading(false);
    }
  }, [failCapture, refs.pendingRef, setCaptureLoading, setErrorText, setState]);
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

function useStartClickCapture(options: {
  setCaptureActive: Dispatch<SetStateAction<boolean>>;
  setErrorText: Dispatch<SetStateAction<string>>;
  snapshotRef: CaptureSnapshotRef;
  state: ClickEditorState;
}): () => void {
  const { setCaptureActive, setErrorText, snapshotRef, state } = options;
  return useCallback(() => {
    snapshotRef.current = state;
    setErrorText('');
    setCaptureActive(true);
  }, [setCaptureActive, setErrorText, snapshotRef, state]);
}
