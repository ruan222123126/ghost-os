'use client';

import { useCallback, type Dispatch, type SetStateAction } from 'react';
import { toErrorMessage } from '@/lib/errors';
import { buildHomeSettingsURL } from '@/lib/settingsQuery';
import type { WebLocale } from '@/lib/i18n/locale';
import type { WebCopy } from '@/lib/i18n/messages';
import {
  addOrchestrationNode,
  buildDefaultOrchestrationNodeSource,
  connectOrchestrationDraftNodes,
  duplicateOrchestrationNode,
  moveOrchestrationNode,
  removeOrchestrationEdge,
  removeOrchestrationNode,
  updateOrchestrationNode,
} from '@/lib/orchestration-editor/draftReducer';
import type { OrchestrationAutosaveSnapshotBuildResult } from '@/lib/orchestration-editor/snapshot';
import {
  type AutosaveController,
  type WorkflowCanvasDraft,
  type WorkflowCanvasNodeDraft,
  type WorkflowCanvasPosition,
  type WorkflowNodeType,
  type WorkflowUpdatePayload,
} from '@/lib/workflow-editor';
import { localizeOrchestrationValidationError } from './orchestrationEditorCopy';

const ORCHESTRATION_SETTINGS_URL = buildHomeSettingsURL('orchestration');

interface OrchestrationEditorRouter {
  push(path: string): void;
}

interface OrchestrationDraftActionOptions {
  locale: WebLocale;
  toolNames: string[];
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>;
}

interface OrchestrationSaveActionOptions {
  copy: WebCopy;
  locale: WebLocale;
  autosaveController: AutosaveController<WorkflowUpdatePayload>;
  snapshotBuild: OrchestrationAutosaveSnapshotBuildResult;
  validationErrors: string[];
  setActionError: Dispatch<SetStateAction<string>>;
}

export function useOrchestrationDraftActions(options: OrchestrationDraftActionOptions) {
  const { locale, toolNames, setDraft } = options;
  return {
    onScheduleChange: (patch: Partial<WorkflowCanvasDraft['schedule']>) =>
      setDraft((state) => ({ ...state, schedule: { ...state.schedule, ...patch } })),
    onAddNode: (type: WorkflowNodeType, position: WorkflowCanvasPosition) => {
      setDraft((state) => addOrchestrationNode(state, {
        type,
        position,
        source: buildDefaultOrchestrationNodeSource({
          draft: state,
          type,
          toolNames,
          locale,
        }),
      }));
    },
    onSelectNode: (nodeID?: string) => setDraft((state) => ({ ...state, selectedNodeId: nodeID })),
    onMoveNode: (nodeID: string, position: WorkflowCanvasPosition) =>
      setDraft((state) => moveOrchestrationNode(state, nodeID, position)),
    onConnectNodes: (sourceNodeID: string, targetNodeID: string) =>
      setDraft((state) => connectOrchestrationDraftNodes(state, sourceNodeID, targetNodeID)),
    onDeleteEdge: (edgeID: string) => setDraft((state) => removeOrchestrationEdge(state, edgeID)),
    onDuplicateNode: (nodeID: string) => setDraft((state) => duplicateOrchestrationNode(state, nodeID)),
    onUpdateNode: (node: WorkflowCanvasNodeDraft) => setDraft((state) => updateOrchestrationNode(state, node)),
    onDeleteNode: (nodeID: string) => setDraft((state) => removeOrchestrationNode(state, nodeID)),
  };
}

export function useOrchestrationSaveAction(options: OrchestrationSaveActionOptions) {
  const {
    copy,
    locale,
    autosaveController,
    snapshotBuild,
    validationErrors,
    setActionError,
  } = options;

  return useCallback(async () => {
    const blockReason = validationErrors[0]?.trim() || snapshotBuild.errorMessage?.trim();
    if (blockReason || !snapshotBuild.snapshot) {
      const localized = localizeOrchestrationValidationError(blockReason ?? 'Validation failed, not saved', locale);
      autosaveController.markBlocked(localized);
      setActionError(localized);
      return;
    }
    try {
      await autosaveController.flush(snapshotBuild.snapshot, { force: true });
      setActionError('');
    } catch (error) {
      setActionError(toErrorMessage(error, copy.system.failedToSaveOrchestration));
    }
  }, [autosaveController, copy.system.failedToSaveOrchestration, locale, setActionError, snapshotBuild, validationErrors]);
}

export function useOrchestrationBackAction(router: OrchestrationEditorRouter) {
  return useCallback(() => {
    router.push(ORCHESTRATION_SETTINGS_URL);
  }, [router]);
}
