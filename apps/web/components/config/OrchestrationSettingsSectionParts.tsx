'use client';

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
