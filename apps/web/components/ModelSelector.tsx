'use client';

import type { FC } from 'react';

const MODELS = ['gpt-4o', 'gpt-4.1-mini', 'claude-3-7-sonnet-latest', 'qwen-plus'] as const;

interface ModelSelectorProps {
  model: string;
  disabled?: boolean;
  onChange: (model: string) => void;
}

export const ModelSelector: FC<ModelSelectorProps> = ({ model, disabled, onChange }) => {
  const hasPresetModel = MODELS.some((item) => item === model);

  return (
    <label className="flex items-center gap-2 text-sm text-app-muted">
      Model
      <select
        value={model}
        disabled={disabled}
        aria-label="Model selector"
        onChange={(event) => onChange(event.target.value)}
        className="mono rounded-lg border border-app-border bg-[#090f1d] px-2 py-1 text-sm text-app-text outline-none transition focus:border-app-accent disabled:opacity-60"
      >
        {MODELS.map((item) => (
          <option key={item} value={item}>
            {item}
          </option>
        ))}
        {!hasPresetModel && <option value={model}>{model}</option>}
      </select>
    </label>
  );
};
