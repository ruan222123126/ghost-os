// Root layout for metadata, global styles, and top-level providers.

import type { Metadata } from 'next';
import './globals.css';
import './styles/primitives.css';
import './styles/shell.css';
import './styles/messages.css';
import './styles/composer.css';
import './styles/settings.css';
import './styles/responsive.css';

export const metadata: Metadata = {
  title: 'Ghost-OS Console',
  description: 'Minimal web console for Ghost-OS bridge agent',
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="bg-app-bg text-app-text antialiased">
        {children}
      </body>
    </html>
  );
}
