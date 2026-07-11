import type { WorkflowNodeType } from '@/lib/workflow-editor';
import type { WorkflowNodeMeta } from '@/components/workflow/WorkflowCanvasNode';
import type { WebCopy } from '@/lib/i18n/messages';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';

type WorkflowNodeLabelKey = keyof WorkflowCopy['nodeLabels'];

const NODE_GLYPH_MAP: Record<WorkflowNodeType, string> = {
  start: '⚡',
  end: '✕',
  agent: '◈',
  group: '◎',
  llm: '⌨',
  tool: '⌁',
  if: '?',
  loop: '↻',
};

const NODE_LABEL_KEY_MAP: Record<WorkflowNodeType, WorkflowNodeLabelKey> = {
  start: 'start',
  end: 'end',
  agent: 'agent',
  group: 'group',
  llm: 'llm',
  tool: 'tool',
  if: 'if',
  loop: 'loop',
};

const NODE_LIBRARY_ORDER: WorkflowNodeType[] = [
  'agent',
  'llm',
  'tool',
  'if',
  'loop',
];

export function nodeLibraryForCopy(
  copy: WorkflowCopy | WebCopy,
  nodeTypes: readonly WorkflowNodeType[] = NODE_LIBRARY_ORDER,
): WorkflowNodeMeta[] {
  return nodeTypes.map((type) => ({
    id: type,
    label: labelByType(copy, type),
    glyph: NODE_GLYPH_MAP[type],
  }));
}

export function metadataForNodeType(
  type: WorkflowNodeType,
  copy: WorkflowCopy | WebCopy,
): WorkflowNodeMeta {
  return {
    id: type,
    label: labelByType(copy, type),
    glyph: NODE_GLYPH_MAP[type] ?? '·',
  };
}

function labelByType(copy: WorkflowCopy | WebCopy, type: WorkflowNodeType): string {
  const labels = 'workflow' in copy ? copy.workflow.nodeLabels : copy.nodeLabels;
  return labels[NODE_LABEL_KEY_MAP[type]];
}
