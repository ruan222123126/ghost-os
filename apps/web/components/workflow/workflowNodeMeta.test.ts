import { copyForWorkflow } from '@/lib/i18n/messages/workflow';
import { metadataForNodeType, nodeLibraryForCopy } from './workflowNodeMeta';

describe('components/workflow/workflowNodeMeta', () => {
  it('builds the default node library from workflow copy labels', () => {
    const library = nodeLibraryForCopy(copyForWorkflow('en-US'));

    expect(library.map((item) => item.id)).toEqual(['agent', 'llm', 'tool', 'if', 'loop']);
    expect(library.map((item) => item.label)).toEqual([
      'AGENT',
      'LLM MODEL',
      'TOOLS USE',
      'IF BRANCH',
      'LOOP',
    ]);
  });

  it('resolves metadata for non-library boundary nodes', () => {
    expect(metadataForNodeType('start', copyForWorkflow('en-US'))).toEqual({
      id: 'start',
      label: 'START',
      glyph: '⚡',
    });
  });
});
