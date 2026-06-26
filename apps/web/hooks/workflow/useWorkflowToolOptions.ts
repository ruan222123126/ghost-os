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

interface WorkflowToolCatalogState {
  error: string;
  loading: boolean;
  tools: ToolPayload[];
}

interface WorkflowToolOptionLabels {
  disabled: (name: string) => string;
  unavailable: (name: string) => string;
}

interface WorkflowToolCatalogLoadResult {
  error?: string;
  tools?: ToolPayload[];
}

interface WorkflowToolCatalogSetters {
  setError: (value: string) => void;
  setTools: (value: ToolPayload[]) => void;
}

interface EnabledToolOptions {
  names: Set<string>;
  options: WorkflowToolOption[];
}

export function useWorkflowToolOptions(currentToolName: string): UseWorkflowToolOptionsResult {
  const { copy } = useWebLocale();
  const catalog = useWorkflowToolCatalog(copy.system.failedToLoadTools);

  return {
    options: useMemo(
      () => buildWorkflowToolOptions({
        tools: catalog.tools,
        currentToolName,
        labels: {
          disabled: copy.workflow.toolNameDisabled,
          unavailable: copy.workflow.toolNameUnavailable,
        },
      }),
      [catalog.tools, copy.workflow.toolNameDisabled, copy.workflow.toolNameUnavailable, currentToolName],
    ),
    loading: catalog.loading,
    error: catalog.error,
  };
}

function useWorkflowToolCatalog(fallbackError: string): WorkflowToolCatalogState {
  const [tools, setTools] = useState<ToolPayload[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError('');
    void loadWorkflowToolCatalog(fallbackError).then((result) => {
      if (cancelled) {
        return;
      }
      applyWorkflowToolCatalogResult(result, { setTools, setError });
      setLoading(false);
    });
    return () => {
      cancelled = true;
    };
  }, [fallbackError]);

  return {
    error,
    loading,
    tools,
  };
}

async function loadWorkflowToolCatalog(fallbackError: string): Promise<WorkflowToolCatalogLoadResult> {
  try {
    return { tools: await listTools() };
  } catch (cause) {
    return { error: toErrorMessage(cause, fallbackError) };
  }
}

function applyWorkflowToolCatalogResult(
  result: WorkflowToolCatalogLoadResult,
  setters: WorkflowToolCatalogSetters,
): void {
  if (result.tools !== undefined) {
    setters.setTools(result.tools);
  }
  if (result.error !== undefined) {
    setters.setError(result.error);
  }
}

export function buildWorkflowToolOptions(options: {
  tools: ToolPayload[];
  currentToolName: string;
  labels: WorkflowToolOptionLabels;
}): WorkflowToolOption[] {
  const enabledTools = buildEnabledToolOptions(options.tools);
  const normalizedCurrent = normalizeToolName(options.currentToolName);
  if (!shouldPrependCurrentTool(normalizedCurrent, enabledTools.names)) {
    return enabledTools.options;
  }
  return [
    buildUnavailableCurrentToolOption(normalizedCurrent, options.tools, options.labels),
    ...enabledTools.options,
  ];
}

function buildEnabledToolOptions(tools: ToolPayload[]): EnabledToolOptions {
  const orderedTools = tools
    .filter((tool) => tool.enabled)
    .sort((left, right) => left.name.localeCompare(right.name));
  return {
    names: new Set(orderedTools.map((tool) => tool.name)),
    options: orderedTools.map(buildEnabledToolOption),
  };
}

function buildEnabledToolOption(tool: ToolPayload): WorkflowToolOption {
  return {
    name: tool.name,
    label: tool.name,
    disabled: false,
    inputSchema: tool.input_schema,
  };
}

function buildUnavailableCurrentToolOption(
  toolName: string,
  tools: ToolPayload[],
  labels: WorkflowToolOptionLabels,
): WorkflowToolOption {
  const label = hasDisabledTool(tools, toolName) ? labels.disabled(toolName) : labels.unavailable(toolName);
  return {
    name: toolName,
    label,
    disabled: true,
    inputSchema: undefined,
  };
}

function shouldPrependCurrentTool(currentToolName: string, enabledToolNames: Set<string>): boolean {
  return currentToolName.length > 0 && !enabledToolNames.has(currentToolName);
}

function hasDisabledTool(tools: ToolPayload[], toolName: string): boolean {
  return tools.some((tool) => tool.name === toolName && !tool.enabled);
}

function normalizeToolName(toolName: string): string {
  return toolName.trim();
}
