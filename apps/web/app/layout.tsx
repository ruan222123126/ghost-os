// Root layout for metadata, global styles, and top-level providers.

import type { Metadata } from 'next';
import { headers } from 'next/headers';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { resolveInitialLocale } from '@/lib/i18n/locale';
import './globals.css';
import './styles/primitives.css';
import './styles/shell.css';
import './styles/messages.css';
import './styles/assistant-markdown.css';
import './styles/message-images.css';
import './styles/composer.css';
import './styles/composer-model-selector.css';
import './styles/composer-images.css';
import './styles/settings.css';
import './styles/settings-panels.css';
import './styles/workflow-layout.css';
import './styles/workflow-sidebar.css';
import './styles/workflow-node.css';
import './styles/workflow-properties.css';
import './styles/workflow-properties-fields.css';
import './styles/workflow-variable-autocomplete.css';
import './styles/workflow-settings.css';
import './styles/workflow-screen-composer.css';
import './styles/responsive.css';

export const metadata: Metadata = {
  title: 'Ghost-OS Console',
  description: 'Minimal web console for Ghost-OS bridge agent',
};

function resolveRequestLocale(): ReturnType<typeof resolveInitialLocale> {
  const acceptLanguage = headers().get('accept-language') ?? undefined;
  return resolveInitialLocale({ browserLocale: acceptLanguage });
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const initialLocale = resolveRequestLocale();
  return (
    <html lang={initialLocale}>
      <body className="bg-app-bg text-app-text antialiased">
        <WebLocaleProvider initialLocale={initialLocale}>{children}</WebLocaleProvider>
      </body>
    </html>
  );
}
