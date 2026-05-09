'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { WorkflowCanvasPropertiesPanel } from '@/components/workflow/WorkflowCanvasPropertiesPanel';
import {
  WorkflowCanvasSettingsModal,
  type WorkflowImportControls,
} from '@/components/workflow/WorkflowCanvasSettingsModal';
import { WorkflowCanvasSidebar } from '@/components/workflow/WorkflowCanvasSidebar';
import { WorkflowCanvasStage } from '@/components/workflow/WorkflowCanvasStage';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import { useWebLocale } from '@/lib/i18n/provider';
import type { WebLocale } from '@/lib/i18n/locale';
import type { PresetPayload } from '@/lib/types';
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
  importControls?: WorkflowImportControls;
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  agentRuntimeLoading: boolean;
  agentRuntimeError: string;
  presets?: PresetPayload[];
  presetLoading?: boolean;
  presetError?: string;
  workflowCopy?: WorkflowCopy;
  nodeLibraryTypes?: readonly WorkflowNodeType[];
  localizeValidationError?: (message: string, locale: WebLocale) => string;
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
    importControls,
    agentRuntimeCatalog,
    agentRuntimeLoading,
    agentRuntimeError,
    presets,
    presetLoading = false,
    presetError = '',
    workflowCopy = copy.workflow,
    nodeLibraryTypes,
    localizeValidationError,
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
        autosaveState={autosaveState}
        workflowCopy={workflowCopy}
        nodeLibraryTypes={nodeLibraryTypes}
        localizeValidationError={localizeValidationError}
        onToggle={() => setIsSidebarOpen((open) => !open)}
        onAddNode={onAddNode}
        onOpenSettings={() => setIsSettingsOpen(true)}
        onBack={onBack}
        onSave={onSave}
      />
      <WorkflowCanvasStage
        editorKind={editorKind}
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
        draft={draft}
        selectedNode={selectedNode}
        agentRuntimeCatalog={agentRuntimeCatalog}
        agentRuntimeLoading={agentRuntimeLoading}
        agentRuntimeError={agentRuntimeError}
        presets={presets}
        presetLoading={presetLoading}
        presetError={presetError}
        onClose={() => onSelectNode(undefined)}
        onUpdateNode={onUpdateNode}
        onDeleteNode={onDeleteNode}
      />
      <WorkflowCanvasSettingsModal
        open={isSettingsOpen}
        schedule={draft.schedule}
        importControls={importControls}
        workflowCopy={workflowCopy}
        onClose={() => setIsSettingsOpen(false)}
        onScheduleChange={onScheduleChange}
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
