'use client';

import { useEffect, useState } from 'react';

interface UseInitialLoadingOverlayOptions {
  ready: boolean;
}

export function useInitialLoadingOverlay(options: UseInitialLoadingOverlayOptions): boolean {
  const [visible, setVisible] = useState(true);

  useEffect(() => {
    if (!visible || !options.ready) {
      return;
    }

    setVisible(false);
  }, [options.ready, visible]);

  return visible;
}
