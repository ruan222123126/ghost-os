'use client';

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

export function ProviderList(props: ProviderListProps) {
  const { providers, activeProvider, loading, controlsDisabled, onActivate, onDelete, onEdit } = props;

  return (
    <div className="provider-list">
      {loading ? (
        Array.from({ length: 3 }).map((_, index) => (
          <div key={`provider-skeleton-${index}`} className="session-skeleton" />
        ))
      ) : providers.length === 0 ? (
        <div className="empty-state">No providers configured yet. Add one to populate `~/.ghost-os/config.toml`.</div>
      ) : (
        providers.map((provider) => {
          const isActive = stringsEqualIgnoreCase(provider.name, activeProvider);

          return (
            <article key={provider.name} className={`provider-card${isActive ? ' is-active' : ''}`}>
              <div className="provider-card-head">
                <div>
                  <h3>{provider.name}</h3>
                  <p className="provider-card-label">{labelForProviderType(provider.type)}</p>
                </div>
                <span className={`status-chip${isActive ? ' status-chip-active' : ''}`}>
                  {isActive ? 'Active' : 'Saved'}
                </span>
              </div>

              <dl className="detail-list">
                <div className="detail-row">
                  <dt>Base URL</dt>
                  <dd className="mono">{provider.base_url || 'Default endpoint'}</dd>
                </div>
                <div className="detail-row">
                  <dt>Models</dt>
                  <dd>{provider.models?.length ? provider.models.join(', ') : 'Any compatible model'}</dd>
                </div>
              </dl>

              <div className="provider-card-actions">
                <button
                  type="button"
                  disabled={controlsDisabled || isActive}
                  onClick={() => {
                    ignorePromise(onActivate(provider.name));
                  }}
                  className="button-secondary"
                >
                  {isActive ? 'Active' : 'Activate'}
                </button>
                <button
                  type="button"
                  disabled={controlsDisabled}
                  onClick={() => onEdit(provider)}
                  className="button-secondary"
                >
                  Edit
                </button>
                <button
                  type="button"
                  disabled={controlsDisabled}
                  onClick={() => handleProviderDelete(provider.name, onDelete)}
                  className="button-danger"
                >
                  Delete
                </button>
              </div>
            </article>
          );
        })
      )}
    </div>
  );
}

function handleProviderDelete(
  name: string,
  onDelete: (name: string) => Promise<void>,
) {
  if (!window.confirm(`Delete provider "${name}"?`)) {
    return;
  }

  ignorePromise(onDelete(name));
}
