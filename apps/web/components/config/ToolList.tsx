'use client';

import { useEffect, useMemo, useState } from 'react';
import { ToolPromptEditor } from '@/components/config/ToolPromptEditor';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ToolPayload, ToolUpdateRequest } from '@/lib/types';

const TOOL_SKELETON_COUNT = 4;

interface ToolListProps {
  tools: ToolPayload[];
  loading: boolean;
  controlsDisabled: boolean;
  onUpdate: (name: string, input: ToolUpdateRequest) => Promise<void>;
}

interface ToolCardProps {
  tool: ToolPayload;
  controlsDisabled: boolean;
  onToggle: () => void;
  onOpenDetails: () => void;
}

export function ToolList(props: ToolListProps) {
  const { copy } = useWebLocale();
  const { tools, loading, controlsDisabled, onUpdate } = props;
  const { activeTool, openTool, closeTool } = useActiveTool(tools);
  const orderedTools = useMemo(() => prioritizeEnabledTools(tools), [tools]);

  if (loading) {
    return (
      <div className="grid grid-cols-1 gap-3">
        {Array.from({ length: TOOL_SKELETON_COUNT }).map((_, index) => (
          <div
            key={`tool-skeleton-${index}`}
            className="h-[128px] animate-pulse rounded-[16px] border border-[#E5E5E5] bg-[#FAFAFA]"
          />
        ))}
      </div>
    );
  }

  if (tools.length === 0) {
    return (
      <div className="rounded-[16px] border border-[#E5E5E5] bg-white px-6 py-8 text-center text-[13px] text-[#737373]">
        {copy.settings.toolsNoItems}
      </div>
    );
  }

  return (
    <>
      <div className="grid grid-cols-1 gap-3">
        {orderedTools.map((tool) => (
          <ToolCard
            key={tool.name}
            tool={tool}
            controlsDisabled={controlsDisabled}
            onOpenDetails={() => openTool(tool.name)}
            onToggle={() => ignorePromise(onUpdate(tool.name, { enabled: !tool.enabled }))}
          />
        ))}
      </div>

      {activeTool ? (
        <ToolPromptEditor
          tool={activeTool}
          controlsDisabled={controlsDisabled}
          onUpdate={onUpdate}
          onClose={closeTool}
        />
      ) : null}
    </>
  );
}

function useActiveTool(tools: ToolPayload[]) {
  const [activeToolName, setActiveToolName] = useState<string | null>(null);

  useEffect(() => {
    if (activeToolName === null) {
      return;
    }
    if (!tools.some((item) => item.name === activeToolName)) {
      setActiveToolName(null);
    }
  }, [activeToolName, tools]);

  const activeTool = useMemo(() => {
    if (activeToolName === null) {
      return null;
    }
    return tools.find((item) => item.name === activeToolName) ?? null;
  }, [activeToolName, tools]);

  return {
    activeTool,
    openTool: setActiveToolName,
    closeTool: () => setActiveToolName(null),
  };
}

function ToolCard(props: ToolCardProps) {
  const { copy } = useWebLocale();
  const { tool, controlsDisabled, onToggle, onOpenDetails } = props;

  return (
    <article
      role="button"
      tabIndex={0}
      onClick={onOpenDetails}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault();
          onOpenDetails();
        }
      }}
      className="rounded-[16px] border border-[#E5E5E5] bg-white p-5 transition-all hover:border-[#111111] focus:outline-none focus:ring-2 focus:ring-[#111111]/20"
    >
      <ToolCardHeader
        name={tool.name}
        enabled={tool.enabled}
        controlsDisabled={controlsDisabled}
        onToggle={onToggle}
        onEdit={onOpenDetails}
      />
      <p className="text-[12px] text-[#737373]">
        <span className="font-medium text-[#111111]">{copy.settings.toolsPromptModeLabel}</span>
        : {hasPromptOverride(tool) ? copy.settings.toolsPromptModeCustom : copy.settings.toolsPromptModeDefault}
      </p>
    </article>
  );
}

function ToolCardHeader(props: {
  name: string;
  enabled: boolean;
  controlsDisabled: boolean;
  onToggle: () => void;
  onEdit: () => void;
}) {
  const { copy } = useWebLocale();
  const { name, enabled, controlsDisabled, onToggle, onEdit } = props;

  return (
    <div className="mb-3 flex items-center justify-between gap-4">
      <div className="min-w-0">
        <div className="mb-1 flex items-center gap-2">
          <span className="inline-flex rounded-full bg-[#F5F5F5] px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-[#111111]">
            {enabled ? copy.settings.enabled : copy.settings.disabled}
          </span>
        </div>
        <p className="truncate font-mono text-[13px] text-[#111111]">{name}</p>
      </div>

      <div className="flex items-center gap-2">
        <button
          type="button"
          disabled={controlsDisabled}
          onClick={(event) => {
            event.stopPropagation();
            onEdit();
          }}
          className="rounded-full border border-[#E5E5E5] px-3 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {copy.settings.toolsEditPrompt}
        </button>
        <button
          type="button"
          disabled={controlsDisabled}
          onClick={(event) => {
            event.stopPropagation();
            onToggle();
          }}
          className="rounded-full border border-[#E5E5E5] px-3 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {enabled ? copy.settings.tasksDisable : copy.settings.tasksEnable}
        </button>
      </div>
    </div>
  );
}

function prioritizeEnabledTools(tools: ToolPayload[]): ToolPayload[] {
  const enabledTools: ToolPayload[] = [];
  const disabledTools: ToolPayload[] = [];

  for (const tool of tools) {
    if (tool.enabled) {
      enabledTools.push(tool);
      continue;
    }
    disabledTools.push(tool);
  }
  return [...enabledTools, ...disabledTools];
}

function hasPromptOverride(tool: ToolPayload): boolean {
  return (tool.prompt_override ?? '').trim() !== '';
}
