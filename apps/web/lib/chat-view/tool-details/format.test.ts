import type { ToolChatMessage } from '@/lib/types';
import { formatToolAction, formatToolCardDetails, formatToolDetails } from './format';

describe('lib/chat-view/tool-details/format', () => {
  it('aggregates script_exec report actions with readable categories and fixed order', () => {
    const report = {
      summary: { step_count: 8, failed_steps: 0, write_steps: 2 },
      steps: [
        { tool: 'read_file', args: { path: '/tmp/a.ts' } },
        { tool: 'tools.read', args: { path: 'C:\\repo\\b.ts' } },
        { tool: 'read_file', args: { path: '/tmp/c.ts' } },
        {
          tool: 'write_file',
          args: { path: '/tmp/out.txt', content_summary: { lines: 4 } },
          write_change: { operation: 'write_file', path: '/tmp/out.txt', content_lines: 4 },
        },
        {
          tool: 'apply_diff',
          args: { path: '/tmp/app.ts', diff_summary: { added_lines: 3, removed_lines: 1 } },
          write_change: { operation: 'apply_diff', path: '/tmp/app.ts', added_lines: 3, removed_lines: 1 },
        },
        { tool: 'bash_exec', args: { command: 'pnpm test --filter web' } },
        { tool: 'fetch_webpage', args: { url: 'https://example.com/docs?x=1' } },
        { tool: 'search_files', args: { query: 'formatToolDetails' } },
      ],
    };

    const details = formatToolDetails(buildToolMessage({
      toolName: 'script_exec',
      rawOutput: JSON.stringify(report),
    }));

    expect(readSummaryLine(details)).toBe(
      'summary: read /tmp/a.ts, C:/repo/b.ts +1 | write /tmp/out.txt +4 | edit /tmp/app.ts +3 -1 | run pnpm test --filter web | web example.com/docs?x=1 | search formatToolDetails',
    );
  });

  it('extracts helper calls from script_exec streaming args when report is unavailable', () => {
    const details = formatToolDetails(buildToolMessage({
      toolName: 'script_exec',
      content: JSON.stringify({
        script: [
          "read_file(path='/tmp/main.go')",
          "search_files(query='needle', path='.')",
          "bash_exec(command='echo ok')",
        ].join('\n'),
      }),
    }));

    expect(readSummaryLine(details)).toBe('summary: read /tmp/main.go | run echo ok | search needle');
  });

  it('keeps parsing legacy tools helper syntax in summaries for historical logs', () => {
    const details = formatToolDetails(buildToolMessage({
      toolName: 'script_exec',
      content: JSON.stringify({
        script: [
          "tools.read_file(path='/tmp/main.go')",
          "tools.search_files(query='needle', path='.')",
          "tools.bash_exec(command='echo ok')",
        ].join('\n'),
      }),
    }));

    expect(readSummaryLine(details)).toBe('summary: read /tmp/main.go | run echo ok | search needle');
  });

  it('shows line deltas only when data is available for write/edit actions', () => {
    const writeDetails = formatToolDetails(buildToolMessage({
      toolName: 'write_file',
      content: JSON.stringify({
        path: '/tmp/new.txt',
        content_summary: { lines: 5 },
      }),
    }));

    const editDetails = formatToolDetails(buildToolMessage({
      toolName: 'apply_diff',
      content: JSON.stringify({
        path: '/tmp/new.txt',
      }),
    }));
    const editWithDiffDetails = formatToolDetails(buildToolMessage({
      toolName: 'apply_diff',
      content: JSON.stringify({
        path: '/tmp/edit.txt',
        diff_text: '@@ -1,3 +1,4 @@\n alpha\n-beta\n+beta2\n+delta\n gamma\n',
      }),
    }));

    expect(readSummaryLine(writeDetails)).toBe('summary: write /tmp/new.txt +5');
    expect(readSummaryLine(editDetails)).toBe('summary: edit /tmp/new.txt');
    expect(readSummaryLine(editWithDiffDetails)).toBe('summary: edit /tmp/edit.txt +2 -1');
    expect(writeDetails).not.toContain('+0');
    expect(writeDetails).not.toContain('-0');
    expect(editDetails).not.toContain('+0');
    expect(editDetails).not.toContain('-0');
  });

  it('preserves normalized full paths for file tool summaries', () => {
    const listDetails = formatToolDetails(buildToolMessage({
      toolName: 'list_files',
      content: JSON.stringify({ path: '/' }),
    }));

    const writeDetails = formatToolDetails(buildToolMessage({
      toolName: 'write_file',
      content: JSON.stringify({
        path: 'C:\\repo\\new.txt\\',
        content_summary: { lines: 1 },
      }),
    }));

    expect(readSummaryLine(listDetails)).toBe('summary: list /');
    expect(readSummaryLine(writeDetails)).toBe('summary: write C:/repo/new.txt +1');
  });

  it('uses tool call arguments for finished list_files summaries and compact steps', () => {
    const tool = buildToolMessage({
      toolName: 'list_files',
      toolCallId: 'call-list-1',
      toolCalls: [
        {
          id: 'call-list-1',
          name: 'list_files',
          arguments: { path: '/media/ruan/Files/Game/main' },
        },
      ],
      content: '["README.md","src/"]',
      rawOutput: '["README.md","src/"]',
    });

    expect(readSummaryLine(formatToolDetails(tool))).toBe('summary: list /media/ruan/Files/Game/main');
    expect(formatToolDetails(tool, { compactOutputEnabled: true })).toBe('step1: list /media/ruan/Files/Game/main');
  });

  it('keeps promoted file action title after streaming output replaces argument payload', () => {
    const tool = buildToolMessage({
      toolName: 'read_file',
      toolInput: JSON.stringify({ path: '/tmp/main.go' }),
      content: 'package main',
      rawOutput: 'package main',
    });

    expect(formatToolAction(tool)).toEqual({
      actionKind: 'read',
      actionText: '/tmp/main.go',
      text: 'read /tmp/main.go',
      variant: 'action',
    });
  });

  it('promotes screen_control screenshot actions into a readable title action', () => {
    const tool = buildToolMessage({
      toolName: 'screen_control',
      toolInput: JSON.stringify({ action: 'screenshot', display_id: 67 }),
      content: JSON.stringify({ action: 'screenshot', artifact: { type: 'image' } }),
    });

    expect(formatToolAction(tool)).toEqual({
      actionKind: 'screen',
      actionText: '',
      text: 'screen',
      variant: 'action',
    });
  });

  it('truncates long run commands with a stable readable prefix', () => {
    const longCommand = `echo ${'x'.repeat(120)}`;
    const details = formatToolDetails(buildToolMessage({
      toolName: 'bash_exec',
      content: JSON.stringify({ command: longCommand }),
    }));

    const summary = readSummaryLine(details);
    expect(summary.startsWith('summary: run echo ')).toBe(true);
    expect(summary.endsWith('...')).toBe(true);
  });

  it('shows bash_exec command in the card title and reduces compact details to status plus output preview', () => {
    const tool = buildToolMessage({
      toolName: 'bash_exec',
      toolInput: JSON.stringify({ command: 'echo ok' }),
      content: '/home/ruan/.nvm/versions/node/v24.14.0/bin/agent-browser\nagent-browser 0.26.0\nextra',
      rawOutput: '/home/ruan/.nvm/versions/node/v24.14.0/bin/agent-browser\nagent-browser 0.26.0\nextra',
    });

    expect(formatToolAction(tool)).toEqual({ text: 'echo ok', variant: 'command' });
    expect(formatToolCardDetails(tool, { compactOutputEnabled: true })).toBe(
      'success\n/home/ruan/.nvm/versions/node/v24.14.0/bin/agent-browser\nagent-browser 0.26.0',
    );
  });

  it('uses tool call arguments for bash_exec title after history-style output-only hydration', () => {
    const tool = buildToolMessage({
      toolName: 'bash_exec',
      toolCallId: 'call-bash-1',
      toolCalls: [
        {
          id: 'call-bash-1',
          name: 'bash_exec',
          arguments: { command: 'which agent-browser' },
        },
      ],
      content: '/home/ruan/.nvm/versions/node/v24.14.0/bin/agent-browser',
      rawOutput: '/home/ruan/.nvm/versions/node/v24.14.0/bin/agent-browser',
    });

    expect(formatToolAction(tool)).toEqual({ text: 'which agent-browser', variant: 'command' });
  });

  it('uses tool call arguments for codex_exec title after history-style output-only hydration', () => {
    const tool = buildToolMessage({
      toolName: 'codex_exec',
      toolCallId: 'call-codex-exec-1',
      toolCalls: [
        {
          id: 'call-codex-exec-1',
          name: 'codex_exec',
          arguments: { command: 'rg --files -g README.md' },
        },
      ],
      content: 'apps/android/README.md',
      rawOutput: 'apps/android/README.md',
    });

    expect(formatToolAction(tool)).toEqual({ text: 'rg --files -g README.md', variant: 'command' });
  });

  it('promotes codex patch targets into readable edit actions', () => {
    expect(formatToolAction(buildToolMessage({
      toolName: 'codex_patch',
      toolInput: JSON.stringify({
        changes: [
          { path: '/tmp/main.go' },
          { new_path: '/tmp/app.ts' },
          'README.md',
        ],
      }),
      content: 'completed',
    }))).toEqual({
      actionKind: 'edit',
      actionText: '/tmp/main.go, /tmp/app.ts +1',
      text: 'edit /tmp/main.go, /tmp/app.ts +1',
      variant: 'action',
    });
  });

  it('includes search query in summary for search_files', () => {
    const details = formatToolDetails(buildToolMessage({
      toolName: 'search_files',
      content: JSON.stringify({ query: 'bridge runtime selector' }),
    }));

    expect(readSummaryLine(details)).toBe('summary: search bridge runtime selector');
  });

  it('promotes web search and codex actions into the tool title', () => {
    expect(formatToolAction(buildToolMessage({
      toolName: 'web_search',
      toolInput: JSON.stringify({ query: 'Ghost OS release notes' }),
      content: JSON.stringify([{ title: 'Ghost OS' }]),
    }))).toEqual({
      actionKind: 'web',
      actionText: 'Ghost OS release notes',
      text: 'web Ghost OS release notes',
      variant: 'action',
    });

    expect(formatToolAction(buildToolMessage({
      toolName: 'codex_cli',
      toolInput: JSON.stringify({ op: 'status', session_id: 'cmd-123' }),
      content: JSON.stringify({ status: 'running' }),
    }))).toEqual({
      actionKind: 'codex',
      actionText: 'status',
      text: 'codex status',
      variant: 'action',
    });
  });

  it('shows sfind load, unload, and list actions with skill names', () => {
    expect(readSummaryLine(formatToolDetails(buildToolMessage({
      toolName: 'sfind',
      rawOutput: JSON.stringify({ action: 'load', items: [{ name: 'release_flow' }, { name: 'ship_checklist' }] }),
    })))).toBe('summary: load release_flow, ship_checklist');
    expect(readSummaryLine(formatToolDetails(buildToolMessage({
      toolName: 'sfind',
      rawOutput: JSON.stringify({ action: 'unload', items: [{ name: 'release_flow' }] }),
    })))).toBe('summary: unload release_flow');
    expect(readSummaryLine(formatToolDetails(buildToolMessage({
      toolName: 'sfind',
      rawOutput: JSON.stringify({ action: 'list', items: [{ name: 'release_flow', status: 'active' }, { name: 'ship_checklist', status: 'pending' }] }),
    })))).toBe('summary: list release_flow(active), ship_checklist(pending)');
    expect(formatToolDetails(buildToolMessage({
      toolName: 'sfind',
      rawOutput: JSON.stringify({ action: 'load', items: [{ name: 'release_flow' }] }),
    }), { compactOutputEnabled: true })).toBe('step1: load release_flow');
  });

  it('promotes file and sfind atomic actions into the tool title', () => {
    expect(formatToolAction(buildToolMessage({
      toolName: 'read_file',
      content: JSON.stringify({ path: '/tmp/main.go' }),
    }))).toEqual({
      actionKind: 'read',
      actionText: '/tmp/main.go',
      text: 'read /tmp/main.go',
      variant: 'action',
    });

    expect(formatToolAction(buildToolMessage({
      toolName: 'write_file',
      content: JSON.stringify({ path: '/tmp/out.txt', content_summary: { lines: 4 } }),
    }))).toEqual({
      actionKind: 'write',
      actionText: '/tmp/out.txt +4',
      text: 'write /tmp/out.txt +4',
      variant: 'action',
    });

    expect(formatToolAction(buildToolMessage({
      toolName: 'apply_diff',
      toolInput: JSON.stringify({
        path: '/tmp/edit.txt',
        diff_text: '@@ -1,2 +1,3 @@\n-old\n+new\n+extra\n same\n',
      }),
      content: 'applied 1 hunks to /tmp/edit.txt',
    }))).toEqual({
      actionKind: 'edit',
      actionText: '/tmp/edit.txt +2 -1',
      text: 'edit /tmp/edit.txt +2 -1',
      variant: 'action',
    });

    expect(formatToolAction(buildToolMessage({
      toolName: 'sfind',
      rawOutput: JSON.stringify({ action: 'load', items: [{ name: 'release_flow' }] }),
    }))).toEqual({
      actionKind: 'load',
      actionText: 'release_flow',
      text: 'load release_flow',
      variant: 'action',
    });
  });

  it('shows action-title tool details without repeating summary, steps, or transport metadata', () => {
    const readTool = buildToolMessage({
      toolName: 'read_file',
      toolStatus: 'success',
      traceId: 'trace-read-1',
      content: 'package main\nfunc main() {}',
      toolCalls: [
        {
          id: 'call-read-1',
          name: 'read_file',
          arguments: { path: '/tmp/main.go' },
        },
      ],
      toolCallId: 'call-read-1',
    });

    expect(formatToolCardDetails(readTool)).toBe('package main\nfunc main() {}');
    expect(formatToolCardDetails(readTool)).not.toContain('summary:');
    expect(formatToolCardDetails(readTool)).not.toContain('status:');
    expect(formatToolCardDetails(readTool)).not.toContain('trace_id:');
    expect(formatToolCardDetails(readTool)).not.toContain('tool_call_id:');
    expect(formatToolCardDetails(readTool, { compactOutputEnabled: true })).toBe(
      'success\npackage main\nfunc main() {}',
    );

    const sfindTool = buildToolMessage({
      toolName: 'sfind',
      toolStatus: 'success',
      rawOutput: JSON.stringify({ action: 'load', items: [{ name: 'release_flow' }] }),
      toolCalls: [
        {
          id: 'call-sfind-1',
          name: 'sfind',
          arguments: { action: 'load', skill_names: ['release_flow'] },
        },
      ],
      toolCallId: 'call-sfind-1',
    });

    expect(formatToolCardDetails(sfindTool)).toBe('{"action":"load","items":[{"name":"release_flow"}]}');
    expect(formatToolCardDetails(sfindTool, { compactOutputEnabled: true })).toBe('success');
  });

  it('shows failure error line before summary without transport metadata', () => {
    const details = formatToolDetails(buildToolMessage({
      toolName: 'read_file',
      toolStatus: 'failed',
      content: 'permission denied',
      traceId: 'trace-err-1',
    }));

    const lines = details.split('\n');
    expect(lines[0]).toBe('error: permission denied');
    expect(lines[1]).toBe('summary: read');
    expect(details).not.toContain('status:');
    expect(details).not.toContain('trace_id:');
  });

  it('keeps original detail text when summary parsing fails', () => {
    const details = formatToolDetails(buildToolMessage({
      toolName: 'script_exec',
      content: '{"script":',
      rawOutput: '{"summary":1,,}',
    }));

    expect(readSummaryLine(details)).toBe('summary: run');
    expect(details).toContain('{"script":');
    expect(details).toContain('{"summary":1,,}');
  });

  it('falls back to generic run summary when script_exec has lifecycle text only', () => {
    const details = formatToolDetails(buildToolMessage({
      toolName: 'script_exec',
      toolStatus: 'success',
      content: 'script_exec finished',
    }));

    expect(readSummaryLine(details)).toBe('summary: run');
    expect(details).toContain('script_exec finished');
  });

  it('renders compact details as ordered step lines when enabled', () => {
    const details = formatToolDetails(buildToolMessage({
      toolName: 'script_exec',
      toolStatus: 'success',
      rawOutput: JSON.stringify({
        summary: { step_count: 2 },
        steps: [
          { tool: 'read_file', args: { path: '/tmp/main.go' } },
          { tool: 'bash_exec', args: { command: 'go test ./...' } },
        ],
      }),
      traceId: 'trace-compact-1',
    }), { compactOutputEnabled: true });

    expect(details).toBe('step1: read /tmp/main.go\nstep2: run go test ./...');
    expect(details).not.toContain('summary:');
    expect(details).not.toContain('status:');
    expect(details).not.toContain('trace_id:');
  });

  it('renders compact failure details as error plus steps', () => {
    const details = formatToolDetails(buildToolMessage({
      toolName: 'script_exec',
      toolStatus: 'failed',
      content: JSON.stringify({
        script: [
          "search_files(query='needle', path='.')",
          "bash_exec(command='echo ok')",
        ].join('\n'),
      }),
      rawOutput: 'permission denied',
      traceId: 'trace-compact-2',
      toolCallId: 'tool-call-2',
    }), { compactOutputEnabled: true });

    expect(details).toBe('error: permission denied\nstep1: search needle\nstep2: run echo ok');
    expect(details).not.toContain('summary:');
    expect(details).not.toContain('status:');
    expect(details).not.toContain('trace_id:');
    expect(details).not.toContain('tool_call_id:');
  });

  it('keeps parsing legacy tools helper syntax in compact mode for historical logs', () => {
    const details = formatToolDetails(buildToolMessage({
      toolName: 'script_exec',
      toolStatus: 'success',
      content: JSON.stringify({
        script: [
          "tools.search_files(query='needle', path='.')",
          "tools.bash_exec(command='echo ok')",
        ].join('\n'),
      }),
    }), { compactOutputEnabled: true });

    expect(details).toBe('step1: search needle\nstep2: run echo ok');
  });

  it('falls back to step1 run in compact mode when no step can be inferred', () => {
    const details = formatToolDetails(buildToolMessage({
      toolName: '',
      content: 'script_exec finished',
    }), { compactOutputEnabled: true });

    expect(details).toBe('step1: run');
  });

  it('renders orchestration_dispatch private_send compact output as ordered private messages only', () => {
    const details = formatToolDetails(buildToolMessage({
      toolName: 'orchestration_dispatch',
      toolStatus: 'success',
      content: JSON.stringify({
        action: 'private_send',
        private_deliveries: [
          { participant_id: 'agent-2', content: 'secret-role' },
          { participant_id: 'agent-3', content: 'follow-up' },
        ],
      }),
      rawOutput: JSON.stringify({
        action: 'private_send',
        private_deliveries: [
          { participant_id: 'agent-2', content: 'secret-role' },
          { participant_id: 'agent-3', content: 'follow-up' },
        ],
      }),
    }), { compactOutputEnabled: true });

    expect(details).toBe('向agent-2发了私信：secret-role\n向agent-3发了私信：follow-up');
  });

  it('uses orchestration_dispatch private_messages args in compact mode before result hydration', () => {
    const details = formatToolDetails(buildToolMessage({
      toolName: 'orchestration_dispatch',
      toolStatus: 'running',
      content: JSON.stringify({
        action: 'private_send',
        private_messages: [
          { participant_id: 'agent-2', content: 'first role' },
          { participant_id: 'agent-4', content: 'second role' },
        ],
      }),
    }), { compactOutputEnabled: true });

    expect(details).toBe('向agent-2发了私信：first role\n向agent-4发了私信：second role');
  });

  it('detects legacy orchestration_dispatch envelopes even when toolName is missing', () => {
    const details = formatToolDetails(buildToolMessage({
      content: JSON.stringify({
        status: 'success',
        tool: 'orchestration_dispatch',
        trace_id: 'trace-owner-1',
        output: JSON.stringify({
          action: 'private_send',
          private_deliveries: [
            { participant_id: 'agent-2', content: 'legacy secret' },
            { participant_id: 'agent-5', content: 'legacy follow-up' },
          ],
        }),
        error: '',
      }),
    }), { compactOutputEnabled: true });

    expect(details).toBe('向agent-2发了私信：legacy secret\n向agent-5发了私信：legacy follow-up');
  });
});

function buildToolMessage(overrides: Partial<ToolChatMessage>): ToolChatMessage {
  return {
    id: 'tool-1',
    kind: 'tool',
    content: '',
    ...overrides,
  };
}

function readSummaryLine(details: string): string {
  return details.split('\n').find((line) => line.startsWith('summary: ')) || '';
}
