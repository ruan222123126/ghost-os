'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { WorkflowCanvasPropertiesPanel } from '@/components/workflow/WorkflowCanvasPropertiesPanel';
import { WorkflowCanvasSettingsModal } from '@/components/workflow/WorkflowCanvasSettingsModal';
import { WorkflowCanvasSidebar } from '@/components/workflow/WorkflowCanvasSidebar';
import { WorkflowCanvasStage } from '@/components/workflow/WorkflowCanvasStage';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import { useWebLocale } from '@/lib/i18n/provider';
import type { WebLocale } from '@/lib/i18n/locale';
import type {
  AutosaveState,
  WorkflowAgentRuntimeCatalog,
  WorkflowCanvasDraft,
  WorkflowEditorKind,
  WorkflowCanvasNodeDraft,
  WorkflowCanvasPosition,
  WorkflowNodeType,
} from '@/lib/workflow-editor';

interface WorkflowCanvasWorkbenchProps {
  editorKind: WorkflowEditorKind;
  draft: WorkflowCanvasDraft;
  autosaveState: AutosaveState;
  actionError: string;
  validationErrors: string[];
  importSessionID: string;
  importLoading: boolean;
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  agentRuntimeLoading: boolean;
  agentRuntimeError: string;
  workflowCopy?: WorkflowCopy;
  nodeLibraryTypes?: readonly WorkflowNodeType[];
  localizeValidationError?: (message: string, locale: WebLocale) => string;
  onChangeImportSessionID: (value: string) => void;
  onImportFromSession: () => void;
  onScheduleChange: (patch: Partial<WorkflowCanvasDraft['schedule']>) => void;
  onAddNode: (type: WorkflowNodeType, position: WorkflowCanvasPosition) => void;
  onSelectNode: (nodeID?: string) => void;
  onMoveNode: (nodeID: string, position: WorkflowCanvasPosition) => void;
  onConnectNodes: (sourceNodeID: string, targetNodeID: string) => void;
  onDeleteEdge: (edgeID: string) => void;
  onDuplicateNode: (nodeID: string) => void;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
  onDeleteNode: (nodeID: string) => void;
  onSave: () => void;
  onBack: () => void;
}

export function WorkflowCanvasWorkbench(props: WorkflowCanvasWorkbenchProps) {
  const { copy } = useWebLocale();
  const {
    editorKind,
    draft,
    autosaveState,
    actionError,
    validationErrors,
    importSessionID,
    importLoading,
    agentRuntimeCatalog,
    agentRuntimeLoading,
    agentRuntimeError,
    workflowCopy = copy.workflow,
    nodeLibraryTypes,
    localizeValidationError,
    onChangeImportSessionID,
    onImportFromSession,
    onScheduleChange,
    onAddNode,
    onSelectNode,
    onMoveNode,
    onConnectNodes,
    onDeleteEdge,
    onDuplicateNode,
    onUpdateNode,
    onDeleteNode,
    onSave,
    onBack,
  } = props;
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const selectedNode = useMemo(
    () => draft.nodes.find((node) => node.id === draft.selectedNodeId),
    [draft.nodes, draft.selectedNodeId],
  );
  useSaveShortcut(onSave);

  return (
    <main className="workflow-arch-root">
      <div className="workflow-arch-watermark">{workflowCopy.watermark}</div>
      <WorkflowCanvasSidebar
        isOpen={isSidebarOpen}
        draft={draft}
        autosaveState={autosaveState}
        importSessionID={importSessionID}
        importLoading={importLoading}
        workflowCopy={workflowCopy}
        nodeLibraryTypes={nodeLibraryTypes}
        localizeValidationError={localizeValidationError}
        onToggle={() => setIsSidebarOpen((open) => !open)}
        onAddNode={onAddNode}
        onChangeImportSessionID={onChangeImportSessionID}
        onImportFromSession={onImportFromSession}
        onScheduleChange={onScheduleChange}
        onOpenSettings={() => setIsSettingsOpen(true)}
        onBack={onBack}
        onSave={onSave}
      />
      <WorkflowCanvasStage
        draft={draft}
        actionError={actionError}
        validationErrors={validationErrors}
        workflowCopy={workflowCopy}
        localizeValidationError={localizeValidationError}
        onSelectNode={onSelectNode}
        onMoveNode={onMoveNode}
        onConnectNodes={onConnectNodes}
        onDeleteEdge={onDeleteEdge}
        onDuplicateNode={onDuplicateNode}
        onDeleteNode={onDeleteNode}
      />
      <WorkflowCanvasPropertiesPanel
        editorKind={editorKind}
        selectedNode={selectedNode}
        agentRuntimeCatalog={agentRuntimeCatalog}
        agentRuntimeLoading={agentRuntimeLoading}
        agentRuntimeError={agentRuntimeError}
        onClose={() => onSelectNode(undefined)}
        onUpdateNode={onUpdateNode}
        onDeleteNode={onDeleteNode}
      />
      <WorkflowCanvasSettingsModal
        open={isSettingsOpen}
        schedule={draft.schedule}
        importSessionID={importSessionID}
        importLoading={importLoading}
        workflowCopy={workflowCopy}
        onClose={() => setIsSettingsOpen(false)}
        onScheduleChange={onScheduleChange}
        onChangeImportSessionID={onChangeImportSessionID}
        onImportFromSession={onImportFromSession}
      />
    </main>
  );
}

function useSaveShortcut(onSave: () => void) {
  const onSaveRef = useRef(onSave);
  useEffect(() => {
    onSaveRef.current = onSave;
  }, [onSave]);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      const key = event.key.toLowerCase();
      if (key !== 's' || (!event.metaKey && !event.ctrlKey)) {
        return;
      }
      event.preventDefault();
      onSaveRef.current();
    };
    window.addEventListener('keydown', handleKeyDown, true);
    return () => window.removeEventListener('keydown', handleKeyDown, true);
  }, []);
}
