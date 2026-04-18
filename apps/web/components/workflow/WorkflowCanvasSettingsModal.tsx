'use client';

import type { WorkflowCanvasDraft } from '@/lib/workflow-editor';
import { useWebLocale } from '@/lib/i18n/provider';

interface WorkflowCanvasSettingsModalProps {
  open: boolean;
  schedule: WorkflowCanvasDraft['schedule'];
  importSessionID: string;
  importLoading: boolean;
  onClose: () => void;
  onScheduleChange: (patch: Partial<WorkflowCanvasDraft['schedule']>) => void;
  onChangeImportSessionID: (value: string) => void;
  onImportFromSession: () => void;
}

export function WorkflowCanvasSettingsModal(props: WorkflowCanvasSettingsModalProps) {
  const { copy } = useWebLocale();
  const {
    open,
    schedule,
    importSessionID,
    importLoading,
    onClose,
    onScheduleChange,
    onChangeImportSessionID,
    onImportFromSession,
  } = props;

  if (!open) {
    return null;
  }

  return (
    <div className="workflow-arch-settings-popover" role="dialog" aria-modal="true" aria-labelledby="workflow-settings-title">
      <button type="button" className="workflow-arch-settings-backdrop" onClick={onClose} aria-label={copy.workflow.closeWorkflowSettingsAria} />
      <section className="workflow-arch-settings-panel">
        <button type="button" className="workflow-arch-settings-close" onClick={onClose} aria-label={copy.workflow.closeWorkflowSettingsAria}>
          <IconClose />
        </button>
        <header className="workflow-arch-settings-head">
          <h2 id="workflow-settings-title">{copy.workflow.modalTitle}</h2>
          <p>{copy.workflow.modalDescription}</p>
        </header>
        <div className="workflow-arch-settings-body">
          <label>
            <span>{copy.workflow.modalScheduleMode}</span>
            <select
              value={schedule.mode}
              onChange={(event) =>
                onScheduleChange({ mode: event.target.value as WorkflowCanvasDraft['schedule']['mode'] })}
            >
              <option value="interval">{copy.workflow.modalInterval}</option>
              <option value="cron">{copy.workflow.modalCron}</option>
            </select>
          </label>
          {schedule.mode === 'interval' ? (
            <label>
              <span>{copy.workflow.modalIntervalSeconds}</span>
              <input
                value={schedule.intervalSeconds}
                placeholder="300"
                onChange={(event) => onScheduleChange({ intervalSeconds: event.target.value })}
              />
            </label>
          ) : (
            <label>
              <span>{copy.workflow.modalCronExpr}</span>
              <input
                value={schedule.cronExpr}
                placeholder="*/5 * * * *"
                onChange={(event) => onScheduleChange({ cronExpr: event.target.value })}
              />
            </label>
          )}
          <label>
            <span>{copy.workflow.modalSessionID}</span>
            <input
              value={importSessionID}
              placeholder="session-xxxx"
              onChange={(event) => onChangeImportSessionID(event.target.value)}
            />
          </label>
          <button type="button" className="workflow-arch-settings-import" onClick={onImportFromSession} disabled={importLoading}>
            {importLoading ? copy.workflow.modalImporting : copy.workflow.modalImportTextTasks}
          </button>
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
