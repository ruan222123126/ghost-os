'use client';

import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { CloseButton } from '@/components/CloseButton';
import { WorkflowClickStepEditorModal } from '@/components/workflow/WorkflowClickStepEditorModal';
import { WorkflowFindIconStepEditorModal } from '@/components/workflow/WorkflowFindIconStepEditorModal';
import { useWebLocale } from '@/lib/i18n/provider';
import {
  SCREEN_CONTROL_COMPOSER_ACTIONS,
  normalizeScreenControlComposerAction,
  type ScreenControlAtomicAction,
  type ScreenControlComposerStep,
} from '@/lib/workflow-editor';

interface WorkflowScreenControlComposerModalProps {
  open: boolean;
  steps: ScreenControlComposerStep[];
  onClose: () => void;
  onAppend: (action: ScreenControlAtomicAction) => void;
  onMove: (index: number, direction: 'up' | 'down') => void;
  onRemove: (index: number) => void;
  onUpdateStep: (index: number, step: ScreenControlComposerStep) => void;
}

interface ComposerQueuePanelProps {
  steps: ScreenControlComposerStep[];
  onMove: (index: number, direction: 'up' | 'down') => void;
  onRemove: (index: number) => void;
  onUpdateStep: (index: number, step: ScreenControlComposerStep) => void;
}

interface ComposerStepEditing {
  action: 'find_icon' | 'click';
  index: number;
  step: ScreenControlComposerStep;
}

export function WorkflowScreenControlComposerModal(props: WorkflowScreenControlComposerModalProps) {
  const { open } = props;
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted || !open) {
    return null;
  }

  return createPortal(
    <WorkflowScreenControlComposerModalContent {...props} />,
    document.body,
  );
}

function WorkflowScreenControlComposerModalContent(props: WorkflowScreenControlComposerModalProps) {
  const { copy } = useWebLocale();
  const { steps, onClose, onAppend, onMove, onRemove, onUpdateStep } = props;

  return (
    <div className="workflow-arch-screen-composer" role="dialog" aria-modal="true" aria-labelledby="workflow-screen-composer-title">
      <button type="button" className="workflow-arch-screen-composer-backdrop" onClick={onClose} aria-label={copy.workflow.closeScreenComposerAria} />
      <section className="workflow-arch-screen-composer-panel">
        <header className="workflow-arch-screen-composer-head">
          <div>
            <h3 id="workflow-screen-composer-title">{copy.workflow.screenComposerTitle}</h3>
            <p>{copy.workflow.screenComposerDescription}</p>
          </div>
          <CloseButton className="shrink-0" onClick={onClose} aria-label={copy.workflow.closeScreenComposerAria} />
        </header>
        <div className="workflow-arch-screen-composer-body">
          <ComposerActionsPanel onAppend={onAppend} />
          <ComposerQueuePanel steps={steps} onMove={onMove} onRemove={onRemove} onUpdateStep={onUpdateStep} />
        </div>
      </section>
    </div>
  );
}

function ComposerActionsPanel(props: { onAppend: (action: ScreenControlAtomicAction) => void }) {
  const { copy } = useWebLocale();
  const { onAppend } = props;

  return (
    <section className="workflow-arch-screen-composer-actions">
      <h4>{copy.workflow.screenComposerActionsTitle}</h4>
      <div className="workflow-arch-screen-composer-action-list">
        {SCREEN_CONTROL_COMPOSER_ACTIONS.map((action) => (
          <button key={action} type="button" className="workflow-arch-screen-composer-action" onClick={() => onAppend(action)}>
            {action}
          </button>
        ))}
      </div>
    </section>
  );
}

function ComposerQueuePanel(props: ComposerQueuePanelProps) {
  const { copy } = useWebLocale();
  const { steps, onMove, onRemove, onUpdateStep } = props;
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const editing = resolveComposerStepEditing(steps, editingIndex);

  useEffect(() => {
    if (editingIndex !== null && !resolveComposerStepEditing(steps, editingIndex)) {
      setEditingIndex(null);
    }
  }, [editingIndex, steps]);

  return (
    <section className="workflow-arch-screen-composer-queue">
      <h4>{copy.workflow.screenComposerQueueTitle}</h4>
      <ScreenComposerStepList
        steps={steps}
        emptyText={copy.workflow.screenComposerEmpty}
        onMove={onMove}
        onRemove={onRemove}
        onEdit={setEditingIndex}
      />
      <ComposerStepEditor
        editing={editing}
        steps={steps}
        onClose={() => setEditingIndex(null)}
        onSave={(index, step) => {
          onUpdateStep(index, step);
          setEditingIndex(null);
        }}
      />
    </section>
  );
}

