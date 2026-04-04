'use client';

import type { KeyboardEvent } from 'react';
import { ignorePromise } from '@/lib/errors';
import { labelForProviderType, stringsEqualIgnoreCase } from '@/lib/configProviders';
import type { ProviderConfig } from '@/lib/types';

interface ProviderListProps {
  providers: ProviderConfig[];
  activeProvider: string;
  loading: boolean;
  controlsDisabled: boolean;
  onActivate: (name: string) => Promise<void>;
  onDelete: (name: string) => Promise<void>;
  onEdit: (provider: ProviderConfig) => void;
}

interface ProviderListItemProps {
  provider: ProviderConfig;
  isActive: boolean;
  controlsDisabled: boolean;
  onActivate: (name: string) => Promise<void>;
  onDelete: (name: string) => Promise<void>;
  onEdit: (provider: ProviderConfig) => void;
}

export function ProviderList(props: ProviderListProps) {
  const { providers, activeProvider, loading, controlsDisabled, onActivate, onDelete, onEdit } = props;

  if (loading) {
    return (
      <div className="grid grid-cols-1 gap-3">
        {Array.from({ length: 3 }).map((_, index) => (
          <div
            key={`provider-skeleton-${index}`}
            className="h-[106px] animate-pulse rounded-[16px] border border-[#E5E5E5] bg-[#FAFAFA]"
          />
        ))}
      </div>
    );
  }

  if (providers.length === 0) {
    return (
      <div className="rounded-[16px] border border-[#E5E5E5] bg-white px-6 py-8 text-center text-[13px] text-[#737373]">
        No providers configured yet. Add one to populate <code>~/.ghost-os/config.toml</code>.
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-3">
      {providers.map((provider) => (
        <ProviderListItem
          key={provider.name}
          provider={provider}
          isActive={stringsEqualIgnoreCase(provider.name, activeProvider)}
          controlsDisabled={controlsDisabled}
          onActivate={onActivate}
          onDelete={onDelete}
          onEdit={onEdit}
        />
      ))}
    </div>
  );
}

function ProviderListItem(props: ProviderListItemProps) {
  const { provider, isActive, controlsDisabled, onActivate, onDelete, onEdit } = props;
  const endpoint = provider.base_url || 'Default endpoint';

  return (
    <article
      className="group relative flex cursor-pointer items-center justify-between gap-3 rounded-[16px] border border-[#E5E5E5] bg-white p-5 transition-all hover:border-[#111111]"
      onClick={() => onEdit(provider)}
      onKeyDown={(event) => handleCardKeyDown(event, provider, onEdit)}
      role="button"
      tabIndex={0}
    >
      <div className="min-w-0">
        <div className="mb-1 flex items-center gap-3">
          <span className="text-[15px] font-medium text-[#111111]">{provider.name}</span>
          {isActive ? <ActiveChip /> : null}
        </div>
        <p className="mb-1 text-[12px] text-[#737373]">{labelForProviderType(provider.type)}</p>
        <p className="max-w-[360px] truncate font-mono text-[12px] text-[#737373]">{endpoint}</p>
      </div>

      <div className="flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
        {isActive ? null : (
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={(event) => {
              event.stopPropagation();
              ignorePromise(onActivate(provider.name));
            }}
            className="rounded-full border border-[#E5E5E5] px-3 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
          >
            Activate
          </button>
        )}

        <button
          type="button"
          disabled={controlsDisabled}
          onClick={(event) => {
            event.stopPropagation();
            onEdit(provider);
          }}
          className="rounded-full p-2 text-[#737373] transition-colors hover:bg-[#F5F5F5] hover:text-[#111111] disabled:cursor-not-allowed disabled:opacity-50"
          aria-label={`Edit ${provider.name}`}
        >
          <EditIcon />
        </button>

        <button
          type="button"
          disabled={controlsDisabled}
          onClick={(event) => {
            event.stopPropagation();
            handleProviderDelete(provider.name, onDelete);
          }}
          className="rounded-full p-2 text-[#737373] transition-colors hover:bg-[#FEF2F2] hover:text-[#DC2626] disabled:cursor-not-allowed disabled:opacity-50"
          aria-label={`Delete ${provider.name}`}
        >
          <TrashIcon />
        </button>
      </div>
    </article>
  );
}

function ActiveChip() {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full bg-[#F5F5F5] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-[#111111]">
      <span className="h-1.5 w-1.5 rounded-full bg-[#111111]" />
      Active
    </span>
  );
}

function EditIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path d="m4.6 13.9 8.9-8.9 1.8 1.8-8.9 8.9-2.4.6.6-2.4Z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round" />
      <path d="m12.7 5.8 1.8 1.8" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
    </svg>
  );
}

function TrashIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path d="M4.6 5.5h10.8" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
      <path d="M7.3 5.5V4.2h5.4v1.3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
      <path d="m6.3 5.5.8 10h5.8l.8-10" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round" />
    </svg>
  );
}

function handleCardKeyDown(
  event: KeyboardEvent<HTMLElement>,
  provider: ProviderConfig,
  onEdit: (provider: ProviderConfig) => void,
) {
  if (event.key !== 'Enter' && event.key !== ' ') {
    return;
  }

  event.preventDefault();
  onEdit(provider);
}

function handleProviderDelete(name: string, onDelete: (name: string) => Promise<void>) {
  if (!window.confirm(`Delete provider "${name}"?`)) {
    return;
  }

  ignorePromise(onDelete(name));
}
