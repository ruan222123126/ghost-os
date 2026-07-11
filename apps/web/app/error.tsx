// Global error boundary UI for recoverable runtime errors in the web app.

'use client';

import { useEffect } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';

interface ErrorPageProps {
  error: Error & { digest?: string };
  reset: () => void;
}

export default function ErrorPage({ error, reset }: ErrorPageProps) {
  const { copy } = useWebLocale();

  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <main className="app-shell">
      <section className="panel" style={{ maxWidth: '680px', margin: '12vh auto 0', padding: '24px' }}>
        <p className="kicker">{copy.system.appErrorKicker}</p>
        <h1 style={{ margin: 0, fontSize: '28px' }}>{copy.system.appErrorTitle}</h1>
        <p className="title-copy">{copy.system.appErrorCopy}</p>
        <div className="status-line error" style={{ margin: '18px 0 0' }}>{error.message}</div>
        <div className="panel-head-actions" style={{ marginTop: '18px' }}>
          <button type="button" onClick={reset} className="button-secondary">
            {copy.system.appErrorRetry}
          </button>
        </div>
      </section>
    </main>
  );
}
