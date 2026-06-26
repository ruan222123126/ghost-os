'use client';

import {
  WorkflowCanvasNodeContextMenu,
} from '@/components/workflow/WorkflowCanvasNodeContextMenu';
import { WorkflowCanvasStageNodes } from '@/components/workflow/WorkflowCanvasStageNodes';
import { renderEdge } from '@/components/workflow/workflowCanvasStageHelpers';
import { useWorkflowCanvasStageInteractions } from '@/hooks/workflow/useWorkflowCanvasStageInteractions';
import { localizeWorkflowValidationError } from '@/lib/i18n/workflowValidation';
import type { WebLocale } from '@/lib/i18n/locale';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import { useWebLocale } from '@/lib/i18n/provider';
import type {
  WorkflowCanvasDraft,
  WorkflowEditorKind,
  WorkflowCanvasPosition,
} from '@/lib/workflow-editor';

const STAGE_VALIDATION_ERROR_LIMIT = 3;

interface WorkflowCanvasStageProps {
  editorKind: WorkflowEditorKind;
  draft: WorkflowCanvasDraft;
  actionError: string;
  validationErrors: string[];
  workflowCopy: WorkflowCopy;
  localizeValidationError?: (message: string, locale: WebLocale) => string;
  onSelectNode: (nodeID?: string) => void;
  onMoveNode: (nodeID: string, position: WorkflowCanvasPosition) => void;
  onConnectNodes: (sourceNodeID: string, targetNodeID: string) => void;
  onDeleteEdge: (edgeID: string) => void;
  onDuplicateNode: (nodeID: string) => void;
  onDeleteNode: (nodeID: string) => void;
}

export function WorkflowCanvasStage(props: WorkflowCanvasStageProps) {
  const { copy, locale } = useWebLocale();
  const localizeValidationError = props.localizeValidationError ?? localizeWorkflowValidationError;

  const interactions = useWorkflowCanvasStageInteractions({
    draft: props.draft,
    onMoveNode: props.onMoveNode,
    onSelectNode: props.onSelectNode,
  });

  return (
    <section
      ref={interactions.canvasRef}
      className={interactions.canvasClassName}
      onMouseDown={interactions.onCanvasMouseDown}
      onMouseMove={interactions.onCanvasMouseMove}
      onMouseUp={interactions.onCanvasMouseUp}
      onMouseLeave={interactions.onCanvasMouseLeave}
      onWheel={interactions.onCanvasWheel}
      onClick={interactions.onCanvasClick}
    >
      <canvas ref={interactions.backgroundCanvasRef} className="workflow-arch-background-canvas" />
      <WorkflowCanvasStageWorld
        editorKind={props.editorKind}
        interactions={interactions}
        workflowCopy={props.workflowCopy}
        onConnectNodes={props.onConnectNodes}
        onDeleteEdge={props.onDeleteEdge}
        onSelectNode={props.onSelectNode}
      />
      <WorkflowCanvasNodeContextMenu
        menu={interactions.nodeContextMenu}
        onClose={interactions.closeContextMenu}
        onCopyNode={props.onDuplicateNode}
        onDeleteNode={props.onDeleteNode}
      />
      <WorkflowStageActionError actionError={props.actionError} stageSaveFailed={copy.workflow.stageSaveFailed} />
      <WorkflowStageValidationErrors
        locale={locale}
        validationErrors={props.validationErrors}
        workflowCopy={copy.workflow}
        localizeValidationError={localizeValidationError}
      />
    </section>
  );
}

type WorkflowCanvasStageInteractions = ReturnType<typeof useWorkflowCanvasStageInteractions>;

interface WorkflowCanvasStageWorldProps {
  editorKind: WorkflowEditorKind;
  interactions: WorkflowCanvasStageInteractions;
  workflowCopy: WorkflowCopy;
  onConnectNodes: (sourceNodeID: string, targetNodeID: string) => void;
  onDeleteEdge: (edgeID: string) => void;
  onSelectNode: (nodeID?: string) => void;
}

function WorkflowCanvasStageWorld(props: WorkflowCanvasStageWorldProps) {
  const { editorKind, interactions, workflowCopy, onConnectNodes, onDeleteEdge, onSelectNode } = props;
  return (
    <div className="workflow-arch-world" style={interactions.worldTransform}>
      <svg className="workflow-arch-edge-layer">
        {interactions.renderDraft.edges.map((edge) => renderEdge(edge, interactions.nodeMap, onDeleteEdge))}
      </svg>
      <WorkflowCanvasStageNodes
        editorKind={editorKind}
        draft={interactions.renderDraft}
        workflowCopy={workflowCopy}
        nodeMap={interactions.nodeMap}
        canvasRef={interactions.canvasRef}
        viewport={interactions.viewport}
        connectingSourceNodeID={interactions.connectingSourceNodeID}
        onSelectNode={onSelectNode}
        onOpenContextMenu={interactions.onOpenContextMenu}
        onConnectNodes={onConnectNodes}
        onChangeDragState={interactions.setDragState}
        onChangeConnectingSourceNodeID={interactions.setConnectingSourceNodeID}
      />
    </div>
  );
}

function WorkflowStageActionError(props: { actionError: string; stageSaveFailed: string }) {
  if (!props.actionError) {
    return null;
  }
  return (
    <aside className="workflow-arch-floating-error" role="alert" aria-live="assertive">
      <p>{props.stageSaveFailed}</p>
      <pre>{props.actionError}</pre>
    </aside>
  );
}

interface WorkflowStageValidationErrorsProps {
  locale: WebLocale;
  validationErrors: string[];
  workflowCopy: WorkflowCopy;
  localizeValidationError: (message: string, locale: WebLocale) => string;
}

function WorkflowStageValidationErrors(props: WorkflowStageValidationErrorsProps) {
  const { locale, validationErrors, workflowCopy, localizeValidationError } = props;
  if (validationErrors.length === 0) {
    return null;
  }
  const visibleErrors = validationErrors.slice(0, STAGE_VALIDATION_ERROR_LIMIT);
  const hiddenErrorCount = validationErrors.length - STAGE_VALIDATION_ERROR_LIMIT;

  return (
    <aside className="workflow-arch-floating-validation" role="status" aria-live="polite">
      <p>{workflowCopy.stageValidation}</p>
      <ul>
        {visibleErrors.map((error) => (
          <li key={error}>{localizeValidationError(error, locale)}</li>
        ))}
      </ul>
      <WorkflowStageMoreValidationErrors count={hiddenErrorCount} workflowCopy={workflowCopy} />
    </aside>
  );
}

function WorkflowStageMoreValidationErrors(props: { count: number; workflowCopy: WorkflowCopy }) {
  if (props.count <= 0) {
    return null;
  }
  return <small>{props.workflowCopy.stageMoreErrors(props.count)}</small>;
}
