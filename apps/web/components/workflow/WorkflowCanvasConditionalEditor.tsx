'use client';

import { WorkflowTemplateEnabledField } from '@/components/workflow/WorkflowTemplateEnabledField';
import { useWebLocale } from '@/lib/i18n/provider';
import type { WorkflowCanvasNodeDraft, WorkflowEditorKind } from '@/lib/workflow-editor';
import {
  DEFAULT_LOOP_MAX_ITERATIONS,
  LOOP_ROLE_START,
  WORKFLOW_IF_OPERATORS,
} from '@/lib/workflow-editor/constants';

interface WorkflowCanvasConditionalEditorProps {
  editorKind: WorkflowEditorKind;
  selectedNode: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

export function IfEditor(props: WorkflowCanvasConditionalEditorProps) {
  const { copy } = useWebLocale();
  const { editorKind, selectedNode, onUpdateNode } = props;
  const config = selectedNode.if ?? buildDefaultIfConfig();
  const requiresValue = config.operator !== 'is_empty' && config.operator !== 'not_empty';

  return (
    <div className="workflow-arch-prop-group">
      <IfStartEditorFields
        editorKind={editorKind}
        config={config}
        requiresValue={requiresValue}
        onPatch={(patch) => onUpdateNode(withIfPatch(selectedNode, patch))}
      />
      {editorKind === 'workflow' ? <p className="workflow-arch-field-note">{copy.workflow.runtimeVariableHint}</p> : null}
    </div>
  );
}

interface IfStartEditorFieldsProps {
  editorKind: WorkflowEditorKind;
  config: NonNullable<WorkflowCanvasNodeDraft['if']>;
  requiresValue: boolean;
  onPatch: (patch: Partial<NonNullable<WorkflowCanvasNodeDraft['if']>>) => void;
}

interface IfEditorFieldProps {
  config: NonNullable<WorkflowCanvasNodeDraft['if']>;
  onPatch: (patch: Partial<NonNullable<WorkflowCanvasNodeDraft['if']>>) => void;
}

interface LoopEditorFieldProps {
  config: NonNullable<WorkflowCanvasNodeDraft['loop']>;
  selectedNode: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

function IfStartEditorFields(props: IfStartEditorFieldsProps) {
  const { editorKind, config, requiresValue, onPatch } = props;

  return (
    <>
      <IfSourceNodeField config={config} onPatch={onPatch} />
      <IfOperatorField config={config} onPatch={onPatch} />
      <IfCompareValueField
        editorKind={editorKind}
        config={config}
        visible={requiresValue}
        onPatch={onPatch}
      />
      <IfTargetNodeFields config={config} onPatch={onPatch} />
    </>
  );
}

function IfSourceNodeField(props: IfEditorFieldProps) {
  const { copy } = useWebLocale();
  const { config, onPatch } = props;

  return (
    <>
      <label className="workflow-arch-field-label">{copy.workflow.ifSourceNodeID}</label>
      <input
        type="text"
        value={config.source_node_id ?? ''}
        placeholder={copy.workflow.ifSourceNodePlaceholder}
        onChange={(event) => onPatch({ source_node_id: event.target.value })}
      />
    </>
  );
}

function IfOperatorField(props: IfEditorFieldProps) {
  const { copy } = useWebLocale();
  const { config, onPatch } = props;

  return (
    <>
      <label className="workflow-arch-field-label">{copy.workflow.ifOperator}</label>
      <select
        value={config.operator}
        onChange={(event) => onPatch({ operator: event.target.value as typeof config.operator })}
      >
        {WORKFLOW_IF_OPERATORS.map((operator) => (
          <option key={operator} value={operator}>{operator}</option>
        ))}
      </select>
    </>
  );
}

function IfCompareValueField(props: IfEditorFieldProps & {
  editorKind: WorkflowEditorKind;
  visible: boolean;
}) {
  const { copy } = useWebLocale();
  const { editorKind, config, visible, onPatch } = props;

  if (!visible) {
    return null;
  }

  return (
    <>
      <label className="workflow-arch-field-label">{copy.workflow.ifCompareValue}</label>
      <WorkflowTemplateEnabledField
        editorKind={editorKind}
        mode="input"
        value={config.value ?? ''}
        placeholder={copy.workflow.ifComparePlaceholder}
        onChange={(value) => onPatch({ value })}
      />
    </>
  );
}

function IfTargetNodeFields(props: IfEditorFieldProps) {
  const { copy } = useWebLocale();
  const { config, onPatch } = props;

  return (
    <>
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
  const { selectedNode, onUpdateNode } = props;
  const config = selectedNode.loop ?? {
    role: LOOP_ROLE_START,
    loop_id: '',
    max_iterations: DEFAULT_LOOP_MAX_ITERATIONS,
  };

  return (
    <div className="workflow-arch-prop-group">
      <LoopIdentityFields config={config} />
      <LoopRoleFields config={config} selectedNode={selectedNode} onUpdateNode={onUpdateNode} />
    </div>
  );
}

function LoopIdentityFields(props: { config: NonNullable<WorkflowCanvasNodeDraft['loop']> }) {
  const { copy } = useWebLocale();
  const { config } = props;
  const loopID = config.loop_id?.trim() ?? '';

  return (
    <>
      <label className="workflow-arch-field-label">{copy.workflow.loopRole}</label>
      <input type="text" value={config.role} readOnly disabled />
      <label className="workflow-arch-field-label">{copy.workflow.loopID}</label>
      <input
        type="text"
        value={loopID || copy.workflow.loopIDMissing}
        readOnly
        disabled
      />
    </>
  );
}

function LoopRoleFields(props: LoopEditorFieldProps) {
  const { copy } = useWebLocale();
  const { config, selectedNode, onUpdateNode } = props;

  if (config.role !== LOOP_ROLE_START) {
    return <p className="workflow-arch-summary-label">{copy.workflow.loopEndSummary}</p>;
  }

  return (
    <>
      <label className="workflow-arch-field-label">{copy.workflow.loopMaxIterations}</label>
      <input
        type="number"
        min={1}
        value={config.max_iterations ?? DEFAULT_LOOP_MAX_ITERATIONS}
        onChange={(event) => onUpdateNode(withLoopPatch(selectedNode, {
          max_iterations: parseLoopMaxIterations(event.target.value),
        }))}
      />
    </>
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

function parseLoopMaxIterations(rawValue: string): number {
  const value = Number.parseInt(rawValue, 10);
  if (Number.isFinite(value) && value > 0) {
    return value;
  }
  return DEFAULT_LOOP_MAX_ITERATIONS;
}
