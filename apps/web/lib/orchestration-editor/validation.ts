import { validateOrchestrationGraph } from '@/lib/orchestration-editor/validationGraph';
import {
  validateOrchestrationAgentRuntime,
  type OrchestrationRuntimeValidationOptions,
} from '@/lib/orchestration-editor/validationRuntime';
import type { WorkflowCanvasDraft, WorkflowValidationResult } from '@/lib/workflow-editor/types';

export function validateOrchestrationDraft(
  draft: WorkflowCanvasDraft,
  options?: OrchestrationRuntimeValidationOptions,
): WorkflowValidationResult {
  const errors = [
    ...validateOrchestrationGraph(draft),
    ...validateOrchestrationAgentRuntime(draft, options),
  ];
  return { valid: errors.length === 0, errors };
}
