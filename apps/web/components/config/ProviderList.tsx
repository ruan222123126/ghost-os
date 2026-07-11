'use client';

import type { KeyboardEvent } from 'react';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
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
  const { copy } = useWebLocale();
  const { providers, activeProvider, loading, controlsDisabled, onActivate, onDelete, onEdit } = props;

  if (loading) {
    return (
      <div className="settings-card-list">
        {Array.from({ length: 3 }).map((_, index) => (
          <div
            key={`provider-skeleton-${index}`}
            className="settings-card-skeleton animate-pulse"
          />
        ))}
      </div>
    );
  }

  if (providers.length === 0) {
    return (
      <div className="settings-list-empty">
        {copy.settings.providerNoItems}
      </div>
    );
  }

  return (
    <div className="settings-card-list">
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
  const { copy } = useWebLocale();
  const { provider, isActive, controlsDisabled, onActivate, onDelete, onEdit } = props;
  const endpoint = provider.base_url || copy.settings.providerDefaultEndpoint;

  return (
    <article
      className="settings-config-card is-clickable"
      onClick={() => onEdit(provider)}
      onKeyDown={(event) => handleCardKeyDown(event, provider, onEdit)}
      role="button"
      tabIndex={0}
    >
      <div className="settings-config-card-main">
        <div className="settings-card-badges">
          <span className="settings-card-title is-strong">{provider.name}</span>
          {isActive ? <ActiveChip /> : null}
        </div>
        <p className="settings-card-meta">{labelForProviderType(provider.type)}</p>
        <p className="settings-card-meta is-mono truncate">{endpoint}</p>
      </div>

      <div className="settings-config-card-actions">
        {isActive ? null : (
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={(event) => {
              event.stopPropagation();
              ignorePromise(onActivate(provider.name));
            }}
            className="settings-card-action whitespace-nowrap"
          >
            {copy.settings.providerActivate}
          </button>
        )}

        <button
          type="button"
          disabled={controlsDisabled}
          onClick={(event) => {
            event.stopPropagation();
            onEdit(provider);
          }}
          className="settings-card-icon-action"
          aria-label={copy.settings.providerEditAria(provider.name)}
        >
          <EditIcon />
        </button>

        <button
          type="button"
          disabled={controlsDisabled}
          onClick={(event) => {
            event.stopPropagation();
            handleProviderDelete(provider.name, onDelete, copy.settings.providerDeleteConfirm(provider.name));
          }}
          className="settings-card-icon-action is-danger"
          aria-label={copy.settings.providerDeleteAria(provider.name)}
        >
          <TrashIcon />
        </button>
      </div>
    </article>
  );
}

function ActiveChip() {
  const { copy } = useWebLocale();

  return (
    <span className="settings-card-badge">
      {copy.settings.providerActive}
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

function handleProviderDelete(name: string, onDelete: (name: string) => Promise<void>, message: string) {
  if (!window.confirm(message)) {
    return;
  }

  ignorePromise(onDelete(name));
}
