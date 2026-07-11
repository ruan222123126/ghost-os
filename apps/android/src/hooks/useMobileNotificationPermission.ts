import { useEffect, useState } from "react";
import {
  MOBILE_NOTIFICATION_PERMISSION_PROMPTED_KEY,
  currentMobileNotificationPermission,
  requestMobileNotificationPermission,
  retryPendingMobileNotifications,
  type MobileNotificationPermissionState,
} from "../lib/mobileNotifications";

interface UseMobileNotificationPermissionOptions {
  connected: boolean;
  onDenied: () => void;
}

export function useMobileNotificationPermission(
  options: UseMobileNotificationPermissionOptions,
): MobileNotificationPermissionState {
  const [permission, setPermission] = useState<MobileNotificationPermissionState>("unknown");

  useEffect(() => {
    let cancelled = false;
    void currentMobileNotificationPermission()
      .then((state) => {
        if (!cancelled) {
          setPermission(state);
        }
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          console.error("[useMobileNotificationPermission] permission check failed", error);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!options.connected || permission === "unsupported" || permission === "granted") {
      return;
    }
    if (window.localStorage.getItem(MOBILE_NOTIFICATION_PERMISSION_PROMPTED_KEY) === "true") {
      if (permission === "denied") {
        options.onDenied();
      }
      return;
    }

    let cancelled = false;
    window.localStorage.setItem(MOBILE_NOTIFICATION_PERMISSION_PROMPTED_KEY, "true");
    void requestMobileNotificationPermission()
      .then((state) => {
        if (cancelled) {
          return;
        }
        setPermission(state);
        if (state === "denied") {
          options.onDenied();
          return;
        }
        if (state === "granted") {
          void retryPendingMobileNotifications().catch((error: unknown) => {
            console.error("[useMobileNotificationPermission] notification retry failed", error);
          });
        }
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          console.error("[useMobileNotificationPermission] permission request failed", error);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [options.connected, options.onDenied, permission]);

  return permission;
}
