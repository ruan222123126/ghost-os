import type { ReactNode } from 'react';
import type { WebLocale } from '@/lib/i18n/locale';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import type { WorkflowCanvasNodeDraft } from '@/lib/workflow-editor';

interface WorkflowCanvasNodeSummaryProps {
  node: WorkflowCanvasNodeDraft;
  workflowCopy: WorkflowCopy;
  locale: WebLocale;
}

interface SummaryBlockProps {
  children: ReactNode;
  label: string;
}

interface NodeSummaryProps {
  copy: WorkflowCopy;
  node: WorkflowCanvasNodeDraft;
}

export function WorkflowCanvasNodeSummary(props: WorkflowCanvasNodeSummaryProps) {
  const { node, workflowCopy, locale } = props;

  switch (node.type) {
    case 'start':
      return <StartNodeSummary copy={workflowCopy} locale={locale} />;
    case 'llm':
      return <LLMNodeSummary copy={workflowCopy} node={node} />;
    case 'agent':
      return <AgentNodeSummary copy={workflowCopy} node={node} />;
    case 'group':
      return <GroupNodeSummary copy={workflowCopy} node={node} />;
    case 'tool':
      return <ToolNodeSummary copy={workflowCopy} node={node} />;
    case 'if':
      return <IfNodeSummary copy={workflowCopy} node={node} />;
    case 'loop':
      return <LoopNodeSummary copy={workflowCopy} node={node} />;
    case 'end':
      return <p className="workflow-arch-terminus">{workflowCopy.terminus}</p>;
    default:
      return <DefaultNodeSummary copy={workflowCopy} />;
  }
}

function SummaryBlock(props: SummaryBlockProps) {
  const { children, label } = props;

  return (
    <div className="workflow-arch-summary">
      <p className="workflow-arch-summary-label">{label}</p>
      {children}
    </div>
  );
}

function StartNodeSummary(props: { copy: WorkflowCopy; locale: WebLocale }) {
  const { copy, locale } = props;

  return (
    <SummaryBlock label={copy.nodeInputConfiguration}>
      <p>{startNodeSummary(locale)}</p>
    </SummaryBlock>
  );
}

function LLMNodeSummary(props: NodeSummaryProps) {
  const { copy, node } = props;

  return (
    <SummaryBlock label={copy.nodeInferenceParameters}>
      <p>{node.llm?.prompt?.trim() ? copy.nodePromptDefined : copy.nodeAwaitingPrompt}</p>
    </SummaryBlock>
  );
}

function AgentNodeSummary(props: NodeSummaryProps) {
  const { copy, node } = props;
  const role = node.agent?.message?.trim();

  return (
    <SummaryBlock label={copy.nodeExecutionIdentity}>
      {node.agent?.title?.trim() ? <p>{node.agent.title.trim()}</p> : null}
      <p>{role ? copy.nodeRole(role) : copy.nodeUndefinedRole}</p>
    </SummaryBlock>
  );
}

function GroupNodeSummary(props: NodeSummaryProps) {
  const { copy, node } = props;

  return (
    <SummaryBlock label={copy.nodeGroupConfiguration}>
      <p>{node.group?.title?.trim() || copy.nodeGroupPending}</p>
      <p>{copy.nodeGroupRounds(node.group?.max_rounds ?? 0, node.group?.speaking_mode ?? 'sequential')}</p>
    </SummaryBlock>
  );
}

function ToolNodeSummary(props: NodeSummaryProps) {
  const { copy, node } = props;
  const toolName = node.tool?.tool_name?.trim();
  const argumentCount = Object.keys(node.tool?.arguments ?? {}).length;

  return (
    <SummaryBlock label={copy.nodeToolConfiguration}>
      <p>{toolName ? copy.nodeToolSelected(toolName) : copy.nodeToolUnset}</p>
      <p>{copy.nodeToolArguments(argumentCount)}</p>
    </SummaryBlock>
  );
}

function IfNodeSummary(props: NodeSummaryProps) {
  const { copy, node } = props;

  return (
    <SummaryBlock label={copy.nodeConditionalBranch}>
      <p>{conditionalBranchSummary(copy, node)}</p>
    </SummaryBlock>
  );
}

function LoopNodeSummary(props: NodeSummaryProps) {
  const { copy, node } = props;
  const role = node.loop?.role ?? 'start';
  const loopID = node.loop?.loop_id?.trim();

  return (
    <SummaryBlock label={copy.nodeLoopSegment}>
      <p>{loopID ? copy.nodeLoopResolved(role, loopID) : copy.nodeLoopPending(role)}</p>
    </SummaryBlock>
  );
}

function DefaultNodeSummary(props: { copy: WorkflowCopy }) {
  const { copy } = props;

  return (
    <SummaryBlock label={copy.nodeSystemCapabilities}>
      <p>{copy.nodeStandardToolsActive}</p>
    </SummaryBlock>
  );
}

function startNodeSummary(locale: WebLocale): string {
  if (locale === 'zh-CN') {
    return '通用变量已关闭';
  }
  return 'General variables disabled';
}

function conditionalBranchSummary(copy: WorkflowCopy, node: WorkflowCanvasNodeDraft): string {
  const ifConfig = node.if;

  if (!ifConfig) {
    return copy.nodeBranchPending('equals');
  }

  const operator = ifConfig.operator;
  const trueNodeID = trimmedText(ifConfig.true_node_id);
  const falseNodeID = trimmedText(ifConfig.false_node_id);

  if (!hasResolvedBranch(trueNodeID, falseNodeID)) {
    return copy.nodeBranchPending(operator);
  }

  return copy.nodeBranchResolved(operator, trueNodeID, falseNodeID);
}

function trimmedText(value: string | undefined): string {
  if (value === undefined) {
    return '';
  }
  return value.trim();
}

function hasResolvedBranch(trueNodeID: string, falseNodeID: string): boolean {
  if (!trueNodeID) {
    return false;
  }
  return Boolean(falseNodeID);
}
