'use client';

import {
  useMemo,
  useRef,
  useState,
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
} from 'react';
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

interface WorkflowEditorControllerActions {
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

interface WorkflowEditorCoreState {
  actionError: string;
  agentNormalizationEnabled: boolean;
  autosaveState: AutosaveState;
  draft: WorkflowCanvasDraft;
  phase: WorkflowEditorPhase;
  runtimeRef: MutableRefObject<WorkflowEditorControllerRuntime>;
  setActionError: Dispatch<SetStateAction<string>>;
  setAgentNormalizationEnabled: Dispatch<SetStateAction<boolean>>;
  setAutosaveState: Dispatch<SetStateAction<AutosaveState>>;
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>;
  setPhase: Dispatch<SetStateAction<WorkflowEditorPhase>>;
}

type WorkflowPersistSnapshotRef = MutableRefObject<
  (snapshot: AutosaveSnapshot<WorkflowUpdatePayload>) => Promise<void>
>;

interface WorkflowEditorCoreLifecycleOptions {
  autosaveController: ReturnType<typeof useAutosaveController>;
  derived: ReturnType<typeof useWorkflowEditorDerived>;
  options: UseWorkflowEditorControllerOptions;
  state: WorkflowEditorCoreState;
}

export function useWorkflowEditorController(
  options: UseWorkflowEditorControllerOptions,
): UseWorkflowEditorControllerResult {
  const router = useRouter();
  const core = useWorkflowEditorCore(options, router);
  const actions = useWorkflowEditorControllerActions(options, core, router);

  return buildControllerResult({
    core,
    ...actions,
  });
}

function useWorkflowEditorControllerActions(
  options: UseWorkflowEditorControllerOptions,
  core: ReturnType<typeof useWorkflowEditorCore>,
  router: ReturnType<typeof useRouter>,
): WorkflowEditorControllerActions {
  return {
    importActions: useControllerImportActions(options, core),
    draftActions: useControllerDraftActions(core),
    onSave: useControllerSaveAction(options, core),
    onBack: useWorkflowEditorBackAction(router),
  };
}

function useControllerImportActions(
  options: UseWorkflowEditorControllerOptions,
  core: ReturnType<typeof useWorkflowEditorCore>,
) {
  return useWorkflowImportActions({
    copy: options.copy,
    agentRuntimeReady: core.agentRuntimeReady,
    enabledToolNames: core.enabledToolNames,
    setDraft: core.setDraft,
    setActionError: core.setActionError,
    setAgentNormalizationEnabled: core.setAgentNormalizationEnabled,
  });
}

function useControllerDraftActions(core: ReturnType<typeof useWorkflowEditorCore>) {
  return useWorkflowDraftActions({
    draft: core.draft,
    agentRuntimeReady: core.agentRuntimeReady,
    enabledToolNames: core.enabledToolNames,
    setDraft: core.setDraft,
    setAgentNormalizationEnabled: core.setAgentNormalizationEnabled,
  });
}

function useControllerSaveAction(
  options: UseWorkflowEditorControllerOptions,
  core: ReturnType<typeof useWorkflowEditorCore>,
) {
  return useWorkflowSaveAction({
    draft: core.draft,
    copy: options.copy,
    agentRuntimeReady: core.agentRuntimeReady,
    enabledToolNames: core.enabledToolNames,
    autosaveController: core.autosaveController,
    validationErrors: core.validationErrors,
    setActionError: core.setActionError,
    setAgentNormalizationEnabled: core.setAgentNormalizationEnabled,
  });
}

function useWorkflowEditorCore(
  options: UseWorkflowEditorControllerOptions,
  router: ReturnType<typeof useRouter>,
) {
  const state = useWorkflowEditorCoreState(options.mode, options.taskID);
  const agentRuntimeState = useWorkflowAgentRuntimeCatalog();
  const derived = useWorkflowEditorDerived(state.draft, state.agentNormalizationEnabled, agentRuntimeState);
  const persistSnapshotRef = useWorkflowPersistSnapshotRef(state.runtimeRef, state.setDraft, router);
  const autosaveController = useAutosaveController(persistSnapshotRef, state.setAutosaveState);
  useWorkflowEditorCoreLifecycle({ options, state, derived, autosaveController });

  return {
    phase: state.phase,
    draft: state.draft,
    autosaveState: state.autosaveState,
    actionError: state.actionError,
    agentRuntimeState,
    autosaveController,
    setDraft: state.setDraft,
    setActionError: state.setActionError,
    setAgentNormalizationEnabled: state.setAgentNormalizationEnabled,
    ...derived,
  };
}

function useWorkflowEditorCoreLifecycle(input: WorkflowEditorCoreLifecycleOptions): void {
  const { autosaveController, derived, options, state } = input;
  useWorkflowEditorLifecycle({
    mode: options.mode,
    taskID: options.taskID,
    copy: options.copy,
    autosaveController,
    runtimeRef: state.runtimeRef,
    draft: state.draft,
    currentSnapshot: derived.snapshotBuild.snapshot,
    isLoading: state.phase === 'loading',
    snapshotErrorMessage: derived.snapshotBuild.errorMessage,
    validationErrors: derived.validationErrors,
    agentNormalizationEnabled: state.agentNormalizationEnabled,
    agentRuntimeReady: derived.agentRuntimeReady,
    enabledToolNames: derived.enabledToolNames,
    setDraft: state.setDraft,
    setActionError: state.setActionError,
    setPhase: state.setPhase,
  });
}

function useWorkflowEditorCoreState(
  mode: UseWorkflowEditorControllerOptions['mode'],
  taskID: string | undefined,
): WorkflowEditorCoreState {
  const [draft, setDraft] = useState<WorkflowCanvasDraft>(() => createEmptyWorkflowDraft(mode));
  const [phase, setPhase] = useState<WorkflowEditorPhase>(mode === 'edit' ? 'loading' : 'ready');
  const [autosaveState, setAutosaveState] = useState<AutosaveState>(() => ({
    phase: 'idle',
    message: 'Autosave idle',
    updatedAt: Date.now(),
  }));
  const [actionError, setActionError] = useState('');
  const [agentNormalizationEnabled, setAgentNormalizationEnabled] = useState(false);
  const runtimeRef = useRef<WorkflowEditorControllerRuntime>({
    taskID,
    routeReplaced: mode === 'edit',
    baselineSeeded: false,
  });

  return {
    actionError,
    agentNormalizationEnabled,
    autosaveState,
    draft,
    phase,
    runtimeRef,
    setActionError,
    setAgentNormalizationEnabled,
    setAutosaveState,
    setDraft,
    setPhase,
  };
}

function useWorkflowPersistSnapshotRef(
  runtimeRef: MutableRefObject<WorkflowEditorControllerRuntime>,
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>,
  router: ReturnType<typeof useRouter>,
): WorkflowPersistSnapshotRef {
  const persistSnapshotRef = useRef<(snapshot: AutosaveSnapshot<WorkflowUpdatePayload>) => Promise<void>>(
    async () => undefined,
  );

  persistSnapshotRef.current = async (snapshot) =>
    persistWorkflowSnapshot({ snapshot, runtimeRef, setDraft, router });

  return persistSnapshotRef;
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
