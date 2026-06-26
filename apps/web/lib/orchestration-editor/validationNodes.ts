import type { WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

const SPEAKING_MODES = ['sequential', 'parallel', 'owner'] as const;
const LEGACY_BOUNDARY_ERROR = 'orchestration node type "%s" is no longer supported';

type OrchestrationSpeakingMode = (typeof SPEAKING_MODES)[number];

export function validateOrchestrationNode(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (node.type === 'group') {
    validateGroupNode(node, errors);
    return;
  }
  if (node.type === 'agent') {
    validateAgentNode(node, errors);
    return;
  }
  validateNonOrchestrationNode(node, errors);
}

function validateGroupNode(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (!hasGroupTitle(node)) {
    errors.push(`orchestration group node "${node.id}" requires title`);
  }
  if (!hasSupportedSpeakingMode(node)) {
    errors.push(`orchestration group node "${node.id}" requires speaking_mode sequential|parallel|owner`);
  }
  if (!hasPositiveMaxRounds(node)) {
    errors.push(`orchestration group node "${node.id}" requires max_rounds > 0`);
  }
  if (requiresOwnerAgentID(node)) {
    errors.push(`orchestration group node "${node.id}" requires owner_agent_id in owner mode`);
  }
}

function hasGroupTitle(node: WorkflowCanvasNodeDraft): boolean {
  return Boolean(node.group?.title?.trim());
}

function hasSupportedSpeakingMode(node: WorkflowCanvasNodeDraft): boolean {
  const speakingMode = node.group?.speaking_mode ?? '';
  return SPEAKING_MODES.includes(speakingMode as OrchestrationSpeakingMode);
}

function hasPositiveMaxRounds(node: WorkflowCanvasNodeDraft): boolean {
  return Number.isInteger(node.group?.max_rounds) && (node.group?.max_rounds ?? 0) >= 1;
}

function requiresOwnerAgentID(node: WorkflowCanvasNodeDraft): boolean {
  return node.group?.speaking_mode === 'owner' && !node.group?.owner_agent_id?.trim();
}

function validateAgentNode(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (!hasAgentTitleAndMessage(node)) {
    errors.push(`orchestration agent node "${node.id}" requires title and message`);
  }
}

function hasAgentTitleAndMessage(node: WorkflowCanvasNodeDraft): boolean {
  return Boolean(node.agent?.title?.trim() && node.agent.message.trim());
}

function validateNonOrchestrationNode(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (node.type === 'start' || node.type === 'end') {
    errors.push(LEGACY_BOUNDARY_ERROR.replace('%s', node.type));
    return;
  }
  errors.push(`unsupported orchestration node type "${String(node.type)}"`);
}
