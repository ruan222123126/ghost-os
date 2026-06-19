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
      <div className="settings-card-list">
        {Array.from({ length: TOOL_SKELETON_COUNT }).map((_, index) => (
          <div
            key={`tool-skeleton-${index}`}
            className="settings-card-skeleton animate-pulse"
          />
        ))}
      </div>
    );
  }

  if (tools.length === 0) {
    return (
      <div className="settings-list-empty">
        {copy.settings.toolsNoItems}
      </div>
    );
  }

  return (
    <>
      <div className="settings-card-list">
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
      className="settings-config-card is-clickable"
    >
      <div className="settings-config-card-main">
        <div className="settings-card-badges">
          <span className="settings-card-badge">
            {tool.enabled ? copy.settings.enabled : copy.settings.disabled}
          </span>
        </div>
        <p className="settings-card-title is-strong is-mono truncate">{tool.name}</p>
        <p className="settings-card-meta">
          <span className="font-medium text-[#111111]">{copy.settings.toolsPromptModeLabel}</span>
          : {hasPromptOverride(tool) ? copy.settings.toolsPromptModeCustom : copy.settings.toolsPromptModeDefault}
        </p>
      </div>

      <div className="settings-config-card-actions">
        <button
          type="button"
          disabled={controlsDisabled}
          onClick={(event) => {
            event.stopPropagation();
            onOpenDetails();
          }}
          className="settings-card-action whitespace-nowrap"
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
          className="settings-card-action whitespace-nowrap"
        >
          {tool.enabled ? copy.settings.tasksDisable : copy.settings.tasksEnable}
        </button>
      </div>
    </article>
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
