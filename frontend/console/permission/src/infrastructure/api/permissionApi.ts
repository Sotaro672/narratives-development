// frontend/permission/src/infrastructure/api/permissionApi.ts

import type { Permission } from "../../domain/entity/permission";

// Firebase Auth（member の useMemberList.ts と同じパターン）
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ─────────────────────────────────────────────
// Backend base URL
// ─────────────────────────────────────────────

const RAW_ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

function sanitizeBase(u: string): string {
  return (u || "").replace(/\/+$/g, "");
}

const ENV_BASE = sanitizeBase(RAW_ENV_BASE);
const FINAL_BASE = sanitizeBase(ENV_BASE || FALLBACK_BASE);

if (!FINAL_BASE) {
  throw new Error(
    "[permissionApi] BACKEND BASE URL is empty. Set VITE_BACKEND_BASE_URL in .env.local",
  );
}

// e.g. https://.../permissions/
const PERMISSIONS_URL = `${FINAL_BASE}/permissions/`;

// ─────────────────────────────────────────────
// 共通: 認証付き fetch
// ─────────────────────────────────────────────

async function authFetch<T>(input: string, init: RequestInit = {}): Promise<T> {
  const currentUser = auth.currentUser;
  const token = await currentUser?.getIdToken();

  if (!token) {
    throw new Error("[permissionApi] Not authenticated (no ID token).");
  }

  const res = await fetch(input, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      ...(init.headers ?? {}),
    },
  });

  const text = await res.text().catch(() => "");

  if (!res.ok) {
    // 401/403 はそのままエラーにしておく
    throw new Error(
      `[permissionApi] ${res.status} ${res.statusText} :: ${text.slice(
        0,
        300,
      )}`,
    );
  }

  if (!text) {
    return [] as T; // Permission[] を想定
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
// 公開 API
// ─────────────────────────────────────────────

/**
 * 現在ログイン中ユーザー向けに、利用可能な Permission 一覧を取得する。
 *
 * 想定レスポンス: Permission[]
 *   [
 *     { id: "perm_wallet_view", name: "wallet.view", ... },
 *     ...
 *   ]
 *
 * バックエンド側で:
 *   - 全 Permission を返す
 *   - もしくは ロール/メンバーに応じてフィルタした Permission を返す
 * などの振る舞いを持たせることができます。
 */
export async function fetchPermissions(): Promise<Permission[]> {
  const perms = await authFetch<Permission[]>(PERMISSIONS_URL, {
    method: "GET",
  });

  // 念のため null/undefined を防ぐ
  return Array.isArray(perms) ? perms : [];
}
