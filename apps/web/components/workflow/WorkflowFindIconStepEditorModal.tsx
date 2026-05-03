'use client';
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type Dispatch,
  type RefObject,
  type SetStateAction,
} from 'react';
import { createPortal } from 'react-dom';
import { createChatImageDrafts } from '@/lib/chatImageDrafts';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import { previewFindIcon, uploadFindIconTemplate } from '@/lib/api/tools/findIcon';
import {
  normalizeFindIconComposerParams,
  withFindIconComposerParams,
  type ScreenControlComposerStep,
} from '@/lib/workflow-editor';
import {
  FindIconEditorPanel,
  FindIconEditorShell,
  type FindIconEditorPanelState,
  type FindIconEditorPreview,
  type FindIconTestResult,
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
interface WorkflowFindIconStepEditorModalProps {
  open: boolean;
  stepIndex: number;
  step: ScreenControlComposerStep;
  onClose: () => void;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
}
interface FindIconEditorHandlerOptions {
  step: ScreenControlComposerStep;
  stepIndex: number;
  state: FindIconEditorPanelState;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
  fallbackError: string;
  fileInputRef: RefObject<HTMLInputElement>;
  setState: Dispatch<SetStateAction<FindIconEditorPanelState>>;
  setPreview: Dispatch<SetStateAction<FindIconEditorPreview | null>>;
  setUploading: (value: boolean) => void;
  setSaving: (value: boolean) => void;
  setTesting: (value: boolean) => void;
  setTestResult: (value: FindIconTestResult) => void;
  setErrorText: (value: string) => void;
}

const FIND_ICON_PREVIEW_THRESHOLD = 0.96;

export function WorkflowFindIconStepEditorModal(props: WorkflowFindIconStepEditorModalProps) {
  const { open } = props;
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  if (!mounted || !open) {
    return null;
  }
  return createPortal(<WorkflowFindIconStepEditorModalContent {...props} />, document.body);
}
function WorkflowFindIconStepEditorModalContent(props: WorkflowFindIconStepEditorModalProps) {
  const { copy } = useWebLocale();
  const { step, stepIndex, onClose, onSave } = props;
  const titleID = 'workflow-screen-find-icon-editor-title'; const initial = useMemo(() => buildFindIconInitialEditorState(step), [step]);
  const [state, setState] = useState<FindIconEditorPanelState>(initial);
  const [preview, setPreview] = useState<FindIconEditorPreview | null>(null);
  const [uploading, setUploading] = useState(false), [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false), [testResult, setTestResult] = useState<FindIconTestResult>('idle');
  const [errorText, setErrorText] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);
  useFindIconEditorResetState(initial, setState, setErrorText, setPreview, setTesting, setTestResult);
  useFindIconEditorPreviewCleanup(preview);
  const handlers = useFindIconEditorHandlers({
    step,
    stepIndex,
    state,
    onSave,
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
  return (
    <FindIconEditorShell closeAria={copy.workflow.findIconEditorCloseAria} titleID={titleID} onClose={onClose}>
      <FindIconEditorPanel
        closeAria={copy.workflow.findIconEditorCloseAria}
        titleID={titleID}
        stepTag={buildFindIconStepTag(stepIndex)}
        state={state}
        uploading={uploading}
        saving={saving}
        testing={testing}
        testResult={testResult}
        errorText={errorText}
        preview={preview}
        fileInputRef={fileInputRef}
        onClose={onClose}
        onPickAction={handlers.onPickAction}
        onSave={handlers.onSave}
        onUpload={handlers.onUpload}
        onRemoveImage={handlers.onRemoveImage}
        onTest={handlers.onTest}
      />
    </FindIconEditorShell>
  );
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
      clearFindIconTemplateSelection(fileInputRef, setState, setPreview, setErrorText);
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
  files: FileList | null;
  setState: Dispatch<SetStateAction<FindIconEditorPanelState>>;
  setPreview: Dispatch<SetStateAction<FindIconEditorPreview | null>>;
  setUploading: (value: boolean) => void;
  setErrorText: (value: string) => void;
  fallbackError: string;
}) {
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
  templatePath: string;
  hoverAfterMatch: boolean;
  setTesting: (value: boolean) => void;
  setTestResult: (value: FindIconTestResult) => void;
  setErrorText: (value: string) => void;
  fallbackError: string;
}) {
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
  step: ScreenControlComposerStep;
  stepIndex: number;
  state: FindIconEditorPanelState;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
  setSaving: (value: boolean) => void;
  setErrorText: (value: string) => void;
  fallbackError: string;
}) {
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
