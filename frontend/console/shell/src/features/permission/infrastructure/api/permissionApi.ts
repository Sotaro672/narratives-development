// frontend/console/shell/src/features/permission/infrastructure/api/permissionApi.ts

import type { Permission } from "../../../../shared/types/permission";
import { buildConsoleUrl } from "../../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../../shared/http/authHeaders";

// ─────────────────────────────────────────────
// Backend API URL
// ─────────────────────────────────────────────

const PERMISSIONS_URL = buildConsoleUrl("/permissions");

// ─────────────────────────────────────────────
// HTTP
// ─────────────────────────────────────────────

async function requestJson<T>(
  input: string,
  init: RequestInit = {},
): Promise<T> {
  const authHeaders: Record<string, string> = await getAuthHeaders();
  const headers = new Headers(init.headers);

  for (const [key, value] of Object.entries(authHeaders)) {
    headers.set(key, value);
  }

  const response = await fetch(input, {
    ...init,
    headers,
  });

  const text = await response.text().catch(() => "");

  if (!response.ok) {
    throw new Error(
      `[permissionApi] ${response.status} ${response.statusText} :: ${text.slice(0, 300)}`,
    );
  }

  if (!text) {
    return [] as T;
  }

  const looksLikeHTML = /^\s*<!doctype html>|^\s*<html/i.test(text);

  if (looksLikeHTML) {
    throw new Error(
      "[permissionApi] response is not JSON (HTML received). " +
        `VITE_BACKEND_BASE_URL の設定を確認してください。received head: ${text.slice(0, 120)}`,
    );
  }

  try {
    return JSON.parse(text) as T;
  } catch {
    throw new Error(
      `[permissionApi] JSON parse error. head: ${text.slice(0, 120)}`,
    );
  }
}

// ─────────────────────────────────────────────
// Public API
// ─────────────────────────────────────────────

/**
 * 現在ログイン中のユーザー向けに、
 * 利用可能なPermission一覧を取得する。
 */
export async function fetchPermissions(): Promise<Permission[]> {
  const permissions = await requestJson<Permission[]>(
    PERMISSIONS_URL,
    {
      method: "GET",
    },
  );

  return Array.isArray(permissions) ? permissions : [];
}