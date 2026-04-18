'use client';

import type { MouseEvent } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';
import type { WorkflowCanvasNodeDraft, WorkflowNodeType } from '@/lib/workflow-editor';

export interface WorkflowNodeMeta {
  id: WorkflowNodeType;
  label: string;
  glyph: string;
}

interface WorkflowCanvasNodeProps {
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
  const { copy } = useWebLocale();
  const {
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
        <span>{metadata.label}</span>
      </header>
      <div className="workflow-arch-node-body">{renderNodeSummary(node, copy)}</div>

      {node.type !== 'start' ? (
        <div
          className="workflow-arch-port workflow-arch-port--input"
          onMouseDown={(event) => event.stopPropagation()}
          onClick={(event) => onClickTargetPort(event, node.id)}
        >
          <span className={targetable ? 'workflow-arch-port-dot workflow-arch-port-dot--targetable' : 'workflow-arch-port-dot'} />
        </div>
      ) : null}

      {node.type !== 'end' ? (
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

function renderNodeSummary(node: WorkflowCanvasNodeDraft, copy: ReturnType<typeof useWebLocale>['copy']) {
  if (node.type === 'start') {
    const variables = node.start?.inputs?.length ?? 0;
    return (
      <div className="workflow-arch-summary">
        <p className="workflow-arch-summary-label">{copy.workflow.nodeInputConfiguration}</p>
        <p>{copy.workflow.nodeVariables(variables)}</p>
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
        <p>{role ? copy.workflow.nodeRole(role) : copy.workflow.nodeUndefinedRole}</p>
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
