'use client';

import { useWebLocale } from '@/lib/i18n/provider';

export function LegacyMigrationBanner(props: {
  count: number;
  running: boolean;
  onMigrate: () => void;
}) {
  const { copy } = useWebLocale();

  return (
    <div className="mb-4 rounded-[16px] border border-[#D4D4D4] bg-[#FAFAFA] px-6 py-4 text-[13px] text-[#262626]">
      <p className="font-medium text-[#111111]">{copy.settings.orchestrationLegacyMigrationTitle(props.count)}</p>
      <p className="mt-1 text-[#525252]">{copy.settings.orchestrationLegacyMigrationRisk}</p>
      <button
        type="button"
        onClick={props.onMigrate}
        disabled={props.running}
        className="mt-3 inline-flex items-center rounded-full bg-[#111111] px-4 py-2 text-[12px] font-medium text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:bg-[#A3A3A3]"
      >
        {props.running ? copy.settings.orchestrationLegacyMigrationRunning : copy.settings.orchestrationLegacyMigrationAction}
      </button>
    </div>
  );
}

export function OrchestrationStatusBanner(props: {
  message: string;
  tone: 'error' | 'success';
}) {
  if (!props.message) {
    return null;
  }

  const toneClassName = props.tone === 'success'
    ? 'border-green-200 bg-green-50 text-green-700'
    : 'border-red-200 bg-red-50 text-red-700';

  return <div className={`mb-4 rounded-[16px] border px-6 py-4 text-[13px] ${toneClassName}`}>{props.message}</div>;
}

export function OrchestrationLoadingList() {
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
