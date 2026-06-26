'use client';

import { useMemo, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  type AutosaveSnapshot,
  createEmptyWorkflowDraft,
  enabledWorkflowAgentToolNames,
  normalizeWorkflowDraftAgentNodes,
  validateWorkflowDraft,
} from '@/lib/workflow-editor';
import type {
  AutosaveState,
  WorkflowAgentRuntimeCatalog,
  WorkflowCanvasDraft,
  WorkflowUpdatePayload,
} from '@/lib/workflow-editor';
import { useWorkflowAgentRuntimeCatalog } from './useWorkflowAgentRuntimeCatalog';
import type {
  UseWorkflowEditorControllerOptions,
  UseWorkflowEditorControllerResult,
  WorkflowEditorPhase,
} from './workflowEditorControllerTypes';
import {
  type WorkflowEditorControllerRuntime,
  persistWorkflowSnapshot,
  useAutosaveController,
} from './workflowEditorControllerInternals';
import { buildAutosaveSnapshot } from './workflowEditorDraft';
import { useWorkflowEditorLifecycle } from './useWorkflowEditorLifecycle';
import {
  useWorkflowDraftActions,
  useWorkflowImportActions,
  useWorkflowSaveAction,
} from './useWorkflowEditorActions';
import { useWorkflowEditorBackAction } from './useWorkflowEditorBackAction';

interface WorkflowControllerResultOptions {
  core: ReturnType<typeof useWorkflowEditorCore>;
  importActions: ReturnType<typeof useWorkflowImportActions>;
  draftActions: ReturnType<typeof useWorkflowDraftActions>;
  onSave: () => Promise<void>;
  onBack: () => void;
}

interface EditableWorkflowDraftOptions {
  draft: WorkflowCanvasDraft;
  agentNormalizationEnabled: boolean;
  agentRuntimeReady: boolean;
  enabledToolNames: string[];
}

export function useWorkflowEditorController(
  options: UseWorkflowEditorControllerOptions,
): UseWorkflowEditorControllerResult {
  const router = useRouter();
  const core = useWorkflowEditorCore(options, router);
  const importActions = useWorkflowImportActions({
    copy: options.copy,
    agentRuntimeReady: core.agentRuntimeReady,
    enabledToolNames: core.enabledToolNames,
    setDraft: core.setDraft,
    setActionError: core.setActionError,
    setAgentNormalizationEnabled: core.setAgentNormalizationEnabled,
  });
  const draftActions = useWorkflowDraftActions({
    draft: core.draft,
    agentRuntimeReady: core.agentRuntimeReady,
    enabledToolNames: core.enabledToolNames,
    setDraft: core.setDraft,
    setAgentNormalizationEnabled: core.setAgentNormalizationEnabled,
  });
  const handleSave = useWorkflowSaveAction({
    draft: core.draft,
    copy: options.copy,
    agentRuntimeReady: core.agentRuntimeReady,
    enabledToolNames: core.enabledToolNames,
    autosaveController: core.autosaveController,
    validationErrors: core.validationErrors,
    setActionError: core.setActionError,
    setAgentNormalizationEnabled: core.setAgentNormalizationEnabled,
  });
  const handleBack = useWorkflowEditorBackAction(router);
  return buildControllerResult({
    core,
    importActions,
    draftActions,
    onSave: handleSave,
    onBack: handleBack,
  });
}

