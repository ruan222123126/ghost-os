'use client';

import { WorkflowCanvasWorkbench } from '@/components/workflow/WorkflowCanvasWorkbench';
import { useOrchestrationEditorController } from '@/hooks/orchestration/useOrchestrationEditorController';
import { useWebLocale } from '@/lib/i18n/provider';
import type { WorkflowNodeType } from '@/lib/workflow-editor';

interface OrchestrationEditorClientProps {
  orchestrationID: string;
}

const ORCHESTRATION_NODE_LIBRARY_TYPES: readonly WorkflowNodeType[] = [
  'agent',
  'group',
];

export function OrchestrationEditorClient(props: OrchestrationEditorClientProps) {
  const { copy, locale } = useWebLocale();
  const controller = useOrchestrationEditorController({
    orchestrationID: props.orchestrationID,
    copy,
    locale,
  });

  if (controller.phase === 'loading') {
    return <main className="workflow-loading">{controller.workflowCopy.loadingTask}</main>;
  }

  if (controller.phase === 'missing') {
    return (
      <main className="mx-auto flex min-h-screen w-full max-w-3xl items-center px-6 py-16">
        <section className="w-full rounded-[24px] border border-[#FECACA] bg-white p-8 shadow-sm">
          <h1 className="mb-3 text-[28px] font-semibold tracking-tight text-[#111111]">
            {copy.settings.orchestrationMissingTitle}
          </h1>
          <p className="mb-8 text-[14px] text-[#737373]">
            {copy.settings.orchestrationMissingDescription}
          </p>
          <button
            type="button"
            onClick={controller.onBack}
            className="inline-flex items-center rounded-full border border-[#111111] px-4 py-2 text-[13px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5]"
          >
            {copy.settings.orchestrationBackToSettings}
          </button>
        </section>
      </main>
    );
  }

  return (
    <WorkflowCanvasWorkbench
      editorKind="orchestration"
      draft={controller.draft}
      autosaveState={controller.autosaveState}
      actionError={controller.actionError}
      validationErrors={controller.validationErrors}
      agentRuntimeCatalog={controller.agentRuntimeCatalog}
      agentRuntimeLoading={controller.agentRuntimeLoading}
      agentRuntimeError={controller.agentRuntimeError}
      presets={controller.presets}
      presetLoading={controller.presetLoading}
      presetError={controller.presetError}
      workflowCopy={controller.workflowCopy}
      nodeLibraryTypes={ORCHESTRATION_NODE_LIBRARY_TYPES}
      localizeValidationError={controller.localizeValidationError}
      onScheduleChange={controller.onScheduleChange}
      onAddNode={controller.onAddNode}
      onSelectNode={controller.onSelectNode}
      onMoveNode={controller.onMoveNode}
      onConnectNodes={controller.onConnectNodes}
      onDeleteEdge={controller.onDeleteEdge}
      onDuplicateNode={controller.onDuplicateNode}
      onUpdateNode={controller.onUpdateNode}
      onDeleteNode={controller.onDeleteNode}
      onSave={controller.onSave}
      onBack={controller.onBack}
    />
  );
}
