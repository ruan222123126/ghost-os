'use client';

import type { WebLocale } from '@/lib/i18n/locale';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import { useWebLocale } from '@/lib/i18n/provider';
import { defaultOrchestrationGroupSharedContext } from '@/lib/orchestration-editor/groupDefaults';
import {
  type WorkflowCanvasDraft,
  type WorkflowCanvasNodeDraft,
  withGroupNode,
} from '@/lib/workflow-editor';

interface WorkflowCanvasGroupNodeEditorProps {
  draft?: WorkflowCanvasDraft;
  selectedNode: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

interface GroupFieldProps {
  copy: WorkflowCopy;
  selectedNode: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

interface GroupMemberOption {
  id: string;
  label: string;
}

type GroupSpeakingMode = NonNullable<WorkflowCanvasNodeDraft['group']>['speaking_mode'];

export function WorkflowCanvasGroupNodeEditor(props: WorkflowCanvasGroupNodeEditorProps) {
  const { copy, locale } = useWebLocale();
  const { draft, selectedNode, onUpdateNode } = props;
  const memberOptions = resolveGroupMemberOptions(draft, selectedNode.id);
  const speakingMode = selectedNode.group?.speaking_mode ?? 'sequential';

  return (
    <div className="workflow-arch-prop-group">
      <GroupTitleField copy={copy.workflow} selectedNode={selectedNode} onUpdateNode={onUpdateNode} />
      <GroupSharedContextField locale={locale} selectedNode={selectedNode} onUpdateNode={onUpdateNode} />
      <GroupSpeakingModeField
        copy={copy.workflow}
        selectedNode={selectedNode}
        speakingMode={speakingMode}
        onUpdateNode={onUpdateNode}
      />
      <GroupOwnerField
        copy={copy.workflow}
        memberOptions={memberOptions}
        selectedNode={selectedNode}
        speakingMode={speakingMode}
        onUpdateNode={onUpdateNode}
      />
      <GroupMaxRoundsField
        copy={copy.workflow}
        selectedNode={selectedNode}
        speakingMode={speakingMode}
        onUpdateNode={onUpdateNode}
      />
    </div>
  );
}

function GroupTitleField(props: GroupFieldProps) {
  const { copy, selectedNode, onUpdateNode } = props;

  return (
    <>
      <label className="workflow-arch-field-label">{copy.groupTitle}</label>
      <input
        type="text"
        value={selectedNode.group?.title ?? ''}
        placeholder={copy.groupTitlePlaceholder}
        onChange={(event) => onUpdateNode(withGroupNode(selectedNode, { title: event.target.value }))}
      />
    </>
  );
}

function GroupSharedContextField(props: {
  locale: WebLocale;
  selectedNode: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}) {
  const { locale, selectedNode, onUpdateNode } = props;

  return (
    <>
      <label className="workflow-arch-field-label">{groupSharedContextLabel(locale)}</label>
      <textarea
        rows={7}
        value={selectedNode.group?.shared_context ?? ''}
        placeholder={defaultOrchestrationGroupSharedContext(locale)}
        onChange={(event) => onUpdateNode(withGroupNode(selectedNode, { shared_context: event.target.value }))}
      />
      <p className="workflow-arch-field-note">{groupSharedContextNote(locale)}</p>
    </>
  );
}

function GroupSpeakingModeField(props: GroupFieldProps & { speakingMode: GroupSpeakingMode }) {
  const { copy, selectedNode, speakingMode, onUpdateNode } = props;

  return (
    <>
      <label className="workflow-arch-field-label">{copy.groupSpeakingMode}</label>
      <select
        value={speakingMode}
        onChange={(event) => onUpdateNode(withGroupNode(selectedNode, nextGroupSpeakingModePatch(
          selectedNode,
          event.target.value,
        )))}
      >
        <option value="sequential">{copy.groupSpeakingModeSequential}</option>
        <option value="parallel">{copy.groupSpeakingModeParallel}</option>
        <option value="owner">{copy.groupSpeakingModeOwner}</option>
      </select>
    </>
  );
}

function GroupOwnerField(props: GroupFieldProps & {
  memberOptions: GroupMemberOption[];
  speakingMode: GroupSpeakingMode;
}) {
  const { copy, memberOptions, selectedNode, speakingMode, onUpdateNode } = props;

  if (speakingMode !== 'owner') {
    return null;
  }

  return (
    <>
      <label className="workflow-arch-field-label">{copy.groupOwnerAgent}</label>
      <select
        value={selectedNode.group?.owner_agent_id ?? ''}
        onChange={(event) => onUpdateNode(withGroupNode(selectedNode, { owner_agent_id: event.target.value }))}
      >
        <option value="">{copy.groupOwnerAgentPlaceholder}</option>
        {memberOptions.map((item) => (
          <option key={item.id} value={item.id}>{item.label}</option>
        ))}
      </select>
      <p className="workflow-arch-field-note">{copy.groupOwnerDispatchNote}</p>
    </>
  );
}

function GroupMaxRoundsField(props: GroupFieldProps & { speakingMode: GroupSpeakingMode }) {
  const { copy, selectedNode, speakingMode, onUpdateNode } = props;

  return (
    <>
      <label className="workflow-arch-field-label">{copy.groupMaxRounds}</label>
      <input
        type="number"
        min={1}
        step={1}
        value={selectedNode.group?.max_rounds ?? 1}
        onChange={(event) => onUpdateNode(withGroupNode(selectedNode, {
          max_rounds: parseGroupMaxRounds(event.target.value),
        }))}
      />
      {speakingMode === 'owner' ? (
        <p className="workflow-arch-field-note">{copy.groupOwnerMaxRoundsHint}</p>
      ) : null}
    </>
  );
}

function resolveGroupMemberOptions(draft: WorkflowCanvasDraft | undefined, groupNodeID: string): GroupMemberOption[] {
  if (!draft) {
    return [];
  }
  const nodeByID = new Map(draft.nodes.map((node) => [node.id, node] as const));
  return draft.edges
    .filter((edge) => edge.kind === 'member' && edge.to_node_id === groupNodeID)
    .map((edge) => nodeByID.get(edge.from_node_id))
    .filter((node): node is WorkflowCanvasNodeDraft => node?.type === 'agent')
    .map((node) => ({
      id: node.id,
      label: `${node.agent?.title?.trim() || node.id} (${node.id})`,
    }));
}

function nextGroupSpeakingModePatch(
  selectedNode: WorkflowCanvasNodeDraft,
  rawMode: string,
): { owner_agent_id: string; speaking_mode: GroupSpeakingMode } {
  const speakingMode = rawMode as GroupSpeakingMode;

  if (speakingMode === 'owner') {
    return {
      speaking_mode: speakingMode,
      owner_agent_id: selectedNode.group?.owner_agent_id ?? '',
    };
  }

  return {
    speaking_mode: speakingMode,
    owner_agent_id: '',
  };
}

function parseGroupMaxRounds(rawValue: string): number {
  return Number.parseInt(rawValue, 10) || 0;
}

function groupSharedContextLabel(locale: WebLocale): string {
  if (locale === 'zh-CN') {
    return '群主初始提示词';
  }
  return 'Group Owner Initial Prompt';
}

function groupSharedContextNote(locale: WebLocale): string {
  if (locale === 'zh-CN') {
    return '这段内容会作为群主开场提示和群共享上下文注入给所有成员。';
  }
  return 'This text is injected to every member as the group owner opening prompt and shared context.';
}
