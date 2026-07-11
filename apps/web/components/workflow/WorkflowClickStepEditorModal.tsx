'use client';

import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { useWorkflowClickStepEditor } from '@/hooks/workflow/useWorkflowClickStepEditor';
import type { ScreenControlComposerStep } from '@/lib/workflow-editor';
import { WorkflowClickStepEditorModalView } from '@/components/workflow/WorkflowClickStepEditorModalView';

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
  const editor = useWorkflowClickStepEditor(props);

  return (
    <WorkflowClickStepEditorModalView
      locale={editor.locale}
      closeAria={editor.closeAria}
      titleID={editor.titleID}
      title={editor.title}
      stepTag={editor.stepTag}
      errorText={editor.errorText}
      state={editor.state}
      canUseFindIconReference={editor.canUseFindIconReference}
      saving={editor.saving}
      captureActive={editor.captureActive}
      captureLoading={editor.captureLoading}
      onClose={props.onClose}
      onSave={editor.onSave}
      onStartCapture={editor.onStartCapture}
      onStateChange={editor.onStateChange}
    />
  );
}
