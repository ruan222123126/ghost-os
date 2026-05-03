import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { MessageCopyButton } from './MessageCopyButton';

describe('components/message/MessageCopyButton', () => {
  it('renders icon-only UI for code-block copy buttons', () => {
    const html = renderCopyButton({ text: 'console.log(1)', variant: 'code' });

    expect(html).toContain('copy-button is-code-block');
    expect(html).not.toContain('copy-label');
  });

  it('keeps label for default copy buttons', () => {
    const html = renderCopyButton({ text: 'plain text' });

    expect(html).toContain('copy-label');
    expect(html).toContain('Copy');
  });
});

function renderCopyButton(props: React.ComponentProps<typeof MessageCopyButton>): string {
  return renderToStaticMarkup(
    React.createElement(
      WebLocaleProvider,
      {
        initialLocale: 'en-US',
        children: React.createElement(MessageCopyButton, props),
      },
    ),
  );
}
