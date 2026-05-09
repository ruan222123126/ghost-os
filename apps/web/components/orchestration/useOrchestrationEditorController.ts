'use client';

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useRouter } from 'next/navigation';
import {
  addNode,
  duplicateNode,
  moveNode,
  removeEdge,
  removeNode,
  updateNode,
} from '@/components/workflow/workflowCanvasState';
import { listPresets } from '@/lib/api/presets/api';
import { useWorkflowAgentRuntimeCatalog } from '@/components/workflow/useWorkflowAgentRuntimeCatalog';
import type { WebLocale } from '@/lib/i18n/locale';
import type { WebCopy } from '@/lib/i18n/messages';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import { buildHomeSettingsURL } from '@/lib/settingsQuery';
import type { PresetPayload } from '@/lib/types';
import { availableWorkflowAgentToolNames, defaultOrchestrationAgentRuntimeOverrides, type AutosaveState, type WorkflowAgentRuntimeCatalog, type WorkflowCanvasDraft, type WorkflowCanvasNodeDraft, type WorkflowCanvasPosition, type WorkflowNodeType } from '@/lib/workflow-editor';
import { DEFAULT_ORCHESTRATION_AGENT_TITLE_PREFIX } from '@/lib/workflow-editor/constants';
import { connectOrchestrationNodes } from '@/lib/orchestration-editor/graph';
import { createEmptyOrchestrationDraft } from '@/lib/orchestration-editor/draft';
import { buildDefaultOrchestrationGroupNode } from '@/lib/orchestration-editor/groupDefaults';
import { validateOrchestrationDraft } from '@/lib/orchestration-editor/validation';
import { migrateLegacyOrchestrations } from '@/lib/orchestration-editor/legacyMigration';
import { toErrorMessage } from '@/lib/errors';
import {
  buildOrchestrationWorkflowCopy,
  localizeOrchestrationValidationError,
} from './orchestrationEditorCopy';
import {
  buildOrchestrationAutosaveSnapshot,
} from './orchestrationEditorDraft';
import {
  useOrchestrationAutosaveController,
  useOrchestrationAutosaveSchedule,
  useOrchestrationBootstrap,
  useOrchestrationDraftPersistence,
} from './orchestrationEditorControllerInternals';

const ORCHESTRATION_SETTINGS_URL = buildHomeSettingsURL('orchestration');

type EditorPhase = 'loading' | 'ready' | 'missing';

interface UseOrchestrationEditorControllerOptions {
  orchestrationID: string;
  copy: WebCopy;
  locale: WebLocale;
}

interface UseOrchestrationEditorControllerResult {
  phase: EditorPhase;
  draft: WorkflowCanvasDraft;
  autosaveState: AutosaveState;
  actionError: string;
  validationErrors: string[];
  importSessionID: string;
  importLoading: boolean;
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  agentRuntimeLoading: boolean;
  agentRuntimeError: string;
  presets: PresetPayload[];
  presetLoading: boolean;
  presetError: string;
  workflowCopy: WorkflowCopy;
  localizeValidationError: (message: string, locale: WebLocale) => string;
  onChangeImportSessionID: (value: string) => void;
  onImportFromSession: () => Promise<void>;
  onScheduleChange: (patch: Partial<WorkflowCanvasDraft['schedule']>) => void;
  onAddNode: (type: WorkflowNodeType, position: WorkflowCanvasPosition) => void;
  onSelectNode: (nodeID?: string) => void;
  onMoveNode: (nodeID: string, position: WorkflowCanvasPosition) => void;
  onConnectNodes: (sourceNodeID: string, targetNodeID: string) => void;
  onDeleteEdge: (edgeID: string) => void;
  onDuplicateNode: (nodeID: string) => void;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
  onDeleteNode: (nodeID: string) => void;
  onSave: () => Promise<void>;
  onBack: () => void;
}

