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

export function useOrchestrationEditorController(
  options: UseOrchestrationEditorControllerOptions,
): UseOrchestrationEditorControllerResult {
  const router = useRouter();
  const core = useOrchestrationEditorCore(options);
  const draftActions = useOrchestrationDraftActions({
    locale: options.locale,
    toolNames: core.toolNames,
    setDraft: core.setDraft,
  });
  const handleSave = useOrchestrationSaveAction({
    copy: options.copy,
    locale: options.locale,
    autosaveController: core.autosaveController,
    snapshotBuild: core.snapshotBuild,
    validationErrors: core.validationErrors,
    setActionError: core.setActionError,
  });
  const handleBack = useOrchestrationBackAction(router);
  return buildControllerResult(core, draftActions, handleSave, handleBack);
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
  const toolNames = useMemo(
    () => availableWorkflowAgentToolNames(agentRuntimeState.catalog?.tools ?? []),
    [agentRuntimeState.catalog?.tools],
  );
  const validation = useMemo(() => validateOrchestrationDraft(draft, {
    agentRuntimeCatalog: agentRuntimeState.catalog,
    presets: presetState.loading ? undefined : presetState.presets,
  }), [agentRuntimeState.catalog, draft, presetState.loading, presetState.presets]);
  const snapshotBuild = useMemo(() => buildOrchestrationAutosaveSnapshot(draft), [draft]);
  const workflowCopy = useMemo(() => buildOrchestrationWorkflowCopy(copy.workflow, locale), [copy.workflow, locale]);
  const autosaveController = useOrchestrationAutosaveController({ orchestrationID, draftRef, setAutosaveState });

  useOrchestrationEditorLifecycle({
    orchestrationID,
    loadErrorMessage: copy.system.failedToLoadOrchestration,
    autosaveController,
    draftRef,
    draft,
    phase,
    currentSnapshot: snapshotBuild.snapshot,
    snapshotErrorMessage: snapshotBuild.errorMessage,
    validationErrors: validation.errors,
    setActionError,
    setDraft,
    setPhase,
  });

  return { phase, draft, autosaveState, actionError, validationErrors: validation.errors, agentRuntimeState, presetState, workflowCopy, toolNames, snapshotBuild, autosaveController, setDraft, setActionError };
}

function buildControllerResult(
  core: ReturnType<typeof useOrchestrationEditorCore>,
  draftActions: ReturnType<typeof useOrchestrationDraftActions>,
  onSave: () => Promise<void>,
  onBack: () => void,
): UseOrchestrationEditorControllerResult {
  return {
    phase: core.phase,
    draft: core.draft,
    autosaveState: core.autosaveState,
    actionError: core.actionError,
    validationErrors: core.validationErrors,
    agentRuntimeCatalog: core.agentRuntimeState.catalog,
    agentRuntimeLoading: core.agentRuntimeState.loading,
    agentRuntimeError: core.agentRuntimeState.error,
    presets: core.presetState.presets,
    presetLoading: core.presetState.loading,
    presetError: core.presetState.error,
    workflowCopy: core.workflowCopy,
    localizeValidationError: localizeOrchestrationValidationError,
    ...draftActions,
    onSave,
    onBack,
  };
}
