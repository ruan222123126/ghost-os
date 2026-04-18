'use client';

import { useEffect, useMemo, useState } from 'react';
import { listTools } from '@/lib/api/tools/api';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ToolPayload } from '@/lib/types';

export interface WorkflowToolOption {
  name: string;
  label: string;
  disabled: boolean;
  inputSchema?: Record<string, unknown>;
}

interface UseWorkflowToolOptionsResult {
  options: WorkflowToolOption[];
  loading: boolean;
  error: string;
}

export function useWorkflowToolOptions(currentToolName: string): UseWorkflowToolOptionsResult {
  const { copy } = useWebLocale();
  const [tools, setTools] = useState<ToolPayload[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      setLoading(true);
      setError('');
      try {
        const payload = await listTools();
        if (!cancelled) {
          setTools(payload);
        }
      } catch (cause) {
        if (!cancelled) {
          setError(toErrorMessage(cause, copy.system.failedToLoadTools));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [copy.system.failedToLoadTools]);

  return {
    options: useMemo(
      () => buildWorkflowToolOptions({
        tools,
        currentToolName,
        disabledLabel: copy.workflow.toolNameDisabled,
        unavailableLabel: copy.workflow.toolNameUnavailable,
      }),
      [copy.workflow.toolNameDisabled, copy.workflow.toolNameUnavailable, currentToolName, tools],
    ),
    loading,
    error,
  };
}

function buildWorkflowToolOptions(options: {
  tools: ToolPayload[];
  currentToolName: string;
  disabledLabel: (name: string) => string;
  unavailableLabel: (name: string) => string;
}): WorkflowToolOption[] {
  const { tools, currentToolName, disabledLabel, unavailableLabel } = options;
  const orderedTools = tools
    .filter((tool) => tool.enabled)
    .sort((left, right) => left.name.localeCompare(right.name));
  const names = new Set<string>();
  const mapped = orderedTools.map((tool) => {
    names.add(tool.name);
    return {
      name: tool.name,
      label: tool.name,
      disabled: false,
      inputSchema: tool.input_schema,
    };
  });

  const normalizedCurrent = currentToolName.trim();
  if (normalizedCurrent.length === 0 || names.has(normalizedCurrent)) {
    return mapped;
  }

  const disabledTool = tools.find((tool) => tool.name === normalizedCurrent && !tool.enabled);
  const missingLabel = disabledTool ? disabledLabel(normalizedCurrent) : unavailableLabel(normalizedCurrent);
  return [
    {
      name: normalizedCurrent,
      label: missingLabel,
      disabled: true,
      inputSchema: undefined,
    },
    ...mapped,
  ];
}
