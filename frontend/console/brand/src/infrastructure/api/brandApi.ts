// frontend/console/brand/src/infrastructure/api/brandApi.ts
import { Brand, BrandPatch } from "../../../../brand/src/domain/entity/brand";
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

/**
 * Backend の BASE URL 推論
 * - VITE_BACKEND_BASE_URL があれば使う
 * - ない場合は Cloud Run デフォルトを fallback
 */
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    ""
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-xxxxx-an.a.run.app"; // 必要に応じて修正

const BASE_URL = ENV_BASE || FALLBACK_BASE;

/**
 * Backend API 共通 fetch wrapper
 * - Firebase Auth の ID トークンを付与
 * - JSON パース
 */
async function apiFetch(input: string, init: RequestInit = {}) {
  const user = auth.currentUser;
  const token = user ? await user.getIdToken() : undefined;

  const headers: HeadersInit = {
    "Content-Type": "application/json",
    ...(token ? { "X-Auth-Token": token } : {}),
    ...(init.headers ?? {}),
  };

  const res = await fetch(`${BASE_URL}${input}`, {
    ...init,
    headers,
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`API error ${res.status}: ${text}`);
  }

  return res.json().catch(() => null);
}

/* ------------------------------------------------------------
 * Brand API
 * Backend: /brand, /brand/:id など REST を想定
 * ------------------------------------------------------------ */

/**
 * すべてのBrandを取得
 */
export async function fetchAllBrands(): Promise<Brand[]> {
  return apiFetch(`/brand`, {
    method: "GET",
  });
}

/**
 * ID指定で Brand を取得
 */
export async function fetchBrandById(id: string): Promise<Brand> {
  return apiFetch(`/brand/${encodeURIComponent(id)}`, {
    method: "GET",
  });
}

/**
 * Brand を新規登録
 * backend: POST /brand
 */
export async function createBrand(data: Brand): Promise<Brand> {
  return apiFetch(`/brand`, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

/**
 * Brand を部分更新（PATCH）
 */
export async function updateBrand(
  id: string,
  patch: BrandPatch
): Promise<Brand> {
  return apiFetch(`/brand/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify(patch),
  });
}

/**
 * Brand を論理削除（DELETE）
 */
export async function deleteBrand(id: string): Promise<void> {
  await apiFetch(`/brand/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

/**
 * Brand を有効化
 * backend が /brand/:id/activate を持つ前提
 */
export async function activateBrandApi(id: string): Promise<Brand> {
  return apiFetch(`/brand/${encodeURIComponent(id)}/activate`, {
    method: "POST",
  });
}

/**
 * Brand を停止（無効化）
 */
export async function deactivateBrandApi(id: string): Promise<Brand> {
  return apiFetch(`/brand/${encodeURIComponent(id)}/deactivate`, {
    method: "POST",
  });
}

/* ------------------------------------------------------------
 * 補助ユーティリティ（不要なら削除可）
 * ------------------------------------------------------------ */

/**
 * BrandPatch を簡単に生成するヘルパ（null クリア対応）
 */
export function toBrandPatch(b: Partial<Brand>): BrandPatch {
  return { ...b };
}
