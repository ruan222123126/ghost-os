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
    <main className="mx-auto flex min-h-screen w-full max-w-3xl items-center justify-center px-4 py-8">
      <section className="ui-panel animate-riseSoft w-full border-rose-300/30 bg-rose-300/10 p-6">
        <h1 className="text-xl font-semibold text-rose-100">Application Error</h1>
        <p className="mt-2 text-sm text-rose-200/90">A runtime error interrupted the page. You can retry safely.</p>
        <button
          type="button"
          onClick={reset}
          className="ui-btn-secondary mt-4 border-rose-200/30 bg-rose-200/10 px-4 py-2 text-sm text-rose-100 hover:border-rose-200/45 hover:bg-rose-200/20"
        >
          Try again
        </button>
      </section>
    </main>
  );
}
