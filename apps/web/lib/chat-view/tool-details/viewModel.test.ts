import type { ToolChatMessage } from '@/lib/types';
import { buildToolCardViewModel, getStatusLabel, getToolTone } from './viewModel';

describe('lib/chat-view/tool-details/viewModel', () => {
  it('builds promoted command title, status tone, and compact details', () => {
    const tool: ToolChatMessage = {
      id: 'tool-1',
      kind: 'tool',
      content: 'ok\nextra',
      rawOutput: 'ok\nextra',
      toolInput: JSON.stringify({ command: 'echo ok' }),
      toolName: 'bash_exec',
      toolStatus: 'success',
      traceId: 'trace-1',
    };

    expect(buildToolCardViewModel(tool, { compactOutputEnabled: true })).toEqual({
      title: 'echo ok',
      tone: 'success',
      statusLabel: 'SUCCESS',
      details: 'success\nok\nextra',
      titleMode: 'status',
      showTerminalIcon: true,
    });
  });

  it('uses localized fallback title and preparing details for unnamed pending tools', () => {
    expect(buildToolCardViewModel({
      id: 'tool-2',
      kind: 'tool',
      content: '',
    }, {
      fallbackTitle: '工具',
      preparingDetails: '准备输出',
    })).toEqual({
      title: '工具',
      tone: 'running',
      statusLabel: 'RUNNING',
      details: '准备输出',
      titleMode: 'status',
      showTerminalIcon: true,
    });
  });

  it('uses plain Chinese titles without terminal icons for file actions', () => {
    expect(buildToolCardViewModel({
      id: 'tool-read',
      kind: 'tool',
      content: 'package main',
      toolInput: JSON.stringify({ path: '/tmp/main.go' }),
      toolName: 'read_file',
      toolStatus: 'success',
    })).toMatchObject({
      title: '阅读文件 /tmp/main.go',
      titleMode: 'plain',
      showTerminalIcon: false,
    });

    expect(buildToolCardViewModel({
      id: 'tool-write',
      kind: 'tool',
      content: JSON.stringify({ path: '/tmp/out.txt', content_summary: { lines: 4 } }),
      toolName: 'write_file',
      toolStatus: 'success',
    })).toMatchObject({
      title: '创建文件 /tmp/out.txt +4',
      titleMode: 'plain',
      showTerminalIcon: false,
    });

    expect(buildToolCardViewModel({
      id: 'tool-edit',
      kind: 'tool',
      toolInput: JSON.stringify({
        path: '/tmp/out.txt',
        diff_text: '@@ -1,2 +1,3 @@\n-old\n+new\n+extra\n same\n',
      }),
      content: 'applied 1 hunks to /tmp/out.txt',
      toolName: 'apply_diff',
      toolStatus: 'success',
    })).toMatchObject({
      title: '编辑文件 /tmp/out.txt +2 -1',
      titleMode: 'plain',
      showTerminalIcon: false,
    });
  });

  it('uses plain Chinese titles for web, codex, skill, and search actions', () => {
    expect(buildToolCardViewModel({
      id: 'tool-web',
      kind: 'tool',
      content: JSON.stringify([{ title: 'Ghost OS' }]),
      toolInput: JSON.stringify({ query: 'Ghost OS release notes' }),
      toolName: 'web_search',
      toolStatus: 'success',
    })).toMatchObject({
      title: '网络搜索 Ghost OS release notes',
      titleMode: 'plain',
      showTerminalIcon: false,
    });

    expect(buildToolCardViewModel({
      id: 'tool-codex',
      kind: 'tool',
      content: JSON.stringify({ status: 'running' }),
      toolInput: JSON.stringify({ op: 'status', session_id: 'cmd-123' }),
      toolName: 'codex_cli',
      toolStatus: 'running',
    })).toMatchObject({
      title: '调用codex status',
      titleMode: 'plain',
      showTerminalIcon: false,
    });

    expect(buildToolCardViewModel({
      id: 'tool-sfind',
      kind: 'tool',
      content: '',
      rawOutput: JSON.stringify({ action: 'load', items: [{ name: 'release_flow' }] }),
      toolName: 'sfind',
      toolStatus: 'success',
    })).toMatchObject({
      title: '加载技能 release_flow',
      titleMode: 'plain',
      showTerminalIcon: false,
    });

    expect(buildToolCardViewModel({
      id: 'tool-search',
      kind: 'tool',
      content: JSON.stringify({ query: 'bridge runtime selector' }),
      toolName: 'search_files',
      toolStatus: 'success',
    })).toMatchObject({
      title: '搜索 bridge runtime selector（关键词）',
      titleMode: 'plain',
      showTerminalIcon: false,
    });
  });

  it('uses a plain Chinese title without terminal icon for screen screenshots', () => {
    expect(buildToolCardViewModel({
      id: 'tool-screen',
      kind: 'tool',
      content: '',
      toolInput: JSON.stringify({ action: 'screenshot', display_id: 67 }),
      toolName: 'screen_control',
      toolStatus: 'running',
    }, { preparingDetails: '准备输出' })).toMatchObject({
      title: '截取屏幕',
      details: '准备输出',
      titleMode: 'plain',
      showTerminalIcon: false,
    });
  });

  it('normalizes status labels by tone', () => {
    expect(getToolTone('failed')).toBe('error');
    expect(getToolTone('pending')).toBe('running');
    expect(getStatusLabel('success', 'completed')).toBe('SUCCESS');
    expect(getStatusLabel('success', 'rate_limited')).toBe('RATE LIMITED');
  });
});