export function useOrchestrationEditorController(
  options: UseOrchestrationEditorControllerOptions,
): UseOrchestrationEditorControllerResult {
  const { orchestrationID, copy, locale } = options;
  const router = useRouter();
  const [draft, setDraft] = useState(() => createEmptyOrchestrationDraft('edit'));
  const [phase, setPhase] = useState<EditorPhase>('loading');
  const [autosaveState, setAutosaveState] = useState<AutosaveState>({ phase: 'idle', message: 'Autosave idle', updatedAt: Date.now() });
  const [actionError, setActionError] = useState('');
  const draftRef = useRef(draft);
  const migrationAttemptedRef = useRef(false);
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

  useEffect(() => {
    draftRef.current = draft;
  }, [draft]);

  useEffect(() => {
    if (migrationAttemptedRef.current) {
      return;
    }
    migrationAttemptedRef.current = true;
    void migrateLegacyOrchestrations().catch(() => undefined);
  }, []);

  useOrchestrationBootstrap({
    orchestrationID,
    loadErrorMessage: copy.system.failedToLoadOrchestration,
    autosaveController,
    draftRef,
    setActionError,
    setDraft,
    setPhase,
  });
  useOrchestrationDraftPersistence({
    draft,
    isLoading: phase === 'loading',
    orchestrationID,
  });
  useOrchestrationAutosaveSchedule({
    autosaveController,
    currentSnapshot: snapshotBuild.snapshot,
    phase,
    snapshotErrorMessage: snapshotBuild.errorMessage,
    validationErrors: validation.errors,
  });

  const handleSave = useCallback(async () => {
    const blockReason = validation.errors[0]?.trim() || snapshotBuild.errorMessage?.trim();
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
  }, [autosaveController, copy.system.failedToSaveOrchestration, locale, snapshotBuild, validation.errors]);

  return {
    phase,
    draft,
    autosaveState,
    actionError,
    validationErrors: validation.errors,
    importSessionID: '',
    importLoading: false,
    agentRuntimeCatalog: agentRuntimeState.catalog,
    agentRuntimeLoading: agentRuntimeState.loading,
    agentRuntimeError: agentRuntimeState.error,
    presets: presetState.presets,
    presetLoading: presetState.loading,
    presetError: presetState.error,
    workflowCopy,
    localizeValidationError: localizeOrchestrationValidationError,
    onChangeImportSessionID: () => undefined,
    onImportFromSession: async () => undefined,
    onScheduleChange: (patch) => setDraft((state) => ({ ...state, schedule: { ...state.schedule, ...patch } })),
    onAddNode: (type, position) => {
      setDraft((state) => addNode(state, type, {
        position,
        source: buildNewNodeSource(state, type, toolNames, locale),
      }));
    },
    onSelectNode: (nodeID) => setDraft((state) => ({ ...state, selectedNodeId: nodeID })),
    onMoveNode: (nodeID, position) => setDraft((state) => moveNode(state, nodeID, position)),
    onConnectNodes: (sourceNodeID, targetNodeID) => setDraft((state) => connectOrchestrationNodes(state, sourceNodeID, targetNodeID)),
    onDeleteEdge: (edgeID) => setDraft((state) => removeEdge(state, edgeID)),
    onDuplicateNode: (nodeID) => setDraft((state) => duplicateNode(state, nodeID)),
    onUpdateNode: (node) => setDraft((state) => updateNode(state, node)),
    onDeleteNode: (nodeID) => setDraft((state) => removeNode(state, nodeID)),
    onSave: handleSave,
    onBack: () => router.push(ORCHESTRATION_SETTINGS_URL),
  };
}

function buildNewNodeSource(
  draft: WorkflowCanvasDraft,
  type: WorkflowNodeType,
  toolNames: string[],
  locale: WebLocale,
): Partial<WorkflowCanvasNodeDraft> | undefined {
  if (type === 'group') {
    return {
      group: buildDefaultOrchestrationGroupNode(countNodesByType(draft, 'group') + 1, locale),
    };
  }
  if (type === 'agent') {
    return {
      agent: {
        title: `${DEFAULT_ORCHESTRATION_AGENT_TITLE_PREFIX} ${countNodesByType(draft, 'agent') + 1}`,
        message: '',
        runtime_overrides: defaultOrchestrationAgentRuntimeOverrides(toolNames),
      },
    };
  }
  return undefined;
}

function countNodesByType(draft: WorkflowCanvasDraft, type: WorkflowNodeType): number {
  return draft.nodes.filter((node) => node.type === type).length;
}

function useOrchestrationPresetCatalog(loadErrorMessage: string) {
  const [presets, setPresets] = useState<PresetPayload[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      setLoading(true);
      setError('');
      try {
        const next = await listPresets();
        if (!cancelled) {
          setPresets(next);
        }
      } catch (cause) {
        if (!cancelled) {
          setError(toErrorMessage(cause, loadErrorMessage));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [loadErrorMessage]);

  return { presets, loading, error };
}
