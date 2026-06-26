import type { ProviderConfig, ToolPayload } from '@/lib/types';
import { createEmptyWorkflowDraft } from '@/lib/workflow-editor/draft';
import {
  enabledWorkflowAgentToolNames,
  normalizeWorkflowDraftAgentNodes,
} from '@/lib/workflow-editor/agentRuntime';
import { validateWorkflowDraft } from '@/lib/workflow-editor/validation';

describe('lib/workflow-editor/agentRuntime', () => {
  it('fills legacy agent nodes with explicit enabled-tool snapshot', () => {
    const draft = createEmptyWorkflowDraft();
    draft.nodes = [
      draft.nodes[0],
      {
        id: 'agent-node',
        type: 'agent',
        position: { x: 180, y: 120 },
        ui: { toolArgumentsMode: 'kv' },
        agent: { message: 'run agent' },
      },
      draft.nodes[1],
    ];
    draft.edges = [
      { id: 'start-agent', from_node_id: 'start-node', to_node_id: 'agent-node' },
      { id: 'agent-end', from_node_id: 'agent-node', to_node_id: 'end-node' },
    ];

    const normalized = normalizeWorkflowDraftAgentNodes(
      draft,
      enabledWorkflowAgentToolNames([createTool('web_search', true), createTool('script_exec', true)]),
    );
    const agentNode = normalized.nodes.find((node) => node.id === 'agent-node');

    expect(agentNode?.agent?.runtime_overrides).toEqual({
      tool_allowlist_only: true,
      tool_allowlist: ['script_exec', 'web_search'],
    });
  });

  it('reuses the original draft reference when agent normalization makes no changes', () => {
    const draft = createEmptyWorkflowDraft();
    draft.nodes = [
      draft.nodes[0],
      {
        id: 'agent-node',
        type: 'agent',
        position: { x: 180, y: 120 },
        ui: { toolArgumentsMode: 'kv' },
        agent: {
          message: 'run agent',
          runtime_overrides: {
            tool_allowlist_only: true,
            tool_allowlist: ['script_exec'],
          },
        },
      },
      draft.nodes[1],
    ];
    draft.edges = [
      { id: 'start-agent', from_node_id: 'start-node', to_node_id: 'agent-node' },
      { id: 'agent-end', from_node_id: 'agent-node', to_node_id: 'end-node' },
    ];

    const normalized = normalizeWorkflowDraftAgentNodes(draft, ['script_exec']);

    expect(normalized).toBe(draft);
    expect(normalized.nodes).toBe(draft.nodes);
  });

  it('rejects provider-without-model and unavailable tools when catalog is loaded', () => {
    const draft = createEmptyWorkflowDraft();
    draft.nodes = [
      draft.nodes[0],
      {
        id: 'agent-node',
        type: 'agent',
        position: { x: 180, y: 120 },
        ui: { toolArgumentsMode: 'kv' },
        agent: {
          message: 'run agent',
          runtime_overrides: {
            provider_name: 'openai-main',
            tool_allowlist_only: true,
            tool_allowlist: ['ghost_tool'],
          },
        },
      },
      draft.nodes[1],
    ];
    draft.edges = [
      { id: 'start-agent', from_node_id: 'start-node', to_node_id: 'agent-node' },
      { id: 'agent-end', from_node_id: 'agent-node', to_node_id: 'end-node' },
    ];

    const result = validateWorkflowDraft(draft, {
      agentRuntimeCatalog: {
        activeProvider: 'openai-main',
        providers: [createProvider()],
        tools: [createTool('script_exec', true)],
      },
    });

    expect(result.valid).toBe(false);
    expect(result.errors).toEqual(expect.arrayContaining([
      'workflow agent node "agent-node" provider_name and model must be set together',
      'workflow agent node "agent-node" tool_allowlist contains disabled or unavailable tool "ghost_tool"',
    ]));
  });
});

function createProvider(): ProviderConfig {
  return {
    name: 'openai-main',
    type: 'openai',
    base_url: 'https://api.openai.com/v1',
    provider_id: 'openai-main',
    updated_at: '2026-01-01T00:00:00Z',
    models: ['gpt-5.4'],
    api_key_set: true,
  };
}

function createTool(name: string, enabled: boolean): ToolPayload {
  return {
    name,
    enabled,
  };
}
