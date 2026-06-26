'use client';

import { CloseButton } from '@/components/CloseButton';
import { WorkflowCanvasNodeEditorContent } from '@/components/workflow/WorkflowCanvasNodeEditorContent';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import { useWebLocale } from '@/lib/i18n/provider';
import type { PresetPayload } from '@/lib/types';
import type {
  WorkflowAgentRuntimeCatalog,
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowEditorKind,
} from '@/lib/workflow-editor';
import { isProtectedBoundaryNode } from '@/lib/workflow-editor';

interface WorkflowCanvasPropertiesPanelProps {
  editorKind: WorkflowEditorKind;
  draft: WorkflowCanvasDraft;
  selectedNode?: WorkflowCanvasNodeDraft;
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  agentRuntimeLoading: boolean;
  agentRuntimeError: string;
  presets?: PresetPayload[];
  presetLoading: boolean;
  presetError: string;
  onClose: () => void;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
  onDeleteNode: (nodeID: string) => void;
}

interface PropertiesPanelContentProps extends Omit<WorkflowCanvasPropertiesPanelProps, 'selectedNode'> {
  copy: WorkflowCopy;
  selectedNode: WorkflowCanvasNodeDraft;
}

interface PropertiesPanelNodeEditorProps extends Omit<PropertiesPanelContentProps, 'copy' | 'onClose' | 'onDeleteNode'> {}

export function WorkflowCanvasPropertiesPanel(props: WorkflowCanvasPropertiesPanelProps) {
  const { copy } = useWebLocale();
  return (
    <aside className={propertiesPanelClassName(props.selectedNode)}>
      {props.selectedNode ? (
        <PropertiesPanelContent {...props} copy={copy.workflow} selectedNode={props.selectedNode} />
      ) : null}
    </aside>
  );
}

function PropertiesPanelContent(props: PropertiesPanelContentProps) {
  const allowDelete = !isProtectedBoundaryNode(props.selectedNode);

  return (
    <>
      <PropertiesPanelHeader copy={props.copy} selectedNode={props.selectedNode} onClose={props.onClose} />
      <PropertiesPanelTypePill copy={props.copy} selectedNode={props.selectedNode} />
      <PropertiesPanelNodeEditor {...props} />
      <PropertiesPanelDeleteFooter
        allowDelete={allowDelete}
        copy={props.copy}
        selectedNode={props.selectedNode}
        onDeleteNode={props.onDeleteNode}
      />
    </>
  );
}

function PropertiesPanelNodeEditor(props: PropertiesPanelNodeEditorProps) {
  return (
    <section className="workflow-arch-properties-content">
      <WorkflowCanvasNodeEditorContent
        editorKind={props.editorKind}
        draft={props.draft}
        selectedNode={props.selectedNode}
        agentRuntimeCatalog={props.agentRuntimeCatalog}
        agentRuntimeLoading={props.agentRuntimeLoading}
        agentRuntimeError={props.agentRuntimeError}
        presets={props.presets}
        presetLoading={props.presetLoading}
        presetError={props.presetError}
        onUpdateNode={props.onUpdateNode}
      />
    </section>
  );
}

function PropertiesPanelHeader(props: {
  copy: WorkflowCopy;
  selectedNode: WorkflowCanvasNodeDraft;
  onClose: () => void;
}) {
  return (
    <section className="workflow-arch-properties-head">
      <div>
        <h2>{props.copy.propertiesTitle}</h2>
        <p>{props.copy.propertiesUUID}: {props.selectedNode.id}</p>
      </div>
      <CloseButton className="shrink-0" onClick={props.onClose} aria-label={props.copy.closePropertiesAria} />
    </section>
  );
}

function PropertiesPanelTypePill(props: { copy: WorkflowCopy; selectedNode: WorkflowCanvasNodeDraft }) {
  return (
    <section className="workflow-arch-properties-pill">
      <span />
      <strong>{labelOfNodeType(props.selectedNode.type, props.copy)}</strong>
    </section>
  );
}

function PropertiesPanelDeleteFooter(props: {
  allowDelete: boolean;
  copy: WorkflowCopy;
  selectedNode: WorkflowCanvasNodeDraft;
  onDeleteNode: (nodeID: string) => void;
}) {
  if (!props.allowDelete) {
    return null;
  }
  return (
    <section className="workflow-arch-properties-footer">
      <button type="button" className="workflow-arch-danger-button" onClick={() => props.onDeleteNode(props.selectedNode.id)}>
        <span aria-hidden>⌫</span>
        {props.copy.deleteInstance}
      </button>
    </section>
  );
}

function propertiesPanelClassName(selectedNode?: WorkflowCanvasNodeDraft): string {
  return `workflow-arch-properties ${selectedNode ? 'workflow-arch-properties--open' : ''}`;
}

function labelOfNodeType(
  type: WorkflowCanvasNodeDraft['type'],
  copy: WorkflowCopy,
): string {
  const labels: Partial<Record<WorkflowCanvasNodeDraft['type'], string>> = {
    group: copy.labelGroup,
    if: copy.labelIfBranch,
    llm: copy.labelLLMModel,
    loop: copy.labelLoop,
    tool: copy.labelToolUse,
  };
  return labels[type] ?? type.toUpperCase();
}
