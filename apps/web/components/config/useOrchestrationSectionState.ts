'use client';

import { useEffect, useState } from 'react';
import { toErrorMessage } from '@/lib/errors';
import type { WebCopy } from '@/lib/i18n/messages';
import {
  createOrchestration,
  listOrchestrationSummaries,
  type OrchestrationSummary,
} from '@/lib/orchestrationStore';

export function useOrchestrationSectionState(copy: WebCopy) {
  const [orchestrations, setOrchestrations] = useState<OrchestrationSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    try {
      setOrchestrations(listOrchestrationSummaries());
      setError('');
    } catch (nextError) {
      setError(toErrorMessage(nextError, copy.system.failedToLoadOrchestration));
    } finally {
      setLoading(false);
    }
  }, [copy.system.failedToLoadOrchestration]);

  const handleCancelCreate = () => {
    setCreating(false);
    setName('');
    setError('');
  };

  return {
    orchestrations,
    loading,
    error,
    creating,
    name,
    submitting,
    controlsDisabled: loading || submitting,
    startCreate: () => { setCreating(true); setError(''); },
    setName,
    submitCreate: async () => submitCreate({ copy, name, setError, setOrchestrations, setSubmitting, onSuccess: handleCancelCreate }),
    cancelCreate: handleCancelCreate,
  };
}

async function submitCreate(input: {
  copy: WebCopy;
  name: string;
  setError: (value: string) => void;
  setOrchestrations: (value: OrchestrationSummary[]) => void;
  setSubmitting: (value: boolean) => void;
  onSuccess: () => void;
}) {
  const { copy, name, setError, setOrchestrations, setSubmitting, onSuccess } = input;
  if (name.trim().length === 0) {
    setError(copy.settings.orchestrationNameRequired);
    return;
  }

  setSubmitting(true);
  setError('');
  try {
    createOrchestration({ name });
    setOrchestrations(listOrchestrationSummaries());
    onSuccess();
  } catch (nextError) {
    setError(toErrorMessage(nextError, copy.system.failedToCreateOrchestration));
  } finally {
    setSubmitting(false);
  }
}
