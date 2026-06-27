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
  const ownerID = ownerMembershipID(group);
  if (!ownerID) {
    return;
  }
  if (!isGroupMember(ownerID, members)) {
    errors.push(`orchestration group node "${group.id}" owner_agent_id "${ownerID}" must be an existing member`);
  }
}

function ownerMembershipID(group: WorkflowCanvasNodeDraft): string {
  if (!isOwnerSpeakingMode(group)) {
    return '';
  }
  return group.group?.owner_agent_id?.trim() ?? '';
}

function isOwnerSpeakingMode(group: WorkflowCanvasNodeDraft): boolean {
  return group.group?.speaking_mode === 'owner';
}

function isGroupMember(agentID: string, members: string[]): boolean {
  return members.includes(agentID);
}
