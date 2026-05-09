'use client';

import { WorkflowCanvasWorkbench } from '@/components/workflow/WorkflowCanvasWorkbench';
import { useWorkflowEditorController } from '@/hooks/workflow/useWorkflowEditorController';
import { useWebLocale } from '@/lib/i18n/provider';

interface WorkflowEditorClientProps {
  mode: 'create' | 'edit';
  taskID?: string;
}

export function WorkflowEditorClient(props: WorkflowEditorClientProps) {
  const { copy } = useWebLocale();
  const controller = useWorkflowEditorController({
    mode: props.mode,
    taskID: props.taskID,
    copy,
  });

  if (controller.phase === 'loading') {
    return <main className="workflow-loading">{copy.workflow.loadingTask}</main>;
  }

  return (
    <WorkflowCanvasWorkbench
      editorKind="workflow"
      draft={controller.draft}
      autosaveState={controller.autosaveState}
      actionError={controller.actionError}
      validationErrors={controller.validationErrors}
      importControls={{
        sessionID: controller.importSessionID,
        loading: controller.importLoading,
        onChangeSessionID: controller.onChangeImportSessionID,
        onImportFromSession: controller.onImportFromSession,
      }}
      agentRuntimeCatalog={controller.agentRuntimeCatalog}
      agentRuntimeLoading={controller.agentRuntimeLoading}
      agentRuntimeError={controller.agentRuntimeError}
      onScheduleChange={controller.onScheduleChange}
      onAddNode={controller.onAddNode}
      onSelectNode={controller.onSelectNode}
      onMoveNode={controller.onMoveNode}
      onConnectNodes={controller.onConnectNodes}
      onDeleteEdge={controller.onDeleteEdge}
      onDuplicateNode={controller.onDuplicateNode}
      onUpdateNode={controller.onUpdateNode}
      onDeleteNode={controller.onDeleteNode}
      onSave={controller.onSave}
      onBack={controller.onBack}
    />
  );
}
