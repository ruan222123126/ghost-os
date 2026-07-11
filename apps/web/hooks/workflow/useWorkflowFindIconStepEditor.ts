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
import {
  previewFindIcon,
  uploadFindIconTemplate,
  type FindIconTemplateUploadResponse,
} from '@/lib/api/tools/findIcon';
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

interface FindIconEditorFormState {
  errorText: string;
  fileInputRef: RefObject<HTMLInputElement>;
  preview: FindIconEditorPreview | null;
  saving: boolean;
  setErrorText: Dispatch<SetStateAction<string>>;
  setPreview: Dispatch<SetStateAction<FindIconEditorPreview | null>>;
  setSaving: Dispatch<SetStateAction<boolean>>;
  setState: Dispatch<SetStateAction<FindIconEditorPanelState>>;
  setTesting: Dispatch<SetStateAction<boolean>>;
  setTestResult: Dispatch<SetStateAction<FindIconTestResult>>;
  setUploading: Dispatch<SetStateAction<boolean>>;
  state: FindIconEditorPanelState;
  testing: boolean;
  testResult: FindIconTestResult;
  uploading: boolean;
}

interface FindIconEditorHandlers {
  onPickAction: (hover: boolean) => void;
  onRemoveImage: () => void;
  onSave: () => void;
  onTest: () => Promise<void>;
  onUpload: (event: ChangeEvent<HTMLInputElement>) => void;
}

interface FindIconEditorHandlerOptions {
  fallbackError: string;
  fileInputRef: RefObject<HTMLInputElement>;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
  setErrorText: Dispatch<SetStateAction<string>>;
  setPreview: Dispatch<SetStateAction<FindIconEditorPreview | null>>;
  setSaving: Dispatch<SetStateAction<boolean>>;
  setState: Dispatch<SetStateAction<FindIconEditorPanelState>>;
  setTesting: Dispatch<SetStateAction<boolean>>;
  setTestResult: Dispatch<SetStateAction<FindIconTestResult>>;
  setUploading: Dispatch<SetStateAction<boolean>>;
  state: FindIconEditorPanelState;
  step: ScreenControlComposerStep;
  stepIndex: number;
}

interface UploadTemplateFileOptions {
  fallbackError: string;
  files: FileList | null;
  setErrorText: Dispatch<SetStateAction<string>>;
  setPreview: Dispatch<SetStateAction<FindIconEditorPreview | null>>;
  setState: Dispatch<SetStateAction<FindIconEditorPanelState>>;
  setUploading: Dispatch<SetStateAction<boolean>>;
}

const FIND_ICON_PREVIEW_THRESHOLD = 0.96;
const FIND_ICON_EDITOR_TITLE_ID = 'workflow-screen-find-icon-editor-title';

export function useWorkflowFindIconStepEditor(
  options: UseWorkflowFindIconStepEditorOptions,
): UseWorkflowFindIconStepEditorResult {
  const { copy } = useWebLocale();
  const initial = useMemo(() => buildFindIconInitialEditorState(options.step), [options.step]);
  const formState = useFindIconEditorFormState(initial);
  const handlers = useFindIconEditorHandlers(
    createFindIconEditorHandlerOptions({
      formState,
      fallbackError: copy.system.genericRequestFailed,
      onSave: options.onSave,
      step: options.step,
      stepIndex: options.stepIndex,
    }),
  );

  return buildFindIconEditorResult({
    closeAria: copy.workflow.findIconEditorCloseAria,
    formState,
    handlers,
    stepIndex: options.stepIndex,
  });
}

function createFindIconEditorHandlerOptions(options: {
  fallbackError: string;
  formState: FindIconEditorFormState;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
  step: ScreenControlComposerStep;
  stepIndex: number;
}): FindIconEditorHandlerOptions {
  const { fallbackError, formState, onSave, step, stepIndex } = options;
  return {
    fallbackError,
    fileInputRef: formState.fileInputRef,
    onSave,
    setErrorText: formState.setErrorText,
    setPreview: formState.setPreview,
    setSaving: formState.setSaving,
    setState: formState.setState,
    setTesting: formState.setTesting,
    setTestResult: formState.setTestResult,
    setUploading: formState.setUploading,
    state: formState.state,
    step,
    stepIndex,
  };
}

