import type { WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

export interface WorkflowCanvasNodeSource {
  id: string;
  type: WorkflowCanvasNodeDraft['type'];
  start?: WorkflowCanvasNodeDraft['start'];
  tool?: WorkflowCanvasNodeDraft['tool'];
  llm?: WorkflowCanvasNodeDraft['llm'];
  agent?: WorkflowCanvasNodeDraft['agent'];
  if?: WorkflowCanvasNodeDraft['if'];
  loop?: WorkflowCanvasNodeDraft['loop'];
}
