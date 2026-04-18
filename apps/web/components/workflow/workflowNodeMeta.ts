import type { WorkflowNodeType } from '@/lib/workflow-editor';
import type { WorkflowNodeMeta } from '@/components/workflow/WorkflowCanvasNode';
import type { WebCopy } from '@/lib/i18n/messages';

const NODE_GLYPH_MAP: Record<WorkflowNodeType, string> = {
  start: '⚡',
  end: '✕',
  agent: '◈',
  llm: '⌨',
  tool: '⌁',
  if: '?',
  loop: '↻',
};

const NODE_LIBRARY_ORDER: WorkflowNodeType[] = [
  'start',
  'end',
  'agent',
  'llm',
  'tool',
  'if',
  'loop',
];

export function nodeLibraryForCopy(copy: WebCopy): WorkflowNodeMeta[] {
  return NODE_LIBRARY_ORDER.map((type) => ({
    id: type,
    label: labelByType(copy, type),
    glyph: NODE_GLYPH_MAP[type],
  }));
}

export function metadataForNodeType(type: WorkflowNodeType, copy: WebCopy): WorkflowNodeMeta {
  return {
    id: type,
    label: labelByType(copy, type),
    glyph: NODE_GLYPH_MAP[type] ?? '·',
  };
}

function labelByType(copy: WebCopy, type: WorkflowNodeType): string {
  const labels = copy.workflow.nodeLabels;
  if (type === 'start') {
    return labels.start;
  }
  if (type === 'end') {
    return labels.end;
  }
  if (type === 'agent') {
    return labels.agent;
  }
  if (type === 'llm') {
    return labels.llm;
  }
  if (type === 'tool') {
    return labels.tool;
  }
  if (type === 'if') {
    return labels.if;
  }
  return labels.loop;
}
