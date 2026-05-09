'use client';

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
      <section className="workflow-arch-settings-panel">
        <button type="button" className="workflow-arch-settings-close" onClick={onClose} aria-label={workflowCopy.closeWorkflowSettingsAria}>
          <IconClose />
        </button>
        <header className="workflow-arch-settings-head">
          <h2 id="workflow-settings-title">{workflowCopy.modalTitle}</h2>
          <p>{workflowCopy.modalDescription}</p>
        </header>
        <div className="workflow-arch-settings-body">
          <label>
            <span>{workflowCopy.modalScheduleMode}</span>
            <select
              value={schedule.mode}
              onChange={(event) =>
                onScheduleChange({ mode: event.target.value as WorkflowCanvasDraft['schedule']['mode'] })}
            >
              <option value="interval">{workflowCopy.modalInterval}</option>
              <option value="cron">{workflowCopy.modalCron}</option>
            </select>
          </label>
          {schedule.mode === 'interval' ? (
            <label>
              <span>{workflowCopy.modalIntervalSeconds}</span>
              <input
                value={schedule.intervalSeconds}
                placeholder="300"
                onChange={(event) => onScheduleChange({ intervalSeconds: event.target.value })}
              />
            </label>
          ) : (
            <label>
              <span>{workflowCopy.modalCronExpr}</span>
              <input
                value={schedule.cronExpr}
                placeholder="*/5 * * * *"
                onChange={(event) => onScheduleChange({ cronExpr: event.target.value })}
              />
            </label>
          )}
          {importControls ? (
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
          ) : null}
        </div>
      </section>
    </div>
  );
}

function IconClose(props: { size?: number }) {
  const { size = 18 } = props;
  return (
    <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path d="M5 5 15 15" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
      <path d="M15 5 5 15" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
    </svg>
  );
}
