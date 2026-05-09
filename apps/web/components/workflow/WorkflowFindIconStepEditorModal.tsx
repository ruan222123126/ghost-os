'use client';
import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { useWorkflowFindIconStepEditor } from '@/hooks/workflow/useWorkflowFindIconStepEditor';
import type { ScreenControlComposerStep } from '@/lib/workflow-editor';
import {
  FindIconEditorPanel,
  FindIconEditorShell,
} from '@/components/workflow/WorkflowFindIconStepEditorModalView';
interface WorkflowFindIconStepEditorModalProps {
  open: boolean;
  stepIndex: number;
  step: ScreenControlComposerStep;
  onClose: () => void;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
}

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
  const editor = useWorkflowFindIconStepEditor(props);
  return (
    <FindIconEditorShell closeAria={editor.closeAria} titleID={editor.titleID} onClose={props.onClose}>
      <FindIconEditorPanel
        closeAria={editor.closeAria}
        titleID={editor.titleID}
        stepTag={editor.stepTag}
        state={editor.state}
        uploading={editor.uploading}
        saving={editor.saving}
        testing={editor.testing}
        testResult={editor.testResult}
        errorText={editor.errorText}
        preview={editor.preview}
        fileInputRef={editor.fileInputRef}
        onClose={props.onClose}
        onPickAction={editor.onPickAction}
        onSave={editor.onSave}
        onUpload={editor.onUpload}
        onRemoveImage={editor.onRemoveImage}
        onTest={editor.onTest}
      />
    </FindIconEditorShell>
  );
}
