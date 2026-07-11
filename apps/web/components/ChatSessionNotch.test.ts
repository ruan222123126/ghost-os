import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { ChatSessionNotch } from './ChatSessionNotch';

describe('components/ChatSessionNotch', () => {
  it('renders the session title and full id when a session id exists', () => {
    const html = renderToStaticMarkup(
      React.createElement(ChatSessionNotch, {
        sessionId: 'session-12345678-abcdef',
        title: 'Planning',
      }),
    );

    expect(html).toContain('Planning');
    expect(html).toContain('session-12345678-abcdef');
  });

  it('does not render without a session id', () => {
    const html = renderToStaticMarkup(
      React.createElement(ChatSessionNotch, {
        sessionId: '   ',
        title: 'Draft',
      }),
    );

    expect(html).toBe('');
  });
});
