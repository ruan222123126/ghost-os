// QuestionInput component used by the web console chat/session interface.

'use client';

import type { FC } from 'react';
import { useMemo, useState } from 'react';
import { ignorePromise } from '@/lib/errors';
import type { AskHumanOption } from '@/lib/types';

interface QuestionInputProps {
  loading: boolean;
  selectionMode?: 'single' | 'multiple';
  options?: AskHumanOption[];
  onAnswer: (answer: string) => Promise<void>;
  onCancel: () => Promise<void>;
}

function normalizeOptions(options?: AskHumanOption[]): AskHumanOption[] {
  if (!options || options.length === 0) {
    return [];
  }

  return options
    .map((option) => ({
      label: option.label.trim(),
      allow_custom: option.allow_custom,
    }))
    .filter((option) => option.label.length > 0);
}

function buildChoiceAnswer(options: AskHumanOption[], selectedIndexes: number[], customText: string): string {
  const normalizedCustom = customText.trim();
  const parts = selectedIndexes
    .slice()
    .sort((left, right) => left - right)
    .flatMap((index) => {
      const option = options[index];
      if (!option) {
        return [];
      }
      if (option.allow_custom) {
        return normalizedCustom ? [normalizedCustom] : [];
      }
      return [option.label];
    });

  if (parts.length === 0) {
    return '';
  }
  if (parts.length === 1) {
    return parts[0] ?? '';
  }
  return `- ${parts.join('\n- ')}`;
}

export const QuestionInput: FC<QuestionInputProps> = ({
  loading,
  selectionMode,
  options,
  onAnswer,
  onCancel,
}) => {
  const normalizedOptions = useMemo(() => normalizeOptions(options), [options]);
  const hasOptions = normalizedOptions.length > 0;
  const mode = selectionMode === 'multiple' ? 'multiple' : 'single';
  const [singleIndex, setSingleIndex] = useState<number | null>(null);
  const [selectedIndexes, setSelectedIndexes] = useState<number[]>([]);
  const [customText, setCustomText] = useState('');

  const effectiveSelections = hasOptions
    ? mode === 'multiple'
      ? selectedIndexes
      : singleIndex === null
        ? []
        : [singleIndex]
    : [];
  const customOptionIndex = normalizedOptions.findIndex((option) => option.allow_custom);
  const customSelected = customOptionIndex >= 0 && effectiveSelections.includes(customOptionIndex);
  const canSubmit = hasOptions
    ? effectiveSelections.length > 0 && (!customSelected || customText.trim().length > 0)
    : customText.trim().length > 0;

  async function submit() {
    if (loading || !canSubmit) {
      return;
    }

    const answer = hasOptions
      ? buildChoiceAnswer(normalizedOptions, effectiveSelections, customText)
      : customText.trim();
    if (!answer) {
      return;
    }

    await onAnswer(answer);
    setSingleIndex(null);
    setSelectedIndexes([]);
    setCustomText('');
  }

  function toggleOption(index: number) {
    if (loading) {
      return;
    }

    if (mode === 'multiple') {
      setSelectedIndexes((current) =>
        current.includes(index) ? current.filter((value) => value !== index) : [...current, index]
      );
      return;
    }

    setSingleIndex((current) => (current === index ? null : index));
  }

  return (
    <div className="ui-panel-soft mt-3 border-amber-300/30 bg-amber-300/8 p-3">
      {hasOptions && (
        <div className="space-y-2">
          <div className="ui-hint text-amber-100/70">
            {mode === 'multiple' ? 'Choose one or more options' : 'Choose one option'}
          </div>
          {normalizedOptions.map((option, index) => {
            const checked = effectiveSelections.includes(index);
            return (
              <label
                key={`${option.label}-${index}`}
                className={`flex cursor-pointer items-start gap-3 rounded-lg border px-3 py-2 transition ${
                  checked
                    ? 'border-amber-200/55 bg-amber-200/10 text-amber-50 shadow-lift'
                    : 'border-amber-200/15 bg-app-field/80 text-amber-100'
                } ${loading ? 'cursor-not-allowed opacity-70' : 'hover:border-amber-300/35'}`}
              >
                <input
                  type={mode === 'multiple' ? 'checkbox' : 'radio'}
                  name="ask-human-option"
                  checked={checked}
                  disabled={loading}
                  onChange={() => toggleOption(index)}
                  className="mt-1 h-4 w-4 accent-amber-300"
                />
                <span className="text-sm leading-relaxed">{option.label}</span>
              </label>
            );
          })}
        </div>
      )}

      {(!hasOptions || customSelected) && (
        <textarea
          value={customText}
          disabled={loading}
          aria-label={hasOptions ? 'Custom answer' : 'Answer question'}
          onChange={(event) => setCustomText(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && !event.shiftKey) {
              event.preventDefault();
              ignorePromise(submit());
            }
          }}
          placeholder={hasOptions ? 'Type your custom answer...' : 'Type your answer...'}
          rows={hasOptions ? 3 : 2}
          className="ui-textarea mono mt-3 min-h-[84px] w-full resize-none border-amber-200/20 bg-app-field/85 text-amber-100 focus:border-amber-300/50 focus:shadow-[inset_0_1px_0_rgb(255_255_255_/_0.04),0_0_0_3px_rgb(252_211_77_/_0.16)]"
        />
      )}

      <div className="mt-3 flex items-center justify-between gap-3">
        <span className="ui-hint text-amber-100/70">
          {hasOptions ? 'Submit your selection, or cancel this question' : 'Enter to submit answer, or cancel'}
        </span>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => {
              ignorePromise(onCancel());
            }}
            disabled={loading}
            className="ui-btn-secondary border-rose-300/35 bg-rose-300/12 px-3 py-1.5 text-sm text-rose-100 hover:border-rose-300/55 hover:bg-rose-300/18"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => {
              ignorePromise(submit());
            }}
            disabled={loading || !canSubmit}
            className="ui-btn border-amber-300/40 bg-amber-300/18 px-3 py-1.5 text-sm text-amber-100 hover:border-amber-300/55 hover:bg-amber-300/24"
          >
            {loading ? 'Submitting...' : 'Submit'}
          </button>
        </div>
      </div>
    </div>
  );
};
