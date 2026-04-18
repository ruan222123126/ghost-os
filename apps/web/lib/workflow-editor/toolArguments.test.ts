import {
  jsonTextToToolArguments,
  rowsToToolArguments,
  toolArgumentsToJSONText,
  toolArgumentsToRows,
} from '@/lib/workflow-editor/toolArguments';

describe('lib/workflow-editor/toolArguments', () => {
  it('converts tool arguments between JSON and row modes', () => {
    const args = {
      query: 'ghost-os',
      limit: 5,
      strict: true,
      filters: { source: 'docs' },
      tags: ['bridge'],
    };

    const jsonText = toolArgumentsToJSONText(args);
    const parsedFromJSON = jsonTextToToolArguments(jsonText);
    const rows = toolArgumentsToRows(parsedFromJSON);
    const parsedFromRows = rowsToToolArguments(rows);

    expect(parsedFromRows).toEqual(args);
  });

  it('throws when JSON mode is not object', () => {
    expect(() => jsonTextToToolArguments('[]')).toThrow('tool arguments JSON must be an object');
  });

  it('keeps template references in typed rows', () => {
    expect(rowsToToolArguments([
      { key: 'limit', valueType: 'number', value: '${inputs.limit}' },
      { key: 'strict', valueType: 'boolean', value: '${outputs.if-node.ok}' },
      { key: 'payload', valueType: 'object', value: '${outputs.tool-node.data}' },
      { key: 'items', valueType: 'array', value: '${inputs.items}' },
    ])).toEqual({
      limit: '${inputs.limit}',
      strict: '${outputs.if-node.ok}',
      payload: '${outputs.tool-node.data}',
      items: '${inputs.items}',
    });
  });
});
