'use client';

import { useRouter } from 'next/navigation';
import { OrchestrationCreateForm } from '@/components/config/OrchestrationCreateForm';
import { useWebLocale } from '@/lib/i18n/provider';
import type { OrchestrationSummary } from '@/lib/orchestrationStore';
import { useOrchestrationSectionState } from './useOrchestrationSectionState';

export function OrchestrationSettingsSection() {
  const { copy } = useWebLocale();
  const router = useRouter();
  const state = useOrchestrationSectionState(copy);

  if (state.creating) {
    return (
      <OrchestrationCreateForm
        name={state.name}
        controlsDisabled={state.controlsDisabled}
        saving={state.submitting}
        onChangeName={state.setName}
        onSubmit={state.submitCreate}
        onCancel={state.cancelCreate}
      />
    );
  }

  return (
    <section data-testid="orchestration-settings-section">
      <SectionHeader onCreate={state.startCreate} />
      {state.error ? <ErrorBanner message={state.error} /> : null}
      <SectionBody
        loading={state.loading}
        error={state.error}
        orchestrations={state.orchestrations}
        onOpen={(id) => router.push(`/orchestration/${encodeURIComponent(id)}`)}
      />
    </section>
  );
}

function SectionHeader(props: { onCreate: () => void }) {
  const { copy } = useWebLocale();

  return (
    <header className="mb-10 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">{copy.settings.orchestrationTitle}</h1>
        <p className="text-[14px] text-[#737373]">{copy.settings.orchestrationDescription}</p>
      </div>

      <button
        type="button"
        onClick={props.onCreate}
        className="inline-flex items-center rounded-full bg-[#111111] px-4 py-2 text-[13px] font-medium text-white transition-colors hover:bg-[#333333]"
      >
        {copy.settings.orchestrationNew}
      </button>
    </header>
  );
}

function ErrorBanner(props: { message: string }) {
  return (
    <div className="rounded-[16px] border border-red-200 bg-red-50 px-6 py-4 text-[13px] text-red-700">
      {props.message}
    </div>
  );
}

function SectionBody(props: {
  loading: boolean;
  error: string;
  orchestrations: OrchestrationSummary[];
  onOpen: (id: string) => void;
}) {
  const { copy } = useWebLocale();
  const { loading, error, orchestrations, onOpen } = props;

  if (loading) {
    return <LoadingList />;
  }
  if (error) {
    return null;
  }
  if (orchestrations.length === 0) {
    return (
      <div className="rounded-[16px] border border-[#E5E5E5] bg-white px-6 py-8 text-center text-[13px] text-[#737373]">
        {copy.settings.orchestrationEmpty}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-3">
      {orchestrations.map((orchestration) => (
        <OrchestrationCard
          key={orchestration.id}
          orchestration={orchestration}
          onOpen={onOpen}
        />
      ))}
    </div>
  );
}

function LoadingList() {
  return (
    <div className="grid grid-cols-1 gap-3">
      {Array.from({ length: 3 }).map((_, index) => (
        <div
          key={`orchestration-skeleton-${index}`}
          className="h-[130px] animate-pulse rounded-[16px] border border-[#E5E5E5] bg-[#FAFAFA]"
        />
      ))}
    </div>
  );
}

function OrchestrationCard(props: {
  orchestration: OrchestrationSummary;
  onOpen: (id: string) => void;
}) {
  const { copy } = useWebLocale();
  const { orchestration, onOpen } = props;

  return (
    <button
      type="button"
      onClick={() => onOpen(orchestration.id)}
      className="group rounded-[16px] border border-[#E5E5E5] bg-white p-5 text-left transition-all hover:border-[#111111]"
    >
      <div className="mb-2 flex items-center gap-2">
        <span className="inline-flex rounded-full bg-[#F5F5F5] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-[#111111]">
          {copy.settings.tabOrchestration}
        </span>
        <span className="font-mono text-[11px] text-[#737373]">{orchestration.id}</span>
      </div>
      <p className="mb-1 line-clamp-2 text-[14px] font-medium text-[#111111]">{orchestration.name}</p>
      <p className="text-[12px] text-[#737373]">{copy.settings.orchestrationSteps(orchestration.stepCount)}</p>
      <p className="mt-3 text-[12px] text-[#737373]">{copy.settings.orchestrationOpenEditor}</p>
    </button>
  );
}
