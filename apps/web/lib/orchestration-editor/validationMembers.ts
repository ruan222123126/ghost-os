import type { WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

export function validateOrchestrationMembers(
  groups: WorkflowCanvasNodeDraft[],
  groupMembers: Map<string, string[]>,
): string[] {
  const errors: string[] = [];
  for (const group of groups) {
    validateGroupMembership(group, groupMembers, errors);
  }
  return errors;
}

function validateGroupMembership(
  group: WorkflowCanvasNodeDraft,
  groupMembers: Map<string, string[]>,
  errors: string[],
): void {
  const members = groupMembers.get(group.id) ?? [];
  if (members.length === 0) {
    errors.push(`orchestration group node "${group.id}" requires at least one member`);
    return;
  }
  validateOwnerMembership(group, members, errors);
}

function validateOwnerMembership(
  group: WorkflowCanvasNodeDraft,
  members: string[],
  errors: string[],
): void {
  if (group.group?.speaking_mode !== 'owner') {
    return;
  }
  const ownerID = group.group.owner_agent_id?.trim() ?? '';
  if (!ownerID) {
    return;
  }
  if (!members.includes(ownerID)) {
    errors.push(`orchestration group node "${group.id}" owner_agent_id "${ownerID}" must be an existing member`);
  }
}
