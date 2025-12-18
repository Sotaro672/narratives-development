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

function s(v: unknown): string {
  return String(v ?? "").trim();
}

function n(v: unknown): number {
  const x = Number(v ?? 0);
  return Number.isFinite(x) ? x : 0;
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

// ---------------------------------------------------------
// ✅ Inventory 一覧DTO（管理一覧）
// GET /inventory
// ---------------------------------------------------------
export type InventoryListRowDTO = {
  productBlueprintId: string;
  productName: string;

  tokenBlueprintId: string; // ✅ 必須（detail遷移のキー）
  tokenName: string;

  modelNumber: string;
  stock: number;
};

function normalizeInventoryListRow(raw: any): InventoryListRowDTO | null {
  const productBlueprintId = s(raw?.productBlueprintId ?? raw?.productBlueprintID);
  const productName = s(raw?.productName);

  const tokenBlueprintId = s(raw?.tokenBlueprintId ?? raw?.tokenBlueprintID);
  const tokenName = s(raw?.tokenName);

  const modelNumber = s(raw?.modelNumber ?? raw?.modelNum);
  const stock = n(raw?.stock);

  // ✅ 方針A: pbId/tbId は必須。ここで落とす（"-" 埋めはしない）
  if (!productBlueprintId || !tokenBlueprintId) return null;

  return {
    productBlueprintId,
    productName,
    tokenBlueprintId,
    tokenName,
    modelNumber,
    stock,
  };
}

/**
 * ✅ Inventory 一覧DTO
 * - 戻り値は "必ず tokenBlueprintId を含む" 正規化済み配列
 */
export async function fetchInventoryListDTO(): Promise<InventoryListRowDTO[]> {
  const token = await getIdTokenOrThrow();
  const url = `${API_BASE}/inventory`;

  console.log("[inventory/fetchInventoryListDTO] start", { url });

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    console.error("[inventory/fetchInventoryListDTO] failed", {
      url,
      status: res.status,
      statusText: res.statusText,
      body: text,
    });
    throw new Error(`Failed to fetch inventory list: ${res.status} ${res.statusText}`);
  }

  const data = (await res.json()) as any;

  // ✅ 互換吸収を減らす：基本は配列を期待。どうしても違う場合のみ items を許容。
  const rawItems: any[] = Array.isArray(data)
    ? data
    : Array.isArray(data?.items)
      ? data.items
      : [];

  const mapped = rawItems
    .map(normalizeInventoryListRow)
    .filter((x): x is InventoryListRowDTO => x !== null);

  console.log("[inventory/fetchInventoryListDTO] ok", {
    rawCount: rawItems.length,
    mappedCount: mapped.length,
    sample: mapped.slice(0, 3),
  });

  return mapped;
}

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

  const pbId = s(productBlueprintId);
  if (!pbId) throw new Error("productBlueprintId is empty");

  const url = `${API_BASE}/product-blueprints/${encodeURIComponent(pbId)}`;

  console.log("[inventory/fetchInventoryProductSummary] start", { productBlueprintId: pbId, url });

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    console.error("[inventory/fetchInventoryProductSummary] failed", {
      productBlueprintId: pbId,
      url,
      status: res.status,
      statusText: res.statusText,
      body: text,
    });
    throw new Error(`Failed to fetch product blueprint: ${res.status} ${res.statusText}`);
  }

  const data = await res.json();

  const mapped: InventoryProductSummary = {
    id: s(data?.id),
    productName: s(data?.productName),
    brandId: s(data?.brandId),
    brandName: data?.brandName ? s(data.brandName) : undefined,
    assigneeId: s(data?.assigneeId),
    assigneeName: data?.assigneeName ? s(data.assigneeName) : undefined,
  };

  console.log("[inventory/fetchInventoryProductSummary] ok", { productBlueprintId: pbId, mapped });

  return mapped;
}

/**
 * 在庫一覧（ヘッダー用）:
 * printed == "printed" の ProductBlueprint 一覧を取得
 *
 * GET /product-blueprints/printed
 */
export async function fetchPrintedInventorySummaries(): Promise<InventoryProductSummary[]> {
  const token = await getIdTokenOrThrow();
  const url = `${API_BASE}/product-blueprints/printed`;

  console.log("[inventory/fetchPrintedInventorySummaries] start", { url });

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    console.error("[inventory/fetchPrintedInventorySummaries] failed", {
      url,
      status: res.status,
      statusText: res.statusText,
      body: text,
    });
    throw new Error(`Failed to fetch printed product blueprints: ${res.status} ${res.statusText}`);
  }

  const data = await res.json();
  if (!Array.isArray(data)) return [];

  const mapped: InventoryProductSummary[] = data.map((row: any) => ({
    id: s(row?.id),
    productName: s(row?.productName),
    brandId: s(row?.brandId),
    brandName: row?.brandName ? s(row.brandName) : undefined,
    assigneeId: s(row?.assigneeId),
    assigneeName: row?.assigneeName ? s(row.assigneeName) : undefined,
  }));

  console.log("[inventory/fetchPrintedInventorySummaries] ok", {
    count: mapped.length,
    sample: mapped.slice(0, 5),
  });

  return mapped;
}

// ---------------------------------------------------------
// ✅ inventoryIds 解決 DTO（方針A）
// GET /inventory/ids?productBlueprintId=...&tokenBlueprintId=...
// ---------------------------------------------------------
export type InventoryIDsByProductAndTokenDTO = {
  productBlueprintId: string;
  tokenBlueprintId: string;
  inventoryIds: string[];
};

export async function fetchInventoryIDsByProductAndTokenDTO(
  productBlueprintId: string,
  tokenBlueprintId: string,
): Promise<InventoryIDsByProductAndTokenDTO> {
  const pbId = s(productBlueprintId);
  const tbId = s(tokenBlueprintId);
  if (!pbId) throw new Error("productBlueprintId is empty");
  if (!tbId) throw new Error("tokenBlueprintId is empty");

  const token = await getIdTokenOrThrow();

  const url = `${API_BASE}/inventory/ids?productBlueprintId=${encodeURIComponent(
    pbId,
  )}&tokenBlueprintId=${encodeURIComponent(tbId)}`;

  console.log("[inventory/fetchInventoryIDsByProductAndTokenDTO] start", { pbId, tbId, url });

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    console.error("[inventory/fetchInventoryIDsByProductAndTokenDTO] failed", {
      pbId,
      tbId,
      url,
      status: res.status,
      statusText: res.statusText,
      body: text,
    });
    throw new Error(`Failed to fetch inventory ids: ${res.status} ${res.statusText}`);
  }

  const data = await res.json();

  const idsRaw = Array.isArray(data) ? data : data?.inventoryIds;
  const inventoryIds = Array.isArray(idsRaw)
    ? idsRaw.map((x: any) => s(x)).filter(Boolean)
    : [];

  const mapped: InventoryIDsByProductAndTokenDTO = {
    productBlueprintId: pbId,
    tokenBlueprintId: tbId,
    inventoryIds,
  };

  console.log("[inventory/fetchInventoryIDsByProductAndTokenDTO] ok", {
    pbId,
    tbId,
    count: mapped.inventoryIds.length,
    sample: mapped.inventoryIds.slice(0, 10),
  });

  return mapped;
}