function buildFindIconEditorResult(options: {
  closeAria: string;
  formState: FindIconEditorFormState;
  handlers: FindIconEditorHandlers;
  stepIndex: number;
}): UseWorkflowFindIconStepEditorResult {
  const { closeAria, formState, handlers, stepIndex } = options;
  return {
    closeAria,
    errorText: formState.errorText,
    fileInputRef: formState.fileInputRef,
    onPickAction: handlers.onPickAction,
    onRemoveImage: handlers.onRemoveImage,
    onSave: handlers.onSave,
    onTest: handlers.onTest,
    onUpload: handlers.onUpload,
    preview: formState.preview,
    saving: formState.saving,
    state: formState.state,
    stepTag: buildFindIconStepTag(stepIndex),
    testing: formState.testing,
    testResult: formState.testResult,
    titleID: FIND_ICON_EDITOR_TITLE_ID,
    uploading: formState.uploading,
  };
}

function useFindIconEditorFormState(initial: FindIconEditorPanelState): FindIconEditorFormState {
  const [state, setState] = useState<FindIconEditorPanelState>(initial);
  const [preview, setPreview] = useState<FindIconEditorPreview | null>(null);
  const [uploading, setUploading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<FindIconTestResult>('idle');
  const [errorText, setErrorText] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  useFindIconEditorResetState({ initial, setState, setErrorText, setPreview, setTesting, setTestResult });
  useFindIconEditorPreviewCleanup(preview);

  return {
    errorText,
    fileInputRef,
    preview,
    saving,
    setErrorText,
    setPreview,
    setSaving,
    setState,
    setTesting,
    setTestResult,
    setUploading,
    state,
    testing,
    testResult,
    uploading,
  };
}

function useFindIconEditorHandlers(options: FindIconEditorHandlerOptions): FindIconEditorHandlers {
  return {
    onPickAction: createPickActionHandler(options.setState),
    onSave: createSaveFindIconHandler(options),
    onUpload: createUploadTemplateHandler(options),
    onRemoveImage: createRemoveImageHandler(options),
    onTest: createTestFindIconHandler(options),
  };
}

function createPickActionHandler(
  setState: Dispatch<SetStateAction<FindIconEditorPanelState>>,
): (hover: boolean) => void {
  return function pickAction(hover) {
    setState((current) => ({ ...current, hoverAfterMatch: hover }));
  };
}

function createUploadTemplateHandler(options: {
  fallbackError: string;
  setErrorText: Dispatch<SetStateAction<string>>;
  setPreview: Dispatch<SetStateAction<FindIconEditorPreview | null>>;
  setState: Dispatch<SetStateAction<FindIconEditorPanelState>>;
  setTesting: Dispatch<SetStateAction<boolean>>;
  setTestResult: Dispatch<SetStateAction<FindIconTestResult>>;
  setUploading: Dispatch<SetStateAction<boolean>>;
}): (event: ChangeEvent<HTMLInputElement>) => void {
  const { setTesting, setTestResult } = options;

  return function uploadTemplate(event) {
    resetFindIconTestState(setTesting, setTestResult);
    void handleUploadTemplateFile({ ...options, files: event.target.files });
  };
}

function createRemoveImageHandler(options: {
  fileInputRef: RefObject<HTMLInputElement>;
  setErrorText: Dispatch<SetStateAction<string>>;
  setPreview: Dispatch<SetStateAction<FindIconEditorPreview | null>>;
  setState: Dispatch<SetStateAction<FindIconEditorPanelState>>;
  setTesting: Dispatch<SetStateAction<boolean>>;
  setTestResult: Dispatch<SetStateAction<FindIconTestResult>>;
}): () => void {
  const { fileInputRef, setState, setPreview, setTesting, setTestResult, setErrorText } = options;

  return function removeImage() {
    clearFindIconTemplateSelection({ fileInputRef, setState, setPreview, setErrorText });
    resetFindIconTestState(setTesting, setTestResult);
  };
}

function createTestFindIconHandler(options: {
  fallbackError: string;
  setErrorText: Dispatch<SetStateAction<string>>;
  setTesting: Dispatch<SetStateAction<boolean>>;
  setTestResult: Dispatch<SetStateAction<FindIconTestResult>>;
  state: FindIconEditorPanelState;
}): () => Promise<void> {
  return function testFindIcon() {
    return handleTestFindIcon({
      templatePath: options.state.templatePath,
      hoverAfterMatch: options.state.hoverAfterMatch,
      setTesting: options.setTesting,
      setTestResult: options.setTestResult,
      setErrorText: options.setErrorText,
      fallbackError: options.fallbackError,
    });
  };
}

function createSaveFindIconHandler(options: {
  fallbackError: string;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
  setErrorText: Dispatch<SetStateAction<string>>;
  setSaving: Dispatch<SetStateAction<boolean>>;
  state: FindIconEditorPanelState;
  step: ScreenControlComposerStep;
  stepIndex: number;
}): () => void {
  return function saveFindIcon() {
    handleSaveFindIcon(options);
  };
}

async function handleUploadTemplateFile(options: UploadTemplateFileOptions): Promise<void> {
  const { files, setState, setPreview, setUploading, setErrorText, fallbackError } = options;
  const file = files?.[0];
  if (!file) {
    return;
  }
  setUploading(true);
  setErrorText('');
  try {
    const uploaded = await uploadFindIconTemplateFile(file);
    applyUploadedFindIconTemplate({ file, uploaded, setState, setPreview });
  } catch (error) {
    setErrorText(toErrorMessage(error, fallbackError));
  } finally {
    setUploading(false);
  }
}

async function uploadFindIconTemplateFile(file: File): Promise<FindIconTemplateUploadResponse> {
  const drafts = await createChatImageDrafts([file]);
  const draft = drafts[0];
  if (!draft?.content.url) {
    throw new Error('template image url is missing');
  }
  return uploadFindIconTemplate({
    filename: file.name,
    mime_type: file.type,
    data_url: draft.content.url,
  });
}

function applyUploadedFindIconTemplate(options: {
  file: File;
  setPreview: Dispatch<SetStateAction<FindIconEditorPreview | null>>;
  setState: Dispatch<SetStateAction<FindIconEditorPanelState>>;
  uploaded: FindIconTemplateUploadResponse;
}): void {
  const { file, uploaded, setState, setPreview } = options;
  const preview = createRevocableFindIconPreview(file);
  setState((current) => ({
    ...current,
    templatePath: uploaded.template_path,
    templateName: uploaded.template_name,
  }));
  setPreview((current) => {
    revokeFindIconPreviewURL(current);
    return preview;
  });
}

function createRevocableFindIconPreview(file: File): FindIconEditorPreview {
  return {
    url: URL.createObjectURL(file),
    revocable: true,
  };
}

function resetFindIconTestState(
  setTesting: Dispatch<SetStateAction<boolean>>,
  setTestResult: Dispatch<SetStateAction<FindIconTestResult>>,
): void {
  setTesting(false);
  setTestResult('idle');
}

async function handleTestFindIcon(options: {
  fallbackError: string;
  hoverAfterMatch: boolean;
  setErrorText: Dispatch<SetStateAction<string>>;
  setTesting: Dispatch<SetStateAction<boolean>>;
  setTestResult: Dispatch<SetStateAction<FindIconTestResult>>;
  templatePath: string;
}): Promise<void> {
  const { templatePath, hoverAfterMatch, setTesting, setTestResult, setErrorText, fallbackError } = options;
  setTesting(true);
  setErrorText('');
  try {
    const result = await runFindIconPreview(templatePath, hoverAfterMatch);
    setTestResult(result.exists ? 'success' : 'failure');
  } catch (error) {
    setTestResult('failure');
    setErrorText(toErrorMessage(error, fallbackError));
  } finally {
    setTesting(false);
  }
}

async function runFindIconPreview(templatePath: string, hoverAfterMatch: boolean) {
  return previewFindIcon({
    template_path: requireFindIconTemplatePath(templatePath),
    max_results: 1,
    threshold: FIND_ICON_PREVIEW_THRESHOLD,
    hover_after_match: hoverAfterMatch,
  });
}

function handleSaveFindIcon(options: {
  fallbackError: string;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
  setErrorText: Dispatch<SetStateAction<string>>;
  setSaving: Dispatch<SetStateAction<boolean>>;
  state: FindIconEditorPanelState;
  step: ScreenControlComposerStep;
  stepIndex: number;
}): void {
  const { step, stepIndex, state, onSave, setSaving, setErrorText, fallbackError } = options;
  setSaving(true);
  setErrorText('');
  try {
    onSave(stepIndex, buildSavedFindIconStep(step, state));
  } catch (error) {
    setErrorText(toErrorMessage(error, fallbackError));
  } finally {
    setSaving(false);
  }
}

function buildSavedFindIconStep(
  step: ScreenControlComposerStep,
  state: FindIconEditorPanelState,
): ScreenControlComposerStep {
  const current = normalizeFindIconComposerParams(step.params);
  const baseStep = withFindIconComposerParams(step, {
    template_path: requireFindIconTemplatePath(state.templatePath),
    template_name: state.templateName.trim() || undefined,
    threshold: current?.threshold,
    max_results: current?.max_results,
  });
  return mergeFindIconHoverAction(baseStep, state.hoverAfterMatch);
}

function requireFindIconTemplatePath(templatePath: string): string {
  const path = templatePath.trim();
  if (!path) {
    throw new Error('template_path is required');
  }
  return path;
}
