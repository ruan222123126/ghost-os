// ModelSelector component used by the web console chat/session interface.

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
    <label className="flex items-center gap-2">
      <span className="ui-hint">Model</span>
      <select
        value={model}
        disabled={disabled}
        aria-label="Model selector"
        onChange={(event) => onChange(event.target.value)}
        className="ui-select mono min-w-[210px] text-sm"
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
