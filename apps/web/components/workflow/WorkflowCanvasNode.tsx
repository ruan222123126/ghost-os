'use client';

import type { MouseEvent } from 'react';
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
  const selectedClass = selected
    ? 'workflow-arch-node--selected'
    : 'workflow-arch-node--idle';
  const sourceActive = connectingSourceNodeID === node.id;

  const showInputPort = canShowInputPort(editorKind, node);
  const showOutputPort = canShowOutputPort(editorKind, node);
  const headerTitle = resolveNodeHeader(node, metadata.label);

  return (
    <article
      className={`workflow-arch-node ${selectedClass}`}
      style={{ transform: `translate(${node.position.x}px, ${node.position.y}px)` }}
      onMouseDown={(event) => {
        if (event.button !== PRIMARY_MOUSE_BUTTON) {
          return;
        }
        event.preventDefault();
        event.stopPropagation();
        onStartDrag(event, node.id);
      }}
      onClick={(event) => {
        event.stopPropagation();
        onSelectNode(node.id);
      }}
      onContextMenu={(event) => onOpenContextMenu(event, node.id)}
    >
      <div className={`workflow-arch-node-topline ${selected ? 'workflow-arch-node-topline--active' : ''}`} />
      <header className="workflow-arch-node-header">
        <span>{headerTitle}</span>
      </header>
      <div className="workflow-arch-node-body">{renderNodeSummary(node, copy, locale)}</div>

      {showInputPort ? (
        <div
          className="workflow-arch-port workflow-arch-port--input"
          onMouseDown={(event) => event.stopPropagation()}
          onClick={(event) => onClickTargetPort(event, node.id)}
        >
          <span className={targetable ? 'workflow-arch-port-dot workflow-arch-port-dot--targetable' : 'workflow-arch-port-dot'} />
        </div>
      ) : null}

      {showOutputPort ? (
        <div
          className="workflow-arch-port workflow-arch-port--output"
          onMouseDown={(event) => event.stopPropagation()}
          onClick={(event) => onClickSourcePort(event, node.id)}
        >
          <span className={sourceActive ? 'workflow-arch-port-dot workflow-arch-port-dot--active' : 'workflow-arch-port-dot'} />
        </div>
      ) : null}
    </article>
  );
}

function renderNodeSummary(
  node: WorkflowCanvasNodeDraft,
  copy: ReturnType<typeof useWebLocale>['copy'],
  locale: string,
) {
  if (node.type === 'start') {
    return (
      <div className="workflow-arch-summary">
        <p className="workflow-arch-summary-label">{copy.workflow.nodeInputConfiguration}</p>
        <p>{startNodeSummary(locale)}</p>
      </div>
    );
  }
  if (node.type === 'llm') {
    return (
      <div className="workflow-arch-summary">
        <p className="workflow-arch-summary-label">{copy.workflow.nodeInferenceParameters}</p>
        <p>{node.llm?.prompt?.trim() ? copy.workflow.nodePromptDefined : copy.workflow.nodeAwaitingPrompt}</p>
      </div>
    );
  }
  if (node.type === 'agent') {
    const role = node.agent?.message?.trim();
    return (
      <div className="workflow-arch-summary">
        <p className="workflow-arch-summary-label">{copy.workflow.nodeExecutionIdentity}</p>
        {node.agent?.title?.trim() ? <p>{node.agent.title.trim()}</p> : null}
        <p>{role ? copy.workflow.nodeRole(role) : copy.workflow.nodeUndefinedRole}</p>
      </div>
    );
  }
  if (node.type === 'group') {
    return (
      <div className="workflow-arch-summary">
        <p className="workflow-arch-summary-label">{copy.workflow.nodeGroupConfiguration}</p>
        <p>{node.group?.title?.trim() || copy.workflow.nodeGroupPending}</p>
        <p>{copy.workflow.nodeGroupRounds(node.group?.max_rounds ?? 0, node.group?.speaking_mode ?? 'sequential')}</p>
        {node.group?.speaking_mode === 'owner' && node.group.owner_agent_id?.trim()
          ? <p>{copy.workflow.groupOwnerAgent}: {node.group.owner_agent_id.trim()}</p>
          : null}
      </div>
    );
  }
  if (node.type === 'tool') {
    const toolName = node.tool?.tool_name?.trim();
    const argumentCount = Object.keys(node.tool?.arguments ?? {}).length;
    return (
      <div className="workflow-arch-summary">
        <p className="workflow-arch-summary-label">{copy.workflow.nodeToolConfiguration}</p>
        <p>{toolName ? copy.workflow.nodeToolSelected(toolName) : copy.workflow.nodeToolUnset}</p>
        <p>{copy.workflow.nodeToolArguments(argumentCount)}</p>
      </div>
    );
  }
  if (node.type === 'if') {
    const operator = node.if?.operator ?? 'equals';
    const trueNodeID = node.if?.true_node_id?.trim();
    const falseNodeID = node.if?.false_node_id?.trim();
    return (
      <div className="workflow-arch-summary">
        <p className="workflow-arch-summary-label">{copy.workflow.nodeConditionalBranch}</p>
        <p>
          {trueNodeID && falseNodeID
            ? copy.workflow.nodeBranchResolved(operator, trueNodeID, falseNodeID)
            : copy.workflow.nodeBranchPending(operator)}
        </p>
      </div>
    );
  }
  if (node.type === 'loop') {
    const role = node.loop?.role ?? 'start';
    const loopID = node.loop?.loop_id?.trim();
    return (
      <div className="workflow-arch-summary">
        <p className="workflow-arch-summary-label">{copy.workflow.nodeLoopSegment}</p>
        <p>{loopID ? copy.workflow.nodeLoopResolved(role, loopID) : copy.workflow.nodeLoopPending(role)}</p>
      </div>
    );
  }
  if (node.type === 'end') {
    return <p className="workflow-arch-terminus">{copy.workflow.terminus}</p>;
  }
  return (
    <div className="workflow-arch-summary">
      <p className="workflow-arch-summary-label">{copy.workflow.nodeSystemCapabilities}</p>
      <p>{copy.workflow.nodeStandardToolsActive}</p>
    </div>
  );
}

function startNodeSummary(locale: string): string {
  if (locale === 'zh-CN') {
    return '通用变量已关闭';
  }
  return 'General variables disabled';
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
