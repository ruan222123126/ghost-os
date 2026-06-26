'use client';

import { useMemo, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useWorkflowAgentRuntimeCatalog } from '@/hooks/workflow/useWorkflowAgentRuntimeCatalog';
import { createEmptyOrchestrationDraft } from '@/lib/orchestration-editor/draft';
import { buildOrchestrationAutosaveSnapshot } from '@/lib/orchestration-editor/snapshot';
import { validateOrchestrationDraft } from '@/lib/orchestration-editor/validation';
import {
  availableWorkflowAgentToolNames,
  type AutosaveState,
  type WorkflowAgentRuntimeCatalog,
  type WorkflowCanvasDraft,
} from '@/lib/workflow-editor';
import {
  buildOrchestrationWorkflowCopy,
  localizeOrchestrationValidationError,
} from './orchestrationEditorCopy';
import { useOrchestrationAutosaveController } from './orchestrationEditorControllerInternals';
import type {
  OrchestrationEditorPhase,
  UseOrchestrationEditorControllerOptions,
  UseOrchestrationEditorControllerResult,
} from './orchestrationEditorControllerTypes';
import {
  useOrchestrationBackAction,
  useOrchestrationDraftActions,
  useOrchestrationSaveAction,
} from './useOrchestrationEditorActions';
import { useOrchestrationEditorLifecycle } from './useOrchestrationEditorLifecycle';
import { useOrchestrationPresetCatalog } from './useOrchestrationPresetCatalog';

interface OrchestrationControllerResultOptions {
  core: ReturnType<typeof useOrchestrationEditorCore>;
  draftActions: ReturnType<typeof useOrchestrationDraftActions>;
  onSave: () => Promise<void>;
  onBack: () => void;
}

interface OrchestrationDerivedOptions {
  draft: WorkflowCanvasDraft;
  copy: UseOrchestrationEditorControllerOptions['copy'];
  locale: UseOrchestrationEditorControllerOptions['locale'];
  agentRuntimeState: ReturnType<typeof useWorkflowAgentRuntimeCatalog>;
  presetState: ReturnType<typeof useOrchestrationPresetCatalog>;
}

export function useOrchestrationEditorController(
  options: UseOrchestrationEditorControllerOptions,
): UseOrchestrationEditorControllerResult {
  const router = useRouter();
  const core = useOrchestrationEditorCore(options);
  const draftActions = useOrchestrationDraftActions({
    locale: options.locale,
    toolNames: core.derived.toolNames,
    setDraft: core.setDraft,
  });
  const handleSave = useOrchestrationSaveAction({
    copy: options.copy,
    locale: options.locale,
    autosaveController: core.autosaveController,
    snapshotBuild: core.derived.snapshotBuild,
    validationErrors: core.derived.validationErrors,
    setActionError: core.setActionError,
  });
  const handleBack = useOrchestrationBackAction(router);
  return buildControllerResult({
    core,
    draftActions,
    onSave: handleSave,
    onBack: handleBack,
  });
}

function useOrchestrationEditorCore(options: UseOrchestrationEditorControllerOptions) {
  const { orchestrationID, copy, locale } = options;
  const [draft, setDraft] = useState(() => createEmptyOrchestrationDraft('edit'));
  const [phase, setPhase] = useState<OrchestrationEditorPhase>('loading');
  const [autosaveState, setAutosaveState] = useState<AutosaveState>({ phase: 'idle', message: 'Autosave idle', updatedAt: Date.now() });
  const [actionError, setActionError] = useState('');
  const draftRef = useRef(draft);
  const agentRuntimeState = useWorkflowAgentRuntimeCatalog();
  const presetState = useOrchestrationPresetCatalog(copy.system.failedToLoadPresets);
  const derived = useOrchestrationEditorDerived({
    draft,
    copy,
    locale,
    agentRuntimeState,
    presetState,
  });
  const autosaveController = useOrchestrationAutosaveController({ orchestrationID, draftRef, setAutosaveState });

  useOrchestrationEditorLifecycle({
    orchestrationID,
    loadErrorMessage: copy.system.failedToLoadOrchestration,
    autosaveController,
    draftRef,
    draft,
    phase,
    currentSnapshot: derived.snapshotBuild.snapshot,
    snapshotErrorMessage: derived.snapshotBuild.errorMessage,
    validationErrors: derived.validationErrors,
    setActionError,
    setDraft,
    setPhase,
  });

  return {
    phase,
    draft,
    autosaveState,
    actionError,
    agentRuntimeState,
    presetState,
    derived,
    autosaveController,
    setDraft,
    setActionError,
  };
}

function useOrchestrationEditorDerived(options: OrchestrationDerivedOptions) {
  const { draft, copy, locale, agentRuntimeState, presetState } = options;
  const toolNames = useOrchestrationToolNames(agentRuntimeState.catalog);
  const validationErrors = useOrchestrationValidationErrors({
    draft,
    agentRuntimeState,
    presetState,
  });
  const snapshotBuild = useMemo(() => buildOrchestrationAutosaveSnapshot(draft), [draft]);
  const workflowCopy = useMemo(() => buildOrchestrationWorkflowCopy(copy.workflow, locale), [copy.workflow, locale]);

  return { toolNames, validationErrors, snapshotBuild, workflowCopy };
}

function useOrchestrationToolNames(catalog: WorkflowAgentRuntimeCatalog | undefined): string[] {
  return useMemo(
    () => availableWorkflowAgentToolNames(catalog?.tools ?? []),
    [catalog?.tools],
  );
}

function useOrchestrationValidationErrors(options: {
  draft: WorkflowCanvasDraft;
  agentRuntimeState: ReturnType<typeof useWorkflowAgentRuntimeCatalog>;
  presetState: ReturnType<typeof useOrchestrationPresetCatalog>;
}): string[] {
  const { draft, agentRuntimeState, presetState } = options;
  const validation = useMemo(() => validateOrchestrationDraft(draft, {
    agentRuntimeCatalog: agentRuntimeState.catalog,
    presets: presetState.loading ? undefined : presetState.presets,
  }), [agentRuntimeState.catalog, draft, presetState.loading, presetState.presets]);
  return validation.errors;
}

function buildControllerResult(options: OrchestrationControllerResultOptions): UseOrchestrationEditorControllerResult {
  const { core, draftActions, onSave, onBack } = options;
  return {
    phase: core.phase,
    draft: core.draft,
    autosaveState: core.autosaveState,
    actionError: core.actionError,
    validationErrors: core.derived.validationErrors,
    agentRuntimeCatalog: core.agentRuntimeState.catalog,
    agentRuntimeLoading: core.agentRuntimeState.loading,
    agentRuntimeError: core.agentRuntimeState.error,
    presets: core.presetState.presets,
    presetLoading: core.presetState.loading,
    presetError: core.presetState.error,
    workflowCopy: core.derived.workflowCopy,
    localizeValidationError: localizeOrchestrationValidationError,
    ...draftActions,
    onSave,
    onBack,
  };
}
