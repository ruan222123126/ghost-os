// QuestionInput component used by the web console chat/session interface.

'use client';

import type { FC } from 'react';
import { useMemo, useState } from 'react';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
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
  const { copy } = useWebLocale();
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
    <div className="question-card">
      {hasOptions ? (
        <>
          <div className="field-hint">{mode === 'multiple' ? copy.chat.questionChooseMultiple : copy.chat.questionChooseSingle}</div>
          <div className="choice-list">
            {normalizedOptions.map((option, index) => {
              const checked = effectiveSelections.includes(index);

              return (
                <label key={`${option.label}-${index}`} className={`choice-option${checked ? ' is-selected' : ''}`}>
                  <input
                    type={mode === 'multiple' ? 'checkbox' : 'radio'}
                    name="ask-human-option"
                    checked={checked}
                    disabled={loading}
                    onChange={() => toggleOption(index)}
                  />
                  <span>{option.label}</span>
                </label>
              );
            })}
          </div>
        </>
      ) : null}

      {(!hasOptions || customSelected) ? (
        <textarea
          value={customText}
          disabled={loading}
          aria-label={hasOptions ? copy.chat.questionCustomAnswerAria : copy.chat.questionAnswerAria}
          onChange={(event) => setCustomText(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && !event.shiftKey) {
              event.preventDefault();
              ignorePromise(submit());
            }
          }}
          placeholder={hasOptions ? copy.chat.questionCustomPlaceholder : copy.chat.questionAnswerPlaceholder}
          rows={hasOptions ? 3 : 2}
          className="textarea mono"
        />
      ) : null}

      <div className="question-actions">
        <span className="field-hint">
          {hasOptions ? copy.chat.questionSelectionHint : copy.chat.questionAnswerHint}
        </span>

        <div className="question-actions">
          <button
            type="button"
            onClick={() => {
              ignorePromise(onCancel());
            }}
            disabled={loading}
            className="button-secondary"
          >
            {copy.chat.questionCancel}
          </button>
          <button
            type="button"
            onClick={() => {
              ignorePromise(submit());
            }}
            disabled={loading || !canSubmit}
            className="button"
          >
            {loading ? copy.chat.questionSubmitting : copy.chat.questionSubmit}
          </button>
        </div>
      </div>
    </div>
  );
};
