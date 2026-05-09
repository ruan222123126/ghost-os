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
  const {
    editorKind,
    draft,
    actionError,
    validationErrors,
    workflowCopy,
    localizeValidationError = localizeWorkflowValidationError,
    onSelectNode,
    onMoveNode,
    onConnectNodes,
    onDeleteEdge,
    onDuplicateNode,
    onDeleteNode,
  } = props;

  const interactions = useWorkflowCanvasStageInteractions({
    draft,
    onMoveNode,
    onSelectNode,
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
      <WorkflowCanvasNodeContextMenu
        menu={interactions.nodeContextMenu}
        onClose={interactions.closeContextMenu}
        onCopyNode={onDuplicateNode}
        onDeleteNode={onDeleteNode}
      />
      {actionError ? (
        <aside className="workflow-arch-floating-error" role="alert" aria-live="assertive">
          <p>{copy.workflow.stageSaveFailed}</p>
          <pre>{actionError}</pre>
        </aside>
      ) : null}
      {validationErrors.length > 0 ? (
        <aside className="workflow-arch-floating-validation" role="status" aria-live="polite">
          <p>{copy.workflow.stageValidation}</p>
          <ul>
            {validationErrors.slice(0, 3).map((error) => (
              <li key={error}>{localizeValidationError(error, locale)}</li>
            ))}
          </ul>
          {validationErrors.length > 3 ? (
            <small>{copy.workflow.stageMoreErrors(validationErrors.length - 3)}</small>
          ) : null}
        </aside>
      ) : null}
    </section>
  );
}