function useWorkflowEditorCore(
  options: UseWorkflowEditorControllerOptions,
  router: ReturnType<typeof useRouter>,
) {
  const { mode, taskID, copy } = options;
  const [draft, setDraft] = useState<WorkflowCanvasDraft>(() => createEmptyWorkflowDraft(mode));
  const [phase, setPhase] = useState<WorkflowEditorPhase>(mode === 'edit' ? 'loading' : 'ready');
  const [autosaveState, setAutosaveState] = useState<AutosaveState>(() => ({ phase: 'idle', message: 'Autosave idle', updatedAt: Date.now() }));
  const [actionError, setActionError] = useState('');
  const [agentNormalizationEnabled, setAgentNormalizationEnabled] = useState(false);
  const runtimeRef = useRef<WorkflowEditorControllerRuntime>({ taskID, routeReplaced: mode === 'edit', baselineSeeded: false });
  const agentRuntimeState = useWorkflowAgentRuntimeCatalog();
  const derived = useWorkflowEditorDerived(draft, agentNormalizationEnabled, agentRuntimeState);
  const persistSnapshotRef = useRef<(snapshot: AutosaveSnapshot<WorkflowUpdatePayload>) => Promise<void>>(async () => undefined);

  persistSnapshotRef.current = async (snapshot) =>
    persistWorkflowSnapshot({ snapshot, runtimeRef, setDraft, router });

  const autosaveController = useAutosaveController(persistSnapshotRef, setAutosaveState);
  useWorkflowEditorLifecycle({
    mode,
    taskID,
    copy,
    autosaveController,
    runtimeRef,
    draft,
    currentSnapshot: derived.snapshotBuild.snapshot,
    isLoading: phase === 'loading',
    snapshotErrorMessage: derived.snapshotBuild.errorMessage,
    validationErrors: derived.validationErrors,
    agentNormalizationEnabled,
    agentRuntimeReady: derived.agentRuntimeReady,
    enabledToolNames: derived.enabledToolNames,
    setDraft,
    setActionError,
    setPhase,
  });

  return { phase, draft, autosaveState, actionError, agentRuntimeState, autosaveController, setDraft, setActionError, setAgentNormalizationEnabled, ...derived };
}

function useWorkflowEditorDerived(
  draft: WorkflowCanvasDraft,
  agentNormalizationEnabled: boolean,
  agentRuntimeState: ReturnType<typeof useWorkflowAgentRuntimeCatalog>,
) {
  const catalog = agentRuntimeState.catalog;
  const agentRuntimeReady = catalog !== undefined;
  const enabledToolNames = useEnabledWorkflowToolNames(catalog);
  const editableDraft = useEditableWorkflowDraft({
    draft,
    agentNormalizationEnabled,
    agentRuntimeReady,
    enabledToolNames,
  });
  const validation = useWorkflowDraftValidation(editableDraft, catalog);
  const snapshotBuild = useMemo(() => buildAutosaveSnapshot(editableDraft), [editableDraft]);

  return {
    agentRuntimeReady,
    enabledToolNames,
    validationErrors: validation.errors,
    snapshotBuild,
  };
}

function useEnabledWorkflowToolNames(catalog: WorkflowAgentRuntimeCatalog | undefined): string[] {
  return useMemo(
    () => enabledWorkflowAgentToolNames(catalog?.tools ?? []),
    [catalog?.tools],
  );
}

function useEditableWorkflowDraft(options: EditableWorkflowDraftOptions): WorkflowCanvasDraft {
  const { draft, agentNormalizationEnabled, agentRuntimeReady, enabledToolNames } = options;
  return useMemo(() => {
    if (!agentNormalizationEnabled || !agentRuntimeReady) {
      return draft;
    }
    return normalizeWorkflowDraftAgentNodes(draft, enabledToolNames);
  }, [agentNormalizationEnabled, agentRuntimeReady, draft, enabledToolNames]);
}

function useWorkflowDraftValidation(
  editableDraft: WorkflowCanvasDraft,
  catalog: WorkflowAgentRuntimeCatalog | undefined,
) {
  return useMemo(() => validateWorkflowDraft(editableDraft, {
    agentRuntimeCatalog: catalog,
  }), [catalog, editableDraft]);
}

function buildControllerResult(options: WorkflowControllerResultOptions): UseWorkflowEditorControllerResult {
  const { core, importActions, draftActions, onSave, onBack } = options;
  return {
    phase: core.phase,
    draft: core.draft,
    autosaveState: core.autosaveState,
    actionError: core.actionError,
    importSessionID: importActions.importSessionID,
    importLoading: importActions.importLoading,
    agentRuntimeCatalog: core.agentRuntimeState.catalog,
    agentRuntimeLoading: core.agentRuntimeState.loading,
    agentRuntimeError: core.agentRuntimeState.error,
    validationErrors: core.validationErrors,
    onChangeImportSessionID: importActions.onChangeImportSessionID,
    onImportFromSession: importActions.onImportFromSession,
    ...draftActions,
    onSave,
    onBack,
  };
}
