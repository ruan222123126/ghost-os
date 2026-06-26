'use client';

import type { MouseEvent } from 'react';
import { WorkflowCanvasNodeSummary } from '@/components/workflow/WorkflowCanvasNodeSummary';
import type { WebLocale } from '@/lib/i18n/locale';
import { useWebLocale } from '@/lib/i18n/provider';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
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

interface WorkflowCanvasNodeHeaderProps {
  metadata: WorkflowNodeMeta;
  node: WorkflowCanvasNodeDraft;
  selected: boolean;
}

interface WorkflowCanvasNodeBodyProps {
  locale: WebLocale;
  node: WorkflowCanvasNodeDraft;
  workflowCopy: WorkflowCopy;
}

interface WorkflowCanvasNodePortsProps {
  connectingSourceNodeID?: string;
  editorKind: WorkflowEditorKind;
  node: WorkflowCanvasNodeDraft;
  targetable: boolean;
  onClickSourcePort: (event: MouseEvent<HTMLDivElement>, nodeID: string) => void;
  onClickTargetPort: (event: MouseEvent<HTMLDivElement>, nodeID: string) => void;
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
  const nodeID = props.node.id;

  return (
    <article
      className={`workflow-arch-node ${nodeSelectionClass(props.selected)}`}
      style={{ transform: `translate(${props.node.position.x}px, ${props.node.position.y}px)` }}
      onMouseDown={(event) => handleNodeMouseDown({ event, nodeID, onStartDrag: props.onStartDrag })}
      onClick={(event) => handleNodeClick({ event, nodeID, onSelectNode: props.onSelectNode })}
      onContextMenu={(event) => props.onOpenContextMenu(event, nodeID)}
    >
      <WorkflowCanvasNodeHeader node={props.node} metadata={props.metadata} selected={props.selected} />
      <WorkflowCanvasNodeBody node={props.node} workflowCopy={copy.workflow} locale={locale} />
      <WorkflowCanvasNodePorts {...props} />
    </article>
  );
}

function WorkflowCanvasNodeHeader(props: WorkflowCanvasNodeHeaderProps) {
  const { metadata, node, selected } = props;

  return (
    <>
      <div className={nodeToplineClass(selected)} />
      <header className="workflow-arch-node-header">
        <span>{resolveNodeHeader(node, metadata.label)}</span>
      </header>
    </>
  );
}

function WorkflowCanvasNodeBody(props: WorkflowCanvasNodeBodyProps) {
  const { locale, node, workflowCopy } = props;

  return (
    <div className="workflow-arch-node-body">
      <WorkflowCanvasNodeSummary node={node} workflowCopy={workflowCopy} locale={locale} />
    </div>
  );
}

function WorkflowCanvasNodePorts(props: WorkflowCanvasNodePortsProps) {
  const { connectingSourceNodeID, editorKind, node, targetable, onClickSourcePort, onClickTargetPort } = props;

  return (
    <>
      <WorkflowCanvasNodeInputPort
        nodeID={node.id}
        targetable={targetable}
        visible={canShowInputPort(editorKind, node)}
        onClickTargetPort={onClickTargetPort}
      />
      <WorkflowCanvasNodeOutputPort
        active={connectingSourceNodeID === node.id}
        nodeID={node.id}
        visible={canShowOutputPort(editorKind, node)}
        onClickSourcePort={onClickSourcePort}
      />
    </>
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
  return resolveNodeTitle(node) ?? fallback;
}

function resolveNodeTitle(node: WorkflowCanvasNodeDraft): string | undefined {
  if (node.type === 'agent') {
    return trimmedValue(node.agent?.title);
  }
  if (node.type === 'group') {
    return trimmedValue(node.group?.title);
  }
  return undefined;
}

function trimmedValue(value: string | undefined): string | undefined {
  const trimmed = value?.trim();
  return trimmed || undefined;
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
