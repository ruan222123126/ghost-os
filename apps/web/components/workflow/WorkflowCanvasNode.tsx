'use client';

import type { MouseEvent } from 'react';
import { WorkflowCanvasNodeSummary } from '@/components/workflow/WorkflowCanvasNodeSummary';
import { useWebLocale } from '@/lib/i18n/provider';
import type { WorkflowCanvasNodeDraft, WorkflowEditorKind, WorkflowNodeType } from '@/lib/workflow-editor';

export interface WorkflowNodeMeta {
  id: WorkflowNodeType;
  label: string;
  glyph: string;
}

interface WorkflowCanvasNodeProps {
  editorKind: WorkflowEditorKind;
  node: WorkflowCanvasNodeDraft;
  metadata: WorkflowNodeMeta;
  selected: boolean;
  targetable: boolean;
  connectingSourceNodeID?: string;
  onSelectNode: (nodeID: string) => void;
  onOpenContextMenu: (event: MouseEvent<HTMLElement>, nodeID: string) => void;
  onStartDrag: (event: MouseEvent<HTMLElement>, nodeID: string) => void;
  onClickSourcePort: (event: MouseEvent<HTMLDivElement>, nodeID: string) => void;
  onClickTargetPort: (event: MouseEvent<HTMLDivElement>, nodeID: string) => void;
}

interface WorkflowCanvasNodeInputPortProps {
  nodeID: string;
  targetable: boolean;
  visible: boolean;
  onClickTargetPort: (event: MouseEvent<HTMLDivElement>, nodeID: string) => void;
}

interface WorkflowCanvasNodeOutputPortProps {
  active: boolean;
  nodeID: string;
  visible: boolean;
  onClickSourcePort: (event: MouseEvent<HTMLDivElement>, nodeID: string) => void;
}

interface WorkflowCanvasNodeMouseDownOptions {
  event: MouseEvent<HTMLElement>;
  nodeID: string;
  onStartDrag: (event: MouseEvent<HTMLElement>, nodeID: string) => void;
}

interface WorkflowCanvasNodeClickOptions {
  event: MouseEvent<HTMLElement>;
  nodeID: string;
  onSelectNode: (nodeID: string) => void;
}

const PRIMARY_MOUSE_BUTTON = 0;

export function WorkflowCanvasNode(props: WorkflowCanvasNodeProps) {
  const { locale, copy } = useWebLocale();
  const {
    editorKind,
    node,
    metadata,
    selected,
    targetable,
    connectingSourceNodeID,
    onSelectNode,
    onOpenContextMenu,
    onStartDrag,
    onClickSourcePort,
    onClickTargetPort,
  } = props;
  const nodeID = node.id;
  const headerTitle = resolveNodeHeader(node, metadata.label);

  return (
    <article
      className={`workflow-arch-node ${nodeSelectionClass(selected)}`}
      style={{ transform: `translate(${node.position.x}px, ${node.position.y}px)` }}
      onMouseDown={(event) => handleNodeMouseDown({ event, nodeID, onStartDrag })}
      onClick={(event) => handleNodeClick({ event, nodeID, onSelectNode })}
      onContextMenu={(event) => onOpenContextMenu(event, nodeID)}
    >
      <div className={nodeToplineClass(selected)} />
      <header className="workflow-arch-node-header">
        <span>{headerTitle}</span>
      </header>
      <div className="workflow-arch-node-body">
        <WorkflowCanvasNodeSummary node={node} workflowCopy={copy.workflow} locale={locale} />
      </div>

      <WorkflowCanvasNodeInputPort
        nodeID={nodeID}
        targetable={targetable}
        visible={canShowInputPort(editorKind, node)}
        onClickTargetPort={onClickTargetPort}
      />
      <WorkflowCanvasNodeOutputPort
        active={connectingSourceNodeID === nodeID}
        nodeID={nodeID}
        visible={canShowOutputPort(editorKind, node)}
        onClickSourcePort={onClickSourcePort}
      />
    </article>
  );
}

function WorkflowCanvasNodeInputPort(props: WorkflowCanvasNodeInputPortProps) {
  const { nodeID, targetable, visible, onClickTargetPort } = props;

  if (!visible) {
    return null;
  }

  return (
    <div
      className="workflow-arch-port workflow-arch-port--input"
      onMouseDown={stopMouseEventPropagation}
      onClick={(event) => onClickTargetPort(event, nodeID)}
    >
      <span className={inputPortDotClass(targetable)} />
    </div>
  );
}

function WorkflowCanvasNodeOutputPort(props: WorkflowCanvasNodeOutputPortProps) {
  const { active, nodeID, visible, onClickSourcePort } = props;

  if (!visible) {
    return null;
  }

  return (
    <div
      className="workflow-arch-port workflow-arch-port--output"
      onMouseDown={stopMouseEventPropagation}
      onClick={(event) => onClickSourcePort(event, nodeID)}
    >
      <span className={outputPortDotClass(active)} />
    </div>
  );
}

function handleNodeMouseDown(options: WorkflowCanvasNodeMouseDownOptions): void {
  const { event, nodeID, onStartDrag } = options;

  if (event.button !== PRIMARY_MOUSE_BUTTON) {
    return;
  }

  event.preventDefault();
  event.stopPropagation();
  onStartDrag(event, nodeID);
}

function handleNodeClick(options: WorkflowCanvasNodeClickOptions): void {
  const { event, nodeID, onSelectNode } = options;

  event.stopPropagation();
  onSelectNode(nodeID);
}

function stopMouseEventPropagation(event: MouseEvent<HTMLElement>): void {
  event.stopPropagation();
}

function nodeSelectionClass(selected: boolean): string {
  if (selected) {
    return 'workflow-arch-node--selected';
  }
  return 'workflow-arch-node--idle';
}

function nodeToplineClass(selected: boolean): string {
  if (selected) {
    return 'workflow-arch-node-topline workflow-arch-node-topline--active';
  }
  return 'workflow-arch-node-topline';
}

function inputPortDotClass(targetable: boolean): string {
  if (targetable) {
    return 'workflow-arch-port-dot workflow-arch-port-dot--targetable';
  }
  return 'workflow-arch-port-dot';
}

function outputPortDotClass(active: boolean): string {
  if (active) {
    return 'workflow-arch-port-dot workflow-arch-port-dot--active';
  }
  return 'workflow-arch-port-dot';
}

function resolveNodeHeader(node: WorkflowCanvasNodeDraft, fallback: string): string {
  if (node.type === 'agent' && node.agent?.title?.trim()) {
    return node.agent.title.trim();
  }
  if (node.type === 'group' && node.group?.title?.trim()) {
    return node.group.title.trim();
  }
  return fallback;
}

function canShowInputPort(editorKind: WorkflowEditorKind, node: WorkflowCanvasNodeDraft): boolean {
  if (editorKind === 'orchestration' && node.type === 'agent') {
    return false;
  }
  return node.type !== 'start';
}

function canShowOutputPort(editorKind: WorkflowEditorKind, node: WorkflowCanvasNodeDraft): boolean {
  if (editorKind === 'orchestration' && node.type === 'end') {
    return false;
  }
  return node.type !== 'end';
}
