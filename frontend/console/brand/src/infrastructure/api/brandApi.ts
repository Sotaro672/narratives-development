// frontend/console/brand/src/infrastructure/api/brandApi.ts
import { Brand, BrandPatch } from "../../../../brand/src/domain/entity/brand";
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

/**
 * Backend の BASE URL
 */
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)
    ?.replace(/\/+$/g, "") ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

const BASE_URL = ENV_BASE || FALLBACK_BASE;

/**
 * API Fetch Wrapper
 */
async function apiFetch(path: string, init: RequestInit = {}) {
  const url = `${BASE_URL}${path.startsWith("/") ? path : `/${path}`}`;

  const user = auth.currentUser;
  const token = user ? await user.getIdToken() : null;

  const headers: HeadersInit = {
    ...(init.method === "POST" || init.method === "PATCH"
      ? { "Content-Type": "application/json" }
      : {}),
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(init.headers ?? {}),
  };

  const res = await fetch(url, {
    ...init,
    headers,
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`API Error ${res.status}: ${text}`);
  }

  return res.json().catch(() => null);
}

/* ------------------------------------------------------------
 * Brand API  (REST: /brands)
 * ------------------------------------------------------------ */

/**
 * GET /brands — 全ブランド一覧取得
 */
export async function fetchAllBrands(): Promise<Brand[]> {
  return apiFetch(`/brands`, { method: "GET" });
}

/**
 * GET /brands/:id — ブランド取得
 */
export async function fetchBrandById(id: string): Promise<Brand> {
  return apiFetch(`/brands/${encodeURIComponent(id)}`, {
    method: "GET",
  });
}

/**
 * POST /brands — ブランド新規作成
 * Payload: BrandPatch（id や timestamp は backend 側で採番）
 */
export async function createBrand(data: BrandPatch): Promise<Brand> {
  return apiFetch(`/brands`, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

/**
 * PATCH /brands/:id — 部分更新
 */
export async function updateBrand(
  id: string,
  patch: BrandPatch
): Promise<Brand> {
  return apiFetch(`/brands/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify(patch),
  });
}

/**
 * DELETE /brands/:id — 論理削除
 */
export async function deleteBrand(id: string): Promise<void> {
  await apiFetch(`/brands/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

/**
 * POST /brands/:id/activate — 有効化
 */
export async function activateBrandApi(id: string): Promise<Brand> {
  return apiFetch(`/brands/${encodeURIComponent(id)}/activate`, {
    method: "POST",
  });
}

/**
 * POST /brands/:id/deactivate — 無効化
 */
export async function deactivateBrandApi(id: string): Promise<Brand> {
  return apiFetch(`/brands/${encodeURIComponent(id)}/deactivate`, {
    method: "POST",
  });
}

/* ------------------------------------------------------------
 * Patch Utility
 * ------------------------------------------------------------ */

/**
 * BrandPatch を生成する小ユーティリティ
 */
export function toBrandPatch(p: Partial<Brand>): BrandPatch {
  return { ...p };
}