function ScreenComposerStepList(props: {
  steps: ScreenControlComposerStep[];
  emptyText: string;
  onMove: (index: number, direction: 'up' | 'down') => void;
  onRemove: (index: number) => void;
  onEdit: (index: number) => void;
}) {
  const { steps, emptyText, onMove, onRemove, onEdit } = props;

  if (steps.length === 0) {
    return <p className="workflow-arch-screen-composer-empty">{emptyText}</p>;
  }

  return (
    <ol className="workflow-arch-screen-composer-step-list">
      {steps.map((step, index) => (
        <ScreenComposerStepRow
          key={`${step.action}-${index}`}
          step={step}
          index={index}
          total={steps.length}
          onMove={onMove}
          onRemove={onRemove}
          onEdit={() => onEdit(index)}
        />
      ))}
    </ol>
  );
}

function ComposerStepEditor(props: {
  editing: ComposerStepEditing | undefined;
  steps: ScreenControlComposerStep[];
  onClose: () => void;
  onSave: (index: number, step: ScreenControlComposerStep) => void;
}) {
  const { editing, steps, onClose, onSave } = props;

  if (!editing) {
    return null;
  }

  if (editing.action === 'find_icon') {
    return (
      <WorkflowFindIconStepEditorModal
        open
        stepIndex={editing.index}
        step={editing.step}
        onClose={onClose}
        onSave={onSave}
      />
    );
  }

  return (
    <WorkflowClickStepEditorModal
      open
      stepIndex={editing.index}
      step={editing.step}
      canUseFindIconReference={hasEarlierFindIconStep(steps, editing.index)}
      onClose={onClose}
      onSave={onSave}
    />
  );
}

function ScreenComposerStepRow(props: {
  step: ScreenControlComposerStep;
  index: number;
  total: number;
  onMove: (index: number, direction: 'up' | 'down') => void;
  onRemove: (index: number) => void;
  onEdit: () => void;
}) {
  const { copy } = useWebLocale();
  const { step, index, total, onMove, onRemove, onEdit } = props;
  const normalizedAction = normalizeScreenControlComposerAction(step.action);
  const first = index === 0;
  const last = index === total - 1;

  return (
    <li className="workflow-arch-screen-composer-step">
      <span>{copy.workflow.screenComposerStepLabel(index + 1, normalizedAction)}</span>
      <div className="workflow-arch-screen-composer-step-actions">
        {normalizedAction === 'find_icon' || normalizedAction === 'click' ? (
          <button type="button" className="workflow-arch-screen-composer-step-edit" onClick={onEdit}>
            {normalizedAction === 'find_icon'
              ? copy.workflow.screenComposerEditFindIcon
              : copy.workflow.screenComposerEditClick}
          </button>
        ) : null}
        <button type="button" onClick={() => onMove(index, 'up')} disabled={first} aria-label={copy.workflow.screenComposerMoveUpAria}>↑</button>
        <button type="button" onClick={() => onMove(index, 'down')} disabled={last} aria-label={copy.workflow.screenComposerMoveDownAria}>↓</button>
        <button type="button" onClick={() => onRemove(index)} aria-label={copy.workflow.screenComposerDeleteAria}>✕</button>
      </div>
    </li>
  );
}

function resolveComposerStepEditing(
  steps: ScreenControlComposerStep[],
  editingIndex: number | null,
): ComposerStepEditing | undefined {
  if (editingIndex === null) {
    return undefined;
  }
  const step = steps[editingIndex];
  const action = editableComposerAction(step?.action);
  if (!step || !action) {
    return undefined;
  }
  return {
    action,
    index: editingIndex,
    step,
  };
}

function editableComposerAction(action: ScreenControlAtomicAction | undefined): ComposerStepEditing['action'] | undefined {
  if (!action) {
    return undefined;
  }
  const normalized = normalizeScreenControlComposerAction(action);
  if (normalized === 'find_icon' || normalized === 'click') {
    return normalized;
  }
  return undefined;
}

function hasEarlierFindIconStep(steps: ScreenControlComposerStep[], stepIndex: number): boolean {
  for (let index = 0; index < stepIndex; index += 1) {
    if (normalizeScreenControlComposerAction(steps[index]?.action ?? 'screenshot') === 'find_icon') {
      return true;
    }
  }
  return false;
}
