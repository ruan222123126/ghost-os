import { getCurrentWindow } from "@tauri-apps/api/window";
import { useEffect, useRef, useState } from "react";
import { hasTauriRuntime } from "../lib/bridgeBus";

export function useMobileAppVisibility(): boolean {
  const documentVisibleRef = useRef(isDocumentVisible());
  const windowFocusedRef = useRef(true);
  const [visible, setVisible] = useState(documentVisibleRef.current);

  useEffect(() => {
    let cancelled = false;
    let stopFocusListener: (() => void) | undefined;
    const commitVisibility = (): void => {
      if (!cancelled) {
        setVisible(documentVisibleRef.current && windowFocusedRef.current);
      }
    };
    const handleDocumentVisibility = (): void => {
      documentVisibleRef.current = isDocumentVisible();
      commitVisibility();
    };

    document.addEventListener("visibilitychange", handleDocumentVisibility);
    if (hasTauriRuntime()) {
      const appWindow = getCurrentWindow();
      void Promise.all([
        appWindow.isFocused().then((focused) => {
          windowFocusedRef.current = focused;
          commitVisibility();
        }),
        appWindow.onFocusChanged(({ payload }) => {
          windowFocusedRef.current = payload;
          commitVisibility();
        }).then((stop) => {
          if (cancelled) {
            stop();
            return;
          }
          stopFocusListener = stop;
        }),
      ]).catch((error: unknown) => {
        console.error("[useMobileAppVisibility] window visibility listener failed", error);
      });
    }

    return () => {
      cancelled = true;
      document.removeEventListener("visibilitychange", handleDocumentVisibility);
      stopFocusListener?.();
    };
  }, []);

  return visible;
}

function isDocumentVisible(): boolean {
  return typeof document === "undefined" || document.visibilityState === "visible";
}
