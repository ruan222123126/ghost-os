'use client';

import { type ReactNode, useEffect, useState } from 'react';
import { createPortal } from 'react-dom';

interface BodyPortalProps {
  children: ReactNode;
}

export function BodyPortal({ children }: BodyPortalProps) {
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  // Non-DOM renderers keep children inline; browsers mount overlays under body.
  if (typeof document === 'undefined') {
    return <>{children}</>;
  }

  if (!mounted) {
    return null;
  }

  return createPortal(children, document.body);
}
