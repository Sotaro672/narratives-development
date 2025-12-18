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

  console.log("[inventory/fetchInventoryProductSummary] start", {
    productBlueprintId: pbId,
    url,
  });

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

  console.log("[inventory/fetchInventoryProductSummary] ok", {
    productBlueprintId: pbId,
    mapped,
  });

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

// ---------------------------------------------------------
// ✅ Inventory Detail DTOs (exported)
// GET /inventory/{inventoryId}
// ---------------------------------------------------------
export type TokenBlueprintSummaryDTO = {
  id: string;
  name?: string;
  symbol?: string;
};

export type ProductBlueprintSummaryDTO = {
  id: string;
  name?: string;
};

export type ProductBlueprintPatchDTO = {
  productName?: string | null;
  brandId?: string | null;
  itemType?: string | null;
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;
  productIdTag?: any;
  assigneeId?: string | null;
};

export type InventoryDetailRowDTO = {
  tokenBlueprintId?: string;
  token?: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | null;
  stock: number;
};

export type InventoryDetailDTO = {
  inventoryId: string;

  // ✅ NEW: backend DTO が返す場合がある（方針A）
  inventoryIds?: string[];

  tokenBlueprintId: string;
  productBlueprintId: string;
  modelId: string;

  productBlueprintPatch: ProductBlueprintPatchDTO;

  tokenBlueprint?: TokenBlueprintSummaryDTO;
  productBlueprint?: ProductBlueprintSummaryDTO;

  rows: InventoryDetailRowDTO[];
  totalStock: number;

  updatedAt?: string;
};

function toOptionalString(v: any): string | undefined {
  const x = s(v);
  return x ? x : undefined;
}

function toRgbNumberOrNull(v: any): number | null | undefined {
  if (v === undefined) return undefined;
  if (v === null) return null;

  if (typeof v === "number" && Number.isFinite(v)) return v;

  const str = s(v);
  if (!str) return null;

  const normalized = str.replace(/^#/, "").replace(/^0x/i, "");
  if (/^[0-9a-fA-F]{6}$/.test(normalized)) {
    const nn = Number.parseInt(normalized, 16);
    return Number.isFinite(nn) ? nn : null;
  }

  const d = Number.parseInt(str, 10);
  return Number.isFinite(d) ? d : null;
}

function mapProductBlueprintPatch(raw: any): ProductBlueprintPatchDTO {
  const patchRaw = (raw ?? {}) as any;

  return {
    productName:
      patchRaw.productName !== undefined ? (patchRaw.productName as any) : undefined,
    brandId: patchRaw.brandId !== undefined ? (patchRaw.brandId as any) : undefined,
    itemType: patchRaw.itemType !== undefined ? String(patchRaw.itemType) : undefined,
    fit: patchRaw.fit !== undefined ? (patchRaw.fit as any) : undefined,
    material: patchRaw.material !== undefined ? (patchRaw.material as any) : undefined,
    weight:
      patchRaw.weight !== undefined && patchRaw.weight !== null
        ? Number(patchRaw.weight)
        : undefined,
    qualityAssurance: Array.isArray(patchRaw.qualityAssurance)
      ? patchRaw.qualityAssurance.map((x: any) => String(x))
      : undefined,
    productIdTag: patchRaw.productIdTag !== undefined ? patchRaw.productIdTag : undefined,
    assigneeId:
      patchRaw.assigneeId !== undefined ? (patchRaw.assigneeId as any) : undefined,
  };
}

/**
 * ✅ Inventory Detail DTO
 * GET /inventory/{inventoryId}
 */
export async function fetchInventoryDetailDTO(inventoryId: string): Promise<InventoryDetailDTO> {
  const id = s(inventoryId);
  if (!id) throw new Error("inventoryId is empty");

  const token = await getIdTokenOrThrow();
  const url = `${API_BASE}/inventory/${encodeURIComponent(id)}`;

  console.log("[inventory/fetchInventoryDetailDTO] start", { id, url });

  const res = await fetch(url, {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    console.error("[inventory/fetchInventoryDetailDTO] failed", {
      id,
      url,
      status: res.status,
      statusText: res.statusText,
      body: text,
    });
    throw new Error(
      `Failed to fetch inventory detail: ${res.status} ${res.statusText} ${text}`,
    );
  }

  const data = await res.json();

  const rows: InventoryDetailRowDTO[] = Array.isArray(data?.rows)
    ? data.rows.map((r: any) => ({
        tokenBlueprintId: toOptionalString(
          r?.tokenBlueprintId ?? r?.TokenBlueprintID ?? r?.token_blueprint_id,
        ),
        token: toOptionalString(r?.token ?? r?.Token),
        modelNumber: s(r?.modelNumber ?? r?.ModelNumber),
        size: s(r?.size ?? r?.Size),
        color: s(r?.color ?? r?.Color),
        rgb: toRgbNumberOrNull(r?.rgb ?? r?.RGB),
        stock: Number(r?.stock ?? r?.Stock ?? 0),
      }))
    : [];

  const mapped: InventoryDetailDTO = {
    inventoryId: s(data?.inventoryId ?? data?.id ?? id),
    inventoryIds: Array.isArray(data?.inventoryIds)
      ? data.inventoryIds.map((x: any) => s(x)).filter(Boolean)
      : undefined,

    tokenBlueprintId: s(data?.tokenBlueprintId ?? data?.TokenBlueprintID),
    productBlueprintId: s(data?.productBlueprintId ?? data?.ProductBlueprintID),
    modelId: s(data?.modelId ?? data?.ModelID),

    productBlueprintPatch: mapProductBlueprintPatch(data?.productBlueprintPatch),

    tokenBlueprint: data?.tokenBlueprint
      ? {
          id: s(data.tokenBlueprint.id),
          name: data.tokenBlueprint.name ? s(data.tokenBlueprint.name) : undefined,
          symbol: data.tokenBlueprint.symbol ? s(data.tokenBlueprint.symbol) : undefined,
        }
      : undefined,

    productBlueprint: data?.productBlueprint
      ? {
          id: s(data.productBlueprint.id),
          name: data.productBlueprint.name ? s(data.productBlueprint.name) : undefined,
        }
      : undefined,

    rows,
    totalStock: Number(data?.totalStock ?? 0),
    updatedAt: data?.updatedAt ? String(data.updatedAt) : undefined,
  };

  console.log("[inventory/fetchInventoryDetailDTO] ok", {
    id,
    rows: mapped.rows.length,
    totalStock: mapped.totalStock,
    updatedAt: mapped.updatedAt,
  });

  return mapped;
}
