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

  it('keeps find_icon references in typed rows', () => {
    expect(rowsToToolArguments([
      { key: 'limit', valueType: 'number', value: '${find_icon.x}' },
      { key: 'payload', valueType: 'object', value: '${find_icon}' },
      { key: 'items', valueType: 'array', value: '${find_icon.matches}' },
    ])).toEqual({
      limit: '${find_icon.x}',
      payload: '${find_icon}',
      items: '${find_icon.matches}',
    });
  });

  it('rejects deprecated non-find_icon references in typed rows', () => {
    expect(() => rowsToToolArguments([
      { key: 'limit', valueType: 'number', value: '${inputs.limit}' },
    ])).toThrow('number value is invalid');
  });
});
