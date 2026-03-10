// ModelSelector renders the active model picker inside the chat composer toolbar.

'use client';

import type { FC } from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { ignorePromise } from '@/lib/errors';
import type { ProviderConfig, ProviderModelOption } from '@/lib/types';

interface ModelSelectorProps {
  value: ProviderModelOption | null;
  options: ProviderModelOption[];
  loading?: boolean;
  disabled?: boolean;
  onChange: (option: ProviderModelOption) => void | Promise<unknown>;
}

interface ProviderModelGroup {
  providerName: string;
  providerType: ProviderConfig['type'];
  options: ProviderModelOption[];
}

export const ModelSelector: FC<ModelSelectorProps> = ({ value, options, loading = false, disabled = false, onChange }) => {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement | null>(null);
  const groups = useMemo(() => groupProviderModels(options), [options]);
  const triggerDisabled = disabled || (!loading && options.length === 0);

  useEffect(() => {
    if (!open) {
      return;
    }

    function handlePointerDown(event: MouseEvent | TouchEvent) {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        setOpen(false);
      }
    }

    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('touchstart', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('touchstart', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [open]);

  return (
    <div ref={rootRef} className="composer-model-selector">
      <button
        type="button"
        disabled={triggerDisabled}
        aria-label="Select active model"
        aria-haspopup="dialog"
        aria-expanded={open}
        className={`composer-model-trigger${open ? ' is-open' : ''}`}
        onClick={() => setOpen((current) => !current)}
      >
        <span className="composer-model-trigger-copy">
          <span className="composer-model-trigger-model mono">
            {loading ? '加载模型中…' : value?.model || '未配置模型'}
          </span>
        </span>
        <span className="composer-model-trigger-chevron" aria-hidden="true" />
      </button>

      {open ? (
        <div className="composer-model-panel" role="dialog" aria-label="Active model selector">
          <div className="composer-model-panel-body ui-scroll">
            {groups.length === 0 ? (
              <div className="composer-model-empty">当前没有可选模型，请先在 Settings 中为供应商配置模型列表。</div>
            ) : (
              groups.map((group) => (
                <section key={group.providerName} className="composer-model-group">
                  <div className="composer-model-group-head">
                    <span>{group.providerName}</span>
                    <span>{labelForProviderType(group.providerType)}</span>
                  </div>

                  <div className="composer-model-group-options">
                    {group.options.map((option) => {
                      const isActive = isSameOption(option, value);

                      return (
                        <button
                          key={`${option.providerName}:${option.model}`}
                          type="button"
                          disabled={disabled}
                          className={`composer-model-option${isActive ? ' is-active' : ''}`}
                          onClick={() => {
                            setOpen(false);
                            ignorePromise(Promise.resolve(onChange(option)));
                          }}
                        >
                          <span className="composer-model-option-copy">
                            <span className="composer-model-option-model mono">{option.model}</span>
                          </span>
                          <span className="composer-model-option-check" aria-hidden="true">
                            {isActive ? '●' : ''}
                          </span>
                        </button>
                      );
                    })}
                  </div>
                </section>
              ))
            )}
          </div>
        </div>
      ) : null}
    </div>
  );
};

function groupProviderModels(options: ProviderModelOption[]): ProviderModelGroup[] {
  const groups = new Map<string, ProviderModelGroup>();
  const order: string[] = [];

  for (const option of options) {
    const key = option.providerName.trim().toLowerCase();
    if (!groups.has(key)) {
      order.push(key);
      groups.set(key, {
        providerName: option.providerName,
        providerType: option.providerType,
        options: [],
      });
    }

    groups.get(key)?.options.push(option);
  }

  return order.map((key) => groups.get(key)).filter((group): group is ProviderModelGroup => group !== undefined);
}

function labelForProviderType(providerType: ProviderConfig['type']): string {
  if (providerType === 'openai') {
    return 'OpenAI';
  }
  if (providerType === 'codex') {
    return 'Codex';
  }
  if (providerType === 'anthropic') {
    return 'Anthropic';
  }
  return 'OpenAI-Compatible';
}

function isSameOption(left: ProviderModelOption, right: ProviderModelOption | null): boolean {
  if (!right) {
    return false;
  }

  return left.providerName.trim().toLowerCase() === right.providerName.trim().toLowerCase()
    && left.model.trim().toLowerCase() === right.model.trim().toLowerCase();
}
