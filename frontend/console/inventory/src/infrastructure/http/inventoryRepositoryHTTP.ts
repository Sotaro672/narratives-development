// frontend/console/inventory/src/infrastructure/http/inventoryRepositoryHTTP.ts

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

/**
 * Backend base URL
 * - .env の VITE_BACKEND_BASE_URL を優先
 * - 未設定時は Cloud Run の固定 URL を利用
 */
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

// ---------------------------------------------------------
// 共通: Firebase トークン取得
// ---------------------------------------------------------
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("Not authenticated");
  }
  const token = await user.getIdToken();
  if (!token) {
    throw new Error("Failed to acquire ID token");
  }
  return token;
}

// ---------------------------------------------------------
// Inventory 用：商品情報ヘッダー DTO
// ---------------------------------------------------------
export type InventoryProductSummary = {
  id: string;
  productName: string;
  brandId: string;
  brandName?: string;
  assigneeId: string;
  assigneeName?: string;
};

/**
 * 在庫詳細画面用：
 * ProductBlueprint ID から productName / brandId / assigneeId を取得
 *
 * GET /product-blueprints/{id}
 */
export async function fetchInventoryProductSummary(
  productBlueprintId: string,
): Promise<InventoryProductSummary> {
  const token = await getIdTokenOrThrow();

  const url = `${API_BASE}/product-blueprints/${encodeURIComponent(
    productBlueprintId,
  )}`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    throw new Error(
      `Failed to fetch product blueprint: ${res.status} ${res.statusText}`,
    );
  }

  const data = await res.json();

  const mapped: InventoryProductSummary = {
    id: String(data.id ?? ""),
    productName: String(data.productName ?? ""),
    brandId: String(data.brandId ?? ""),
    brandName: data.brandName ? String(data.brandName) : undefined,
    assigneeId: String(data.assigneeId ?? ""),
    assigneeName: data.assigneeName ? String(data.assigneeName) : undefined,
  };

  return mapped;
}

/**
 * 在庫一覧（ヘッダー用）:
 * printed == "printed" の ProductBlueprint 一覧を取得
 *
 * GET /product-blueprints/printed
 */
export async function fetchPrintedInventorySummaries(): Promise<
  InventoryProductSummary[]
> {
  const token = await getIdTokenOrThrow();

  const url = `${API_BASE}/product-blueprints/printed`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    throw new Error(
      `Failed to fetch printed product blueprints: ${res.status} ${res.statusText}`,
    );
  }

  const data = await res.json();

  if (!Array.isArray(data)) {
    return [];
  }

  const mapped: InventoryProductSummary[] = data.map((row: any) => ({
    id: String(row.id ?? ""),
    productName: String(row.productName ?? ""),
    brandId: String(row.brandId ?? ""),
    brandName: row.brandName ? String(row.brandName) : undefined,
    assigneeId: String(row.assigneeId ?? ""),
    assigneeName: row.assigneeName ? String(row.assigneeName) : undefined,
  }));

  return mapped;
}
