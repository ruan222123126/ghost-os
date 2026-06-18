import { invoke } from "@tauri-apps/api/core";

export async function saveMobileCredential(deviceId: string, secret: string): Promise<void> {
  await invoke("mobile_credential_save", { deviceId, secret });
}

export async function loadMobileCredential(deviceId: string): Promise<string> {
  return invoke<string>("mobile_credential_load", { deviceId });
}

export async function deleteMobileCredential(deviceId: string): Promise<void> {
  await invoke("mobile_credential_delete", { deviceId });
}
