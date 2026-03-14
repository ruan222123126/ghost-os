// Global error boundary UI for recoverable runtime errors in the web app.

'use client';

import { useEffect } from 'react';

interface ErrorPageProps {
  error: Error & { digest?: string };
  reset: () => void;
}

export default function ErrorPage({ error, reset }: ErrorPageProps) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <main className="app-shell">
      <section className="panel" style={{ maxWidth: '680px', margin: '12vh auto 0', padding: '24px' }}>
        <p className="kicker">Application Error</p>
        <h1 style={{ margin: 0, fontSize: '28px' }}>A runtime error interrupted the page.</h1>
        <p className="title-copy">You can safely retry without changing any backend state.</p>
        <div className="status-line error" style={{ margin: '18px 0 0' }}>{error.message}</div>
        <div className="panel-head-actions" style={{ marginTop: '18px' }}>
          <button type="button" onClick={reset} className="button-secondary">
            Try again
          </button>
        </div>
      </section>
    </main>
  );
}
