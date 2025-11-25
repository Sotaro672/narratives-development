// frontend/console/shell/src/auth/application/companyService.ts
/// <reference types="vite/client" />

import type { CompanyDTO } from "../domain/entity/company";
// 🔽 Firebase Auth クライアントを利用して ID トークンを取得
import { auth } from "../infrastructure/config/firebaseClient";

// -------------------------------
// Backend base URL（useMemberDetail と同じ構成）
// -------------------------------
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

// 最終的に使うベース URL
const API_BASE = ENV_BASE || FALLBACK_BASE;

// -------------------------------
// In-memory cache for company names
// -------------------------------
const nameCache = new Map<string, Promise<string | null>>();

// -------------------------------
// Company: fetchers
// -------------------------------
export async function getCompanyById(companyId: string): Promise<CompanyDTO | null> {
  const id = (companyId ?? "").trim();
  if (!id) return null;

  // ★ Firebase ログインユーザーから ID トークン取得
  const user = auth.currentUser;
  if (!user) {
    console.warn("[companyService] getCompanyById called without logged-in user");
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }

  const idToken = await user.getIdToken();

  const res = await fetch(`${API_BASE}/companies/${encodeURIComponent(id)}`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  if (!res.ok) {
    // 404 などはそのまま null で返してもいいし、エラーにしてもOK
    console.error("[companyService] GET /companies failed", {
      status: res.status,
      statusText: res.statusText,
    });
    return null;
  }

  const data = (await res.json()) as CompanyDTO;
  return data ?? null;
}

export async function getCompanyNameById(companyId: string): Promise<string | null> {
  const data = await getCompanyById(companyId);
  const name = (data?.name ?? "").trim();
  return name || null;
}

// Cached version
export function getCompanyNameByIdCached(companyId: string): Promise<string | null> {
  const id = (companyId ?? "").trim();
  if (!id) return Promise.resolve(null);

  const cached = nameCache.get(id);
  if (cached) return cached;

  const p = getCompanyNameById(id).catch((err) => {
    console.error("[companyService] getCompanyNameByIdCached error", err);
    nameCache.delete(id);
    return null;
  });
  nameCache.set(id, p);
  return p;
}

export function clearCompanyNameCache(companyId?: string) {
  if (!companyId) {
    nameCache.clear();
  } else {
    nameCache.delete((companyId ?? "").trim());
  }
}
