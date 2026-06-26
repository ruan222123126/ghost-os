'use client';

import { CloseButton } from '@/components/CloseButton';
import type { WorkflowCanvasDraft } from '@/lib/workflow-editor';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';

export interface WorkflowImportControls {
  sessionID: string;
  loading: boolean;
  onChangeSessionID: (value: string) => void;
  onImportFromSession: () => void;
}

interface WorkflowCanvasSettingsModalProps {
  open: boolean;
  schedule: WorkflowCanvasDraft['schedule'];
  importControls?: WorkflowImportControls;
  workflowCopy: WorkflowCopy;
  onClose: () => void;
  onScheduleChange: (patch: Partial<WorkflowCanvasDraft['schedule']>) => void;
}

interface SettingsPanelProps {
  importControls?: WorkflowImportControls;
  schedule: WorkflowCanvasDraft['schedule'];
  workflowCopy: WorkflowCopy;
  onClose: () => void;
  onScheduleChange: (patch: Partial<WorkflowCanvasDraft['schedule']>) => void;
}

interface ScheduleSettingsFieldsProps {
  schedule: WorkflowCanvasDraft['schedule'];
  workflowCopy: WorkflowCopy;
  onScheduleChange: (patch: Partial<WorkflowCanvasDraft['schedule']>) => void;
}

type ScheduleMode = WorkflowCanvasDraft['schedule']['mode'];

export function WorkflowCanvasSettingsModal(props: WorkflowCanvasSettingsModalProps) {
  const {
    open,
    schedule,
    importControls,
    workflowCopy,
    onClose,
    onScheduleChange,
  } = props;

  if (!open) {
    return null;
  }

  return (
    <div className="workflow-arch-settings-popover" role="dialog" aria-modal="true" aria-labelledby="workflow-settings-title">
      <button type="button" className="workflow-arch-settings-backdrop" onClick={onClose} aria-label={workflowCopy.closeWorkflowSettingsAria} />
      <SettingsPanel
        importControls={importControls}
        schedule={schedule}
        workflowCopy={workflowCopy}
        onClose={onClose}
        onScheduleChange={onScheduleChange}
      />
    </div>
  );
}

function SettingsPanel(props: SettingsPanelProps) {
  const {
    importControls,
    schedule,
    workflowCopy,
    onClose,
    onScheduleChange,
  } = props;

  return (
    <section className="workflow-arch-settings-panel">
      <CloseButton className="absolute right-5 top-5 z-10" onClick={onClose} aria-label={workflowCopy.closeWorkflowSettingsAria} />
      <SettingsHeader workflowCopy={workflowCopy} />
      <div className="workflow-arch-settings-body">
        <ScheduleSettingsFields
          schedule={schedule}
          workflowCopy={workflowCopy}
          onScheduleChange={onScheduleChange}
        />
        <ImportSessionFields importControls={importControls} workflowCopy={workflowCopy} />
      </div>
    </section>
  );
}

function SettingsHeader(props: { workflowCopy: WorkflowCopy }) {
  const { workflowCopy } = props;

  return (
    <header className="workflow-arch-settings-head">
      <h2 id="workflow-settings-title">{workflowCopy.modalTitle}</h2>
      <p>{workflowCopy.modalDescription}</p>
    </header>
  );
}

function ScheduleSettingsFields(props: ScheduleSettingsFieldsProps) {
  const { schedule, workflowCopy, onScheduleChange } = props;

  return (
    <>
      <label>
        <span>{workflowCopy.modalScheduleMode}</span>
        <select
          value={schedule.mode}
          onChange={(event) => onScheduleChange({ mode: event.target.value as ScheduleMode })}
        >
          <option value="interval">{workflowCopy.modalInterval}</option>
          <option value="cron">{workflowCopy.modalCron}</option>
        </select>
      </label>
      <ScheduleDetailField
        schedule={schedule}
        workflowCopy={workflowCopy}
        onScheduleChange={onScheduleChange}
      />
    </>
  );
}

function ScheduleDetailField(props: ScheduleSettingsFieldsProps) {
  const { schedule, workflowCopy, onScheduleChange } = props;

  if (schedule.mode === 'interval') {
    return (
      <label>
        <span>{workflowCopy.modalIntervalSeconds}</span>
        <input
          value={schedule.intervalSeconds}
          placeholder="300"
          onChange={(event) => onScheduleChange({ intervalSeconds: event.target.value })}
        />
      </label>
    );
  }

  return (
    <label>
      <span>{workflowCopy.modalCronExpr}</span>
      <input
        value={schedule.cronExpr}
        placeholder="*/5 * * * *"
        onChange={(event) => onScheduleChange({ cronExpr: event.target.value })}
      />
    </label>
  );
}

function ImportSessionFields(props: {
  importControls?: WorkflowImportControls;
  workflowCopy: WorkflowCopy;
}) {
  const { importControls, workflowCopy } = props;

  if (!importControls) {
    return null;
  }

  return (
    <>
      <label>
        <span>{workflowCopy.modalSessionID}</span>
        <input
          value={importControls.sessionID}
          placeholder="session-xxxx"
          onChange={(event) => importControls.onChangeSessionID(event.target.value)}
        />
      </label>
      <button
        type="button"
        className="workflow-arch-settings-import"
        onClick={importControls.onImportFromSession}
        disabled={importControls.loading}
      >
        {importControls.loading ? workflowCopy.modalImporting : workflowCopy.modalImportTextTasks}
      </button>
    </>
  );
}
