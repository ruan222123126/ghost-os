'use client';

import { WorkflowCanvasNodeEditorContent } from '@/components/workflow/WorkflowCanvasNodeEditorContent';
import { useWebLocale } from '@/lib/i18n/provider';
import type {
  WorkflowAgentRuntimeCatalog,
  WorkflowCanvasNodeDraft,
  WorkflowEditorKind,
} from '@/lib/workflow-editor';
import { isProtectedBoundaryNode } from '@/lib/workflow-editor';

interface WorkflowCanvasPropertiesPanelProps {
  editorKind: WorkflowEditorKind;
  selectedNode?: WorkflowCanvasNodeDraft;
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  agentRuntimeLoading: boolean;
  agentRuntimeError: string;
  onClose: () => void;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
  onDeleteNode: (nodeID: string) => void;
}

export function WorkflowCanvasPropertiesPanel(props: WorkflowCanvasPropertiesPanelProps) {
  const { copy } = useWebLocale();
  const {
    agentRuntimeCatalog,
    agentRuntimeError,
    agentRuntimeLoading,
    editorKind,
    selectedNode,
    onClose,
    onUpdateNode,
    onDeleteNode,
  } = props;
  const allowDelete = !isProtectedBoundaryNode(selectedNode);

  return (
    <aside className={`workflow-arch-properties ${selectedNode ? 'workflow-arch-properties--open' : ''}`}>
      {selectedNode ? (
        <>
          <section className="workflow-arch-properties-head">
            <div>
              <h2>{copy.workflow.propertiesTitle}</h2>
              <p>{copy.workflow.propertiesUUID}: {selectedNode.id}</p>
            </div>
            <button type="button" className="workflow-arch-icon-button" onClick={onClose} aria-label={copy.workflow.closePropertiesAria}>
              <span aria-hidden>✕</span>
            </button>
          </section>
          <section className="workflow-arch-properties-pill">
            <span />
            <strong>{labelOfNodeType(selectedNode.type, copy.workflow)}</strong>
          </section>
          <section className="workflow-arch-properties-content">
            <WorkflowCanvasNodeEditorContent
              editorKind={editorKind}
              selectedNode={selectedNode}
              agentRuntimeCatalog={agentRuntimeCatalog}
              agentRuntimeLoading={agentRuntimeLoading}
              agentRuntimeError={agentRuntimeError}
              onUpdateNode={onUpdateNode}
            />
          </section>
          {allowDelete ? (
            <section className="workflow-arch-properties-footer">
              <button type="button" className="workflow-arch-danger-button" onClick={() => onDeleteNode(selectedNode.id)}>
                <span aria-hidden>⌫</span>
                {copy.workflow.deleteInstance}
              </button>
            </section>
          ) : null}
        </>
      ) : null}
    </aside>
  );
}

function labelOfNodeType(
  type: WorkflowCanvasNodeDraft['type'],
  copy: ReturnType<typeof useWebLocale>['copy']['workflow'],
): string {
  if (type === 'tool') {
    return copy.labelToolUse;
  }
  if (type === 'llm') {
    return copy.labelLLMModel;
  }
  if (type === 'if') {
    return copy.labelIfBranch;
  }
  if (type === 'loop') {
    return copy.labelLoop;
  }
  return type.toUpperCase();
}
