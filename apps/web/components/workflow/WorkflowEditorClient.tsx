'use client';

import type { ComponentProps } from 'react';
import { GlobalLoadingOverlay } from '@/components/GlobalLoadingOverlay';
import { WorkflowCanvasWorkbench } from '@/components/workflow/WorkflowCanvasWorkbench';
import { useWorkflowEditorController } from '@/hooks/workflow/useWorkflowEditorController';
import type { UseWorkflowEditorControllerResult } from '@/hooks/workflow/workflowEditorControllerTypes';
import { useWebLocale } from '@/lib/i18n/provider';

interface WorkflowEditorClientProps {
  mode: 'create' | 'edit';
  taskID?: string;
}

type WorkflowWorkbenchProps = ComponentProps<typeof WorkflowCanvasWorkbench>;
type WorkflowImportControls = NonNullable<WorkflowWorkbenchProps['importControls']>;

export function WorkflowEditorClient(props: WorkflowEditorClientProps) {
  const { copy } = useWebLocale();
  const controller = useWorkflowEditorController({
    mode: props.mode,
    taskID: props.taskID,
    copy,
  });

  if (controller.phase === 'loading') {
    return <GlobalLoadingOverlay />;
  }

  return <WorkflowEditorWorkbench controller={controller} />;
}

function WorkflowEditorWorkbench(props: {
  controller: UseWorkflowEditorControllerResult;
}) {
  return <WorkflowCanvasWorkbench {...buildWorkflowWorkbenchProps(props.controller)} />;
}

function buildWorkflowWorkbenchProps(controller: UseWorkflowEditorControllerResult): WorkflowWorkbenchProps {
  return {
    actionError: controller.actionError,
    agentRuntimeCatalog: controller.agentRuntimeCatalog,
    agentRuntimeError: controller.agentRuntimeError,
    agentRuntimeLoading: controller.agentRuntimeLoading,
    autosaveState: controller.autosaveState,
    draft: controller.draft,
    editorKind: 'workflow',
    importControls: buildWorkflowImportControls(controller),
    onAddNode: controller.onAddNode,
    onBack: controller.onBack,
    onConnectNodes: controller.onConnectNodes,
    onDeleteEdge: controller.onDeleteEdge,
    onDeleteNode: controller.onDeleteNode,
    onDuplicateNode: controller.onDuplicateNode,
    onMoveNode: controller.onMoveNode,
    onSave: controller.onSave,
    onScheduleChange: controller.onScheduleChange,
    onSelectNode: controller.onSelectNode,
    onUpdateNode: controller.onUpdateNode,
    validationErrors: controller.validationErrors,
  };
}

function buildWorkflowImportControls(controller: UseWorkflowEditorControllerResult): WorkflowImportControls {
  return {
    loading: controller.importLoading,
    onChangeSessionID: controller.onChangeImportSessionID,
    onImportFromSession: controller.onImportFromSession,
    sessionID: controller.importSessionID,
  };
}
