import { formatAssistantTextForDisplay } from '../../shared/assistantTextSpacing';

describe('formatAssistantTextForDisplay', () => {
  it('adds light paragraph spacing between adjacent sentence outputs', () => {
    expect(formatAssistantTextForDisplay('先检查项目。测试通过。继续收尾。')).toBe(
      '先检查项目。\n\n测试通过。\n\n继续收尾。',
    );
  });

  it('keeps existing multiline content unchanged', () => {
    const content = '第一行。\n第二行。第三行。';

    expect(formatAssistantTextForDisplay(content)).toBe(content);
  });

  it('does not split markdown image syntax after exclamation marks', () => {
    const content = '![diagram](./diagram.png)说明完成。继续。';

    expect(formatAssistantTextForDisplay(content)).toBe(
      '![diagram](./diagram.png)说明完成。\n\n继续。',
    );
  });
});
