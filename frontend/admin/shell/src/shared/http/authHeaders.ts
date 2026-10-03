// frontend/admin/shell/src/shared/http/authHeaders.ts

import { isAdminAuthenticated } from "../../auth/application/adminAuth";

const ADMIN_DEV_AUTH_HEADER = "X-AMOL-Admin-Dev-Auth";
const ADMIN_DEV_AUTH_VALUE = "AMOL-Admin-2026#Start!";

export async function getAuthHeaders(): Promise<Record<string, string>> {
  if (!isAdminAuthenticated()) {
    throw new Error("ログインが必要です。");
  }

  return {
    [ADMIN_DEV_AUTH_HEADER]: ADMIN_DEV_AUTH_VALUE,
  };
}