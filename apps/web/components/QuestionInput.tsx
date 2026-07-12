'use client';

import type { FC } from 'react';
import { useMemo, useState } from 'react';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { AskHumanOption } from '@/lib/types';

interface QuestionInputProps {
  loading: boolean;
  prompt: string;
  selectionMode?: 'single' | 'multiple';
  options?: AskHumanOption[];
  onAnswer: (answer: string) => Promise<void>;
  onCancel: () => Promise<void>;
}

function normalizeOptions(options?: AskHumanOption[]): AskHumanOption[] {
  return (options ?? [])
    .map((option) => ({ label: option.label.trim(), allow_custom: option.allow_custom }))
    .filter((option) => option.label.length > 0);
}

function buildChoiceAnswer(options: AskHumanOption[], selectedIndexes: number[], customText: string): string {
  const customAnswer = customText.trim();
  const answers = selectedIndexes
    .slice()
    .sort((left, right) => left - right)
    .flatMap((index) => {
      const option = options[index];
      if (!option) return [];
      return option.allow_custom ? (customAnswer ? [customAnswer] : []) : [option.label];
    });

  return answers.length === 1 ? answers[0] ?? '' : answers.length > 1 ? `- ${answers.join('\n- ')}` : '';
}

export const QuestionInput: FC<QuestionInputProps> = ({
  loading,
  prompt,
  selectionMode,
  options,
  onAnswer,
  onCancel,
}) => {
  const { copy } = useWebLocale();
  const normalizedOptions = useMemo(() => normalizeOptions(options), [options]);
  const isMultipleChoice = selectionMode === 'multiple';
  const [selectedIndexes, setSelectedIndexes] = useState<number[]>([]);
  const [customText, setCustomText] = useState('');
  const [showCustomInput, setShowCustomInput] = useState(normalizedOptions.length === 0);
  const customOptionIndex = normalizedOptions.findIndex((option) => option.allow_custom);
  const customSelected = customOptionIndex >= 0 && selectedIndexes.includes(customOptionIndex);
  const answer = buildChoiceAnswer(normalizedOptions, selectedIndexes, customText);
  const canSubmit = answer.length > 0 || (normalizedOptions.length === 0 && customText.trim().length > 0);

  async function submit(nextAnswer = answer) {
    if (loading || !nextAnswer.trim()) return;
    await onAnswer(nextAnswer);
  }

  function selectOption(index: number) {
    if (loading) return;
    const option = normalizedOptions[index];
    if (!option) return;

    if (isMultipleChoice) {
      setSelectedIndexes((current) => current.includes(index)
        ? current.filter((value) => value !== index)
        : [...current, index]);
      if (option.allow_custom) setShowCustomInput(true);
      return;
    }

    setSelectedIndexes([index]);
    if (option.allow_custom) {
      setShowCustomInput(true);
      return;
    }
    ignorePromise(submit(option.label));
  }

  return (
    <section className="plan-question-card" aria-label={copy.chat.questionCardAria}>
      <header className="plan-question-header">
        <h2>{prompt}</h2>
        <div className="plan-question-navigation" aria-label={copy.chat.questionProgressAria}>
          <button type="button" disabled aria-label={copy.chat.questionPreviousAria}><ChevronLeftIcon /></button>
          <span>1 / 1</span>
          <button type="button" disabled aria-label={copy.chat.questionNextAria}><ChevronRightIcon /></button>
          <i aria-hidden="true" />
          <button type="button" disabled={loading} onClick={() => ignorePromise(onCancel())} aria-label={copy.chat.questionCloseAria}><CloseIcon /></button>
        </div>
      </header>

      {normalizedOptions.length > 0 ? (
        <div className="plan-question-options" role={isMultipleChoice ? 'group' : 'radiogroup'}>
          {normalizedOptions.map((option, index) => {
            const selected = selectedIndexes.includes(index);
            return (
              <button
                key={`${option.label}-${index}`}
                type="button"
                className={`plan-question-option${selected ? ' is-selected' : ''}`}
                disabled={loading}
                role={isMultipleChoice ? 'checkbox' : 'radio'}
                aria-checked={selected}
                onClick={() => selectOption(index)}
              >
                <span className="plan-question-option-number">{index + 1}</span>
                <span className="plan-question-option-label">{option.label}</span>
                <ArrowRightIcon />
              </button>
            );
          })}
        </div>
      ) : null}

      {(showCustomInput || customSelected) ? (
        <textarea
          className="plan-question-custom-input"
          value={customText}
          disabled={loading}
          aria-label={copy.chat.questionCustomAnswerAria}
          placeholder={copy.chat.questionCustomPlaceholder}
          rows={3}
          onChange={(event) => setCustomText(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && !event.shiftKey && !isMultipleChoice) {
              event.preventDefault();
              ignorePromise(submit(normalizedOptions.length === 0 ? customText.trim() : answer));
            }
          }}
        />
      ) : null}

      <footer className="plan-question-footer">
        {customOptionIndex >= 0 && !showCustomInput ? (
          <button type="button" className="plan-question-feedback" disabled={loading} onClick={() => selectOption(customOptionIndex)}>
            <EditIcon />
            <span>{copy.chat.questionProvideFeedback}</span>
          </button>
        ) : <span />}
        {isMultipleChoice || normalizedOptions.length === 0 || showCustomInput ? (
          <button type="button" className="plan-question-submit" disabled={loading || !canSubmit} onClick={() => ignorePromise(submit(normalizedOptions.length === 0 ? customText.trim() : answer))}>
            {loading ? copy.chat.questionSubmitting : copy.chat.questionSubmit}
          </button>
        ) : (
          <button type="button" className="plan-question-skip" disabled={loading} onClick={() => ignorePromise(onCancel())}>{copy.chat.questionSkip}</button>
        )}
      </footer>
    </section>
  );
};

const ChevronLeftIcon = () => <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m15 18-6-6 6-6" /></svg>;
const ChevronRightIcon = () => <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m9 18 6-6-6-6" /></svg>;
const CloseIcon = () => <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m18 6-12 12M6 6l12 12" /></svg>;
const ArrowRightIcon = () => <svg className="plan-question-option-arrow" viewBox="0 0 24 24" aria-hidden="true"><path d="M5 12h14m-6-6 6 6-6 6" /></svg>;
const EditIcon = () => <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m4 20 4.2-1 10.5-10.5a2.1 2.1 0 0 0-3-3L5.2 16 4 20Zm10-13 3 3" /></svg>;
