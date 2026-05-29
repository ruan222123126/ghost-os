'use client';

import { useEffect, useMemo, useState } from 'react';
import { CloseButton } from '@/components/CloseButton';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ToolPayload, ToolUpdateRequest } from '@/lib/types';

const SCRIPT_EXEC_TOOL = 'script_exec';
const MAX_SANDBOX_MEMORY_MB = 512;

interface ToolPromptEditorProps {
  tool: ToolPayload;
  controlsDisabled: boolean;
  onUpdate: (name: string, input: ToolUpdateRequest) => Promise<void>;
  onClose: () => void;
}

export function ToolPromptEditor(props: ToolPromptEditorProps) {
  const { copy } = useWebLocale();
  const { tool, controlsDisabled, onUpdate, onClose } = props;
  const [promptDraft, setPromptDraft] = useState(tool.prompt_override ?? '');
  const [sandboxMemoryDraft, setSandboxMemoryDraft] = useState(toSandboxMemoryDraft(tool.sandbox_memory_mb));

  useEffect(() => {
    setPromptDraft(tool.prompt_override ?? '');
    setSandboxMemoryDraft(toSandboxMemoryDraft(tool.sandbox_memory_mb));
  }, [tool.name, tool.prompt_override, tool.sandbox_memory_mb]);

  const savedPrompt = tool.prompt_override ?? '';
  const savedSandboxMemory = toSandboxMemoryDraft(tool.sandbox_memory_mb);
  const promptDirty = promptDraft !== savedPrompt;
  const sandboxMemoryDirty = sandboxMemoryDraft !== savedSandboxMemory;
  const sandboxMemoryError = useMemo(
    () => scriptExecSandboxMemoryError(tool.name, sandboxMemoryDraft, copy.settings.toolsSandboxMemoryInvalid),
    [copy.settings.toolsSandboxMemoryInvalid, sandboxMemoryDraft, tool.name],
  );
  const dirty = promptDirty || sandboxMemoryDirty;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 px-4 py-6" onClick={onClose}>
      <div
        className="w-full max-w-3xl rounded-[20px] border border-[#E5E5E5] bg-white p-6 shadow-2xl"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="mb-3 flex items-center justify-between gap-3">
          <div className="min-w-0">
            <p className="text-[12px] text-[#737373]">{copy.settings.toolsPromptOverrideLabel}</p>
            <p className="truncate font-mono text-[15px] text-[#111111]">{tool.name}</p>
          </div>
          <CloseButton
            onClick={onClose}
            className="shrink-0"
            aria-label={copy.settings.cancel}
          />
        </div>

        <textarea
          value={promptDraft}
          onChange={(event) => setPromptDraft(event.target.value)}
          placeholder={copy.settings.toolsPromptOverridePlaceholder}
          disabled={controlsDisabled}
          rows={12}
          className="mb-4 w-full rounded-[12px] border border-[#E5E5E5] bg-white px-3 py-2 text-[13px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none disabled:opacity-60"
        />

        {tool.name === SCRIPT_EXEC_TOOL ? (
          <div className="mb-4">
            <p className="mb-1 text-[12px] font-medium text-[#111111]">{copy.settings.toolsSandboxMemoryLabel}</p>
            <p className="mb-2 text-[12px] text-[#737373]">{copy.settings.toolsSandboxMemoryDescription}</p>
            <input
              type="number"
              min={1}
              max={MAX_SANDBOX_MEMORY_MB}
              step={1}
              value={sandboxMemoryDraft}
              disabled={controlsDisabled}
              onChange={(event) => setSandboxMemoryDraft(event.target.value)}
              className="w-full rounded-[12px] border border-[#E5E5E5] bg-white px-3 py-2 text-[13px] text-[#111111] transition-colors focus:border-[#111111] focus:outline-none disabled:opacity-60"
            />
            {sandboxMemoryError ? <p className="mt-2 text-[12px] text-[#B42318]">{sandboxMemoryError}</p> : null}
          </div>
        ) : null}

        <div className="flex items-center gap-2">
          <button
            type="button"
            disabled={controlsDisabled || !dirty || !!sandboxMemoryError}
            onClick={() => ignorePromise(onUpdate(tool.name, buildToolUpdateRequest(promptDraft, sandboxMemoryDraft, tool.name, promptDirty, sandboxMemoryDirty)))}
            className="rounded-full bg-[#111111] px-4 py-1.5 text-[12px] font-medium text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.save}
          </button>
          <button
            type="button"
            disabled={controlsDisabled || !dirty}
            onClick={() => {
              setPromptDraft(savedPrompt);
              setSandboxMemoryDraft(savedSandboxMemory);
            }}
            className="rounded-full border border-[#E5E5E5] px-4 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.toolsResetChanges}
          </button>
        </div>
      </div>
    </div>
  );
}

function buildToolUpdateRequest(
  promptDraft: string,
  sandboxMemoryDraft: string,
  toolName: string,
  promptDirty: boolean,
  sandboxMemoryDirty: boolean,
): ToolUpdateRequest {
  const next: ToolUpdateRequest = {};
  if (promptDirty) {
    next.prompt_override = promptDraft;
  }
  if (toolName === SCRIPT_EXEC_TOOL && sandboxMemoryDirty) {
    next.sandbox_memory_mb = Number.parseInt(sandboxMemoryDraft, 10);
  }
  return next;
}

function scriptExecSandboxMemoryError(
  toolName: string,
  draft: string,
  template: (max: number) => string,
): string {
  if (toolName !== SCRIPT_EXEC_TOOL) {
    return '';
  }
  const trimmed = draft.trim();
  const value = Number.parseInt(trimmed, 10);
  if (!trimmed || !Number.isInteger(value) || value < 1 || value > MAX_SANDBOX_MEMORY_MB) {
    return template(MAX_SANDBOX_MEMORY_MB);
  }
  return '';
}

function toSandboxMemoryDraft(value: number | undefined): string {
  return value === undefined ? '' : String(value);
}
