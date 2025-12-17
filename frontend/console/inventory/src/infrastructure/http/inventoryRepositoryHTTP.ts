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

  console.log("[inventory/fetchInventoryProductSummary] start", {
    productBlueprintId,
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
      productBlueprintId,
      url,
      status: res.status,
      statusText: res.statusText,
      body: text,
    });
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

  console.log("[inventory/fetchInventoryProductSummary] ok", {
    productBlueprintId,
    mapped,
    raw: data,
  });

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
    throw new Error(
      `Failed to fetch printed product blueprints: ${res.status} ${res.statusText}`,
    );
  }

  const data = await res.json();

  if (!Array.isArray(data)) {
    console.warn("[inventory/fetchPrintedInventorySummaries] ok but not array", {
      url,
      rawType: typeof data,
      raw: data,
    });
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

  console.log("[inventory/fetchPrintedInventorySummaries] ok", {
    count: mapped.length,
    sample: mapped.slice(0, 5),
  });

  return mapped;
}

// ---------------------------------------------------------
// Inventory Detail DTO (from inventory_query)
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
  token?: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | null; // ✅ backend は *int を返す想定
  stock: number;
};

export type InventoryDetailDTO = {
  inventoryId: string;

  // pbId query の場合は空になり得る（backend 仕様）
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

// ---------------------------------------------------------
// ✅ NEW: inventoryIds 解決 DTO（方針A）
// ---------------------------------------------------------

export type InventoryIDsByProductAndTokenDTO = {
  productBlueprintId: string;
  tokenBlueprintId: string;
  inventoryIds: string[];
};

/**
 * ✅ NEW（方針A）:
 * pbId + tokenBlueprintId から inventoryIds を解決
 *
 * GET /inventory/ids?productBlueprintId={pbId}&tokenBlueprintId={tbId}
 */
export async function fetchInventoryIDsByProductAndTokenDTO(
  productBlueprintId: string,
  tokenBlueprintId: string,
): Promise<InventoryIDsByProductAndTokenDTO> {
  const pbId = String(productBlueprintId ?? "").trim();
  const tbId = String(tokenBlueprintId ?? "").trim();
  if (!pbId) throw new Error("productBlueprintId is empty");
  if (!tbId) throw new Error("tokenBlueprintId is empty");

  const token = await getIdTokenOrThrow();

  const url = `${API_BASE}/inventory/ids?productBlueprintId=${encodeURIComponent(
    pbId,
  )}&tokenBlueprintId=${encodeURIComponent(tbId)}`;

  console.log("[inventory/fetchInventoryIDsByProductAndTokenDTO] start", {
    pbId,
    tbId,
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
    console.error("[inventory/fetchInventoryIDsByProductAndTokenDTO] failed", {
      pbId,
      tbId,
      url,
      status: res.status,
      statusText: res.statusText,
      body: text,
    });
    throw new Error(
      `Failed to fetch inventory ids: ${res.status} ${res.statusText}`,
    );
  }

  const data = await res.json();

  // 返却形式が「配列だけ」でも「{inventoryIds:[...] }」でも受けられるようにする
  const idsRaw = Array.isArray(data) ? data : (data?.inventoryIds ?? []);
  const inventoryIds = Array.isArray(idsRaw)
    ? idsRaw.map((x: any) => String(x ?? "").trim()).filter(Boolean)
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
    raw: data,
  });

  return mapped;
}

// ---------------------------------------------------------
// Helpers (DTO mapping)
// ---------------------------------------------------------

function toOptionalString(v: any): string | undefined {
  if (v === undefined || v === null) return undefined;
  const s = String(v).trim();
  return s ? s : undefined;
}

function toRgbNumberOrNull(v: any): number | null | undefined {
  if (v === undefined) return undefined;
  if (v === null) return null;

  if (typeof v === "number" && Number.isFinite(v)) return v;

  const s = String(v).trim();
  if (!s) return null;

  // "0xRRGGBB" / "#RRGGBB" / "RRGGBB"
  const normalized = s.replace(/^#/, "").replace(/^0x/i, "");
  if (/^[0-9a-fA-F]{6}$/.test(normalized)) {
    const n = Number.parseInt(normalized, 16);
    return Number.isFinite(n) ? n : null;
  }

  // decimal string
  const d = Number.parseInt(s, 10);
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
    productIdTag:
      patchRaw.productIdTag !== undefined ? patchRaw.productIdTag : undefined,
    assigneeId:
      patchRaw.assigneeId !== undefined ? (patchRaw.assigneeId as any) : undefined,
  };
}

/**
 * ✅ NEW (期待値):
 * productBlueprintId で Inventory detail DTO を取得
 *
 * GET /inventory?productBlueprintId={pbId}
 */
