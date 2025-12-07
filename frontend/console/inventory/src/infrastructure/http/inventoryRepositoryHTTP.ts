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
  assigneeId: string;
};

/**
 * 在庫詳細画面用：
 * ProductBlueprint ID（= inventoryId として利用想定）から
 * productName / brandId / assigneeId を取得する。
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

  // 🔍 どこに取りに行っているか
  console.log("[InventoryAPI] fetchInventoryProductSummary request:", {
    url,
    productBlueprintId,
  });

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  console.log("[InventoryAPI] fetchInventoryProductSummary response status:", {
    status: res.status,
    statusText: res.statusText,
  });

  if (!res.ok) {
    throw new Error(
      `Failed to fetch product blueprint: ${res.status} ${res.statusText}`,
    );
  }

  const data = await res.json();

  // 🔍 backend からそのまま返ってきた JSON
  console.log("[InventoryAPI] fetchInventoryProductSummary raw data:", data);

  const mapped: InventoryProductSummary = {
    id: String(data.id ?? ""),
    productName: String(data.productName ?? ""),
    brandId: String(data.brandId ?? ""),
    assigneeId: String(data.assigneeId ?? ""),
  };

  // 🔍 画面に渡す直前の整形済みオブジェクト
  console.log(
    "[InventoryAPI] fetchInventoryProductSummary mapped summary:",
    mapped,
  );

  return mapped;
}

/**
 * 在庫一覧（ヘッダー用）:
 * printed == "printed" の ProductBlueprint 一覧を取得し、
 * productName / brandId / assigneeId をまとめて取る。
 *
 * GET /product-blueprints/printed
 */
export async function fetchPrintedInventorySummaries(): Promise<
  InventoryProductSummary[]
> {
  const token = await getIdTokenOrThrow();

  const url = `${API_BASE}/product-blueprints/printed`;

  console.log("[InventoryAPI] fetchPrintedInventorySummaries request:", { url });

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  console.log(
    "[InventoryAPI] fetchPrintedInventorySummaries response status:",
    {
      status: res.status,
      statusText: res.statusText,
    },
  );

  if (!res.ok) {
    throw new Error(
      `Failed to fetch printed product blueprints: ${res.status} ${res.statusText}`,
    );
  }

  const data = await res.json();

  // 🔍 生の配列（handler の ProductBlueprintListOutput）
  console.log("[InventoryAPI] fetchPrintedInventorySummaries raw data:", data);

  if (!Array.isArray(data)) {
    console.warn(
      "[InventoryAPI] fetchPrintedInventorySummaries: response is not an array",
    );
    return [];
  }

  const mapped: InventoryProductSummary[] = data.map((row: any) => ({
    id: String(row.id ?? ""),
    productName: String(row.productName ?? ""),
    brandId: String(row.brandId ?? ""),
    assigneeId: String(row.assigneeId ?? ""),
  }));

  // 🔍 画面用にマッピング後の一覧
  console.log(
    "[InventoryAPI] fetchPrintedInventorySummaries mapped summaries:",
    mapped,
  );

  return mapped;
}
