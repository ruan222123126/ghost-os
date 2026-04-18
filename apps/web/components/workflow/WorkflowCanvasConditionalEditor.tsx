'use client';

import type { WorkflowCanvasNodeDraft } from '@/lib/workflow-editor';
import { useWebLocale } from '@/lib/i18n/provider';
import {
  DEFAULT_LOOP_MAX_ITERATIONS,
  LOOP_ROLE_START,
  WORKFLOW_IF_OPERATORS,
} from '@/lib/workflow-editor/constants';

interface WorkflowCanvasConditionalEditorProps {
  selectedNode: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

export function IfEditor(props: WorkflowCanvasConditionalEditorProps) {
  const { selectedNode, onUpdateNode } = props;
  const config = selectedNode.if ?? buildDefaultIfConfig();
  const requiresValue = config.operator !== 'is_empty' && config.operator !== 'not_empty';

  return (
    <div className="workflow-arch-prop-group">
      <IfStartEditorFields
        config={config}
        requiresValue={requiresValue}
        onPatch={(patch) => onUpdateNode(withIfPatch(selectedNode, patch))}
      />
    </div>
  );
}

interface IfStartEditorFieldsProps {
  config: NonNullable<WorkflowCanvasNodeDraft['if']>;
  requiresValue: boolean;
  onPatch: (patch: Partial<NonNullable<WorkflowCanvasNodeDraft['if']>>) => void;
}

function IfStartEditorFields(props: IfStartEditorFieldsProps) {
  const { copy } = useWebLocale();
  const { config, requiresValue, onPatch } = props;

  return (
    <>
      <label className="workflow-arch-field-label">{copy.workflow.ifSourceNodeID}</label>
      <input
        type="text"
        value={config.source_node_id ?? ''}
        placeholder={copy.workflow.ifSourceNodePlaceholder}
        onChange={(event) => onPatch({ source_node_id: event.target.value })}
      />
      <label className="workflow-arch-field-label">{copy.workflow.ifOperator}</label>
      <select
        value={config.operator}
        onChange={(event) => onPatch({ operator: event.target.value as typeof config.operator })}
      >
        {WORKFLOW_IF_OPERATORS.map((operator) => (
          <option key={operator} value={operator}>{operator}</option>
        ))}
      </select>
      {requiresValue ? (
        <>
          <label className="workflow-arch-field-label">{copy.workflow.ifCompareValue}</label>
          <input
            type="text"
            value={config.value ?? ''}
            placeholder={copy.workflow.ifComparePlaceholder}
            onChange={(event) => onPatch({ value: event.target.value })}
          />
        </>
      ) : null}
      <label className="workflow-arch-field-label">{copy.workflow.ifTrueNodeID}</label>
      <input
        type="text"
        value={config.true_node_id ?? ''}
        placeholder={copy.workflow.ifTrueNodePlaceholder}
        onChange={(event) => onPatch({ true_node_id: event.target.value })}
      />
      <label className="workflow-arch-field-label">{copy.workflow.ifFalseNodeID}</label>
      <input
        type="text"
        value={config.false_node_id ?? ''}
        placeholder={copy.workflow.ifFalseNodePlaceholder}
        onChange={(event) => onPatch({ false_node_id: event.target.value })}
      />
    </>
  );
}

export function LoopEditor(props: WorkflowCanvasConditionalEditorProps) {
  const { copy } = useWebLocale();
  const { selectedNode, onUpdateNode } = props;
  const config = selectedNode.loop ?? {
    role: LOOP_ROLE_START,
    loop_id: '',
    max_iterations: DEFAULT_LOOP_MAX_ITERATIONS,
  };
  const loopID = config.loop_id?.trim() ?? '';

  return (
    <div className="workflow-arch-prop-group">
      <label className="workflow-arch-field-label">{copy.workflow.loopRole}</label>
      <input type="text" value={config.role} readOnly disabled />
      <label className="workflow-arch-field-label">{copy.workflow.loopID}</label>
      <input
        type="text"
        value={loopID || copy.workflow.loopIDMissing}
        readOnly
        disabled
      />
      {config.role === LOOP_ROLE_START ? (
        <>
          <label className="workflow-arch-field-label">{copy.workflow.loopMaxIterations}</label>
          <input
            type="number"
            min={1}
            value={config.max_iterations ?? DEFAULT_LOOP_MAX_ITERATIONS}
            onChange={(event) => {
              const value = Number.parseInt(event.target.value, 10);
              onUpdateNode(withLoopPatch(selectedNode, {
                max_iterations: Number.isFinite(value) && value > 0 ? value : DEFAULT_LOOP_MAX_ITERATIONS,
              }));
            }}
          />
        </>
      ) : (
        <p className="workflow-arch-summary-label">{copy.workflow.loopEndSummary}</p>
      )}
    </div>
  );
}

function withIfPatch(
  node: WorkflowCanvasNodeDraft,
  patch: Partial<NonNullable<WorkflowCanvasNodeDraft['if']>>,
): WorkflowCanvasNodeDraft {
  return {
    ...node,
    if: {
      ...buildDefaultIfConfig(),
      ...(node.if ?? {}),
      ...patch,
    },
  };
}

function withLoopPatch(
  node: WorkflowCanvasNodeDraft,
  patch: Partial<NonNullable<WorkflowCanvasNodeDraft['loop']>>,
): WorkflowCanvasNodeDraft {
  return {
    ...node,
    loop: {
      role: LOOP_ROLE_START,
      loop_id: '',
      max_iterations: DEFAULT_LOOP_MAX_ITERATIONS,
      ...(node.loop ?? {}),
      ...patch,
    },
  };
}

function buildDefaultIfConfig(): NonNullable<WorkflowCanvasNodeDraft['if']> {
  return {
    source_node_id: '',
    operator: 'equals',
    value: '',
    true_node_id: '',
    false_node_id: '',
  };
}