export async function fetchInventoryDetailDTOByProductBlueprintId(
  productBlueprintId: string,
): Promise<InventoryDetailDTO> {
  const pbId = String(productBlueprintId ?? "").trim();
  if (!pbId) {
    throw new Error("productBlueprintId is empty");
  }

  const token = await getIdTokenOrThrow();

  const url = `${API_BASE}/inventory?productBlueprintId=${encodeURIComponent(pbId)}`;

  console.log("[inventory/fetchInventoryDetailDTOByProductBlueprintId] start", {
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
    console.error("[inventory/fetchInventoryDetailDTOByProductBlueprintId] failed", {
      productBlueprintId: pbId,
      url,
      status: res.status,
      statusText: res.statusText,
      body: text,
    });
    throw new Error(
      `Failed to fetch inventory detail by productBlueprintId: ${res.status} ${res.statusText}`,
    );
  }

  const data = await res.json();

  console.log("[inventory/fetchInventoryDetailDTOByProductBlueprintId] raw", {
    productBlueprintId: pbId,
    raw: data,
  });

  const rows: InventoryDetailRowDTO[] = Array.isArray(data?.rows)
    ? data.rows.map((r: any) => ({
        token: toOptionalString(r?.token),
        modelNumber: String(r?.modelNumber ?? ""),
        size: String(r?.size ?? ""),
        color: String(r?.color ?? ""),
        rgb: toRgbNumberOrNull(r?.rgb),
        stock: Number(r?.stock ?? 0),
      }))
    : [];

  const mapped: InventoryDetailDTO = {
    // backend の互換仕様: pbId query の場合 inventoryId に pbId が入る
    inventoryId: String(data?.inventoryId ?? data?.id ?? pbId),

    tokenBlueprintId: String(data?.tokenBlueprintId ?? ""),
    productBlueprintId: String(data?.productBlueprintId ?? pbId),
    modelId: String(data?.modelId ?? ""),

    productBlueprintPatch: mapProductBlueprintPatch(data?.productBlueprintPatch),

    tokenBlueprint: data?.tokenBlueprint
      ? {
          id: String(data.tokenBlueprint.id ?? ""),
          name: data.tokenBlueprint.name ? String(data.tokenBlueprint.name) : undefined,
          symbol: data.tokenBlueprint.symbol
            ? String(data.tokenBlueprint.symbol)
            : undefined,
        }
      : undefined,

    productBlueprint: data?.productBlueprint
      ? {
          id: String(data.productBlueprint.id ?? ""),
          name: data.productBlueprint.name
            ? String(data.productBlueprint.name)
            : undefined,
        }
      : undefined,

    rows,
    totalStock: Number(data?.totalStock ?? 0),
    updatedAt: data?.updatedAt ? String(data.updatedAt) : undefined,
  };

  console.log("[inventory/fetchInventoryDetailDTOByProductBlueprintId] ok", {
    productBlueprintId: pbId,
    inventoryId: mapped.inventoryId,
    tokenBlueprintId: mapped.tokenBlueprintId,
    modelId: mapped.modelId,
    totalStock: mapped.totalStock,
    rowsCount: mapped.rows.length,
    rowsSample: mapped.rows.slice(0, 5),
    productBlueprintPatch: mapped.productBlueprintPatch,
    tokenBlueprint: mapped.tokenBlueprint,
    productBlueprint: mapped.productBlueprint,
    updatedAt: mapped.updatedAt,
  });

  return mapped;
}

/**
 * 互換: 在庫詳細画面（閲覧専用）のDTOを取得
 *
 * GET /inventory/{inventoryId}
 */
export async function fetchInventoryDetailDTO(
  inventoryId: string,
): Promise<InventoryDetailDTO> {
  const id = String(inventoryId ?? "").trim();
  if (!id) {
    throw new Error("inventoryId is empty");
  }

  const token = await getIdTokenOrThrow();

  const url = `${API_BASE}/inventory/${encodeURIComponent(id)}`;

  console.log("[inventory/fetchInventoryDetailDTO] start", { inventoryId: id, url });

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    console.error("[inventory/fetchInventoryDetailDTO] failed", {
      inventoryId: id,
      url,
      status: res.status,
      statusText: res.statusText,
      body: text,
    });
    throw new Error(
      `Failed to fetch inventory detail: ${res.status} ${res.statusText}`,
    );
  }

  const data = await res.json();

  console.log("[inventory/fetchInventoryDetailDTO] raw", {
    inventoryId: id,
    raw: data,
  });

  const rows: InventoryDetailRowDTO[] = Array.isArray(data?.rows)
    ? data.rows.map((r: any) => ({
        token: toOptionalString(r?.token),
        modelNumber: String(r?.modelNumber ?? ""),
        size: String(r?.size ?? ""),
        color: String(r?.color ?? ""),
        rgb: toRgbNumberOrNull(r?.rgb),
        stock: Number(r?.stock ?? 0),
      }))
    : [];

  const mapped: InventoryDetailDTO = {
    inventoryId: String(data?.inventoryId ?? data?.id ?? id),

    tokenBlueprintId: String(data?.tokenBlueprintId ?? ""),
    productBlueprintId: String(data?.productBlueprintId ?? ""),
    modelId: String(data?.modelId ?? ""),

    productBlueprintPatch: mapProductBlueprintPatch(data?.productBlueprintPatch),

    tokenBlueprint: data?.tokenBlueprint
      ? {
          id: String(data.tokenBlueprint.id ?? ""),
          name: data.tokenBlueprint.name ? String(data.tokenBlueprint.name) : undefined,
          symbol: data.tokenBlueprint.symbol
            ? String(data.tokenBlueprint.symbol)
            : undefined,
        }
      : undefined,

    productBlueprint: data?.productBlueprint
      ? {
          id: String(data.productBlueprint.id ?? ""),
          name: data.productBlueprint.name
            ? String(data.productBlueprint.name)
            : undefined,
        }
      : undefined,

    rows,
    totalStock: Number(data?.totalStock ?? 0),
    updatedAt: data?.updatedAt ? String(data.updatedAt) : undefined,
  };

  console.log("[inventory/fetchInventoryDetailDTO] ok", {
    inventoryId: id,
    tokenBlueprintId: mapped.tokenBlueprintId,
    productBlueprintId: mapped.productBlueprintId,
    modelId: mapped.modelId,
    totalStock: mapped.totalStock,
    rowsCount: mapped.rows.length,
    rowsSample: mapped.rows.slice(0, 5),
    productBlueprintPatch: mapped.productBlueprintPatch,
    tokenBlueprint: mapped.tokenBlueprint,
    productBlueprint: mapped.productBlueprint,
    updatedAt: mapped.updatedAt,
  });

  return mapped;
}
