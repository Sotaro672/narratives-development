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

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `Failed to fetch inventory list: ${res.status} ${res.statusText} ${text}`,
    );
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

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `Failed to fetch product blueprint: ${res.status} ${res.statusText} ${text}`,
    );
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
    const text = await res.text().catch(() => "");
    throw new Error(
      `Failed to fetch printed product blueprints: ${res.status} ${res.statusText} ${text}`,
    );
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

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `Failed to fetch inventory ids: ${res.status} ${res.statusText} ${text}`,
    );
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

// ✅ ProductBlueprintCard に合わせる（productIdTag は string）
export type ProductBlueprintPatchDTO = {
  productName?: string | null;

  // ✅ 追加: brandName を保持できるようにする（view モードで表示用）
  brandId?: string | null;
  brandName?: string | null;

  itemType?: string | null;
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;
  productIdTag?: string | null;
  assigneeId?: string | null;
};

// ✅ NEW: TokenBlueprint patch（Inventory 詳細で使用）
export type TokenBlueprintPatchDTO = {
  tokenName?: string | null;
  symbol?: string | null;
  brandId?: string | null;
  brandName?: string | null;
  description?: string | null;
  minted?: boolean | null;
  metadataUri?: string | null;
  iconUrl?: string | null;
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

  // ✅ NEW: tokenBlueprint patch を保持できるようにする
  tokenBlueprintPatch?: TokenBlueprintPatchDTO;

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

// ✅ productIdTag を "QRコード" のような表示用文字列に正規化する
function toProductIdTagString(v: any): string | null | undefined {
  if (v === undefined) return undefined;
  if (v === null) return null;

  // 既に文字列
  if (typeof v === "string") {
    const str = s(v);
    if (!str) return null;

    // JSON文字列っぽい場合は parse してから抽出を試す
    const looksJson = str.startsWith("{") && str.endsWith("}");
    if (looksJson) {
      try {
        const obj = JSON.parse(str);
        const fromObj = toProductIdTagString(obj);
        return fromObj ?? str;
      } catch {
        return str;
      }
    }
    return str;
  }

  // 数値/boolean 等
  if (typeof v === "number" || typeof v === "boolean") {
    return String(v);
  }

  // オブジェクト: { Type: "QRコード" } など
  if (typeof v === "object") {
    const o: any = v;

    const candidates = [
      o?.label,
      o?.Label,
      o?.type,
      o?.Type,
      o?.value,
      o?.Value,
      o?.name,
      o?.Name,
    ];

    for (const c of candidates) {
      const str = s(c);
      if (str) return str;
    }

    // どうしても取れない場合は null（"[object Object]" を出さない）
    return null;
  }

  return null;
}

function mapProductBlueprintPatch(raw: any): ProductBlueprintPatchDTO {
  const patchRaw = (raw ?? {}) as any;

  return {
    productName:
      patchRaw.productName !== undefined ? (patchRaw.productName as any) : undefined,

    brandId: patchRaw.brandId !== undefined ? (patchRaw.brandId as any) : undefined,
    // ✅ brandName も保持（無ければ undefined のまま）
    brandName:
      patchRaw.brandName !== undefined
        ? (patchRaw.brandName as any)
        : patchRaw.brand !== undefined
          ? (patchRaw.brand as any)
          : undefined,

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

    // ✅ object → "QRコード" に変換
    productIdTag: toProductIdTagString(patchRaw.productIdTag),

    assigneeId:
      patchRaw.assigneeId !== undefined ? (patchRaw.assigneeId as any) : undefined,
  };
}

function mapTokenBlueprintPatch(raw: any): TokenBlueprintPatchDTO | undefined {
  if (raw === undefined) return undefined;
  if (raw === null) return undefined;

  const p = raw as any;

  const mintedRaw = p?.minted;
  const minted =
    mintedRaw === undefined
      ? undefined
      : mintedRaw === null
        ? null
        : typeof mintedRaw === "boolean"
          ? mintedRaw
          : String(mintedRaw).trim().toLowerCase() === "true";

  const iconUrl = s(p?.iconUrl);
  const metadataUri = s(p?.metadataUri ?? p?.metadataURI);

  return {
    tokenName:
      p?.tokenName !== undefined
        ? (p.tokenName as any)
        : p?.name !== undefined
          ? (p.name as any)
          : undefined,
    symbol: p?.symbol !== undefined ? (p.symbol as any) : undefined,
    brandId: p?.brandId !== undefined ? (p.brandId as any) : undefined,
    brandName: p?.brandName !== undefined ? (p.brandName as any) : undefined,
    description: p?.description !== undefined ? (p.description as any) : undefined,
    minted: minted as any,
    metadataUri: metadataUri ? metadataUri : undefined,
    iconUrl: iconUrl ? iconUrl : undefined,
  };
}

/**
 * ✅ NEW: TokenBlueprint Patch DTO
 * GET /token-blueprints/{tokenBlueprintId}/patch
 *
 * - inventoryDetailService が import して呼べるように export する
 * - backend が未実装の場合は 404/501 等になるので、呼び出し側で try/catch する想定
 */
export async function fetchTokenBlueprintPatchDTO(
  tokenBlueprintId: string,
): Promise<TokenBlueprintPatchDTO> {
  const token = await getIdTokenOrThrow();

  const tbId = s(tokenBlueprintId);
  if (!tbId) throw new Error("tokenBlueprintId is empty");

  const url = `${API_BASE}/token-blueprints/${encodeURIComponent(tbId)}/patch`;

  const res = await fetch(url, {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `Failed to fetch token blueprint patch: ${res.status} ${res.statusText} ${text}`,
    );
  }

  const data = await res.json();
  const patch = mapTokenBlueprintPatch(data) ?? {};
  return patch;
}

/**
 * ✅ Inventory Detail DTO
 * GET /inventory/{inventoryId}
 */
export async function fetchInventoryDetailDTO(
  inventoryId: string,
): Promise<InventoryDetailDTO> {
  const id = s(inventoryId);
  if (!id) throw new Error("inventoryId is empty");

  const token = await getIdTokenOrThrow();
  const url = `${API_BASE}/inventory/${encodeURIComponent(id)}`;

  const res = await fetch(url, {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `Failed to fetch inventory detail: ${res.status} ${res.statusText} ${text}`,
    );
  }

  const data = await res.json();

  const patch = mapProductBlueprintPatch(data?.productBlueprintPatch);
  const tokenBlueprintPatch = mapTokenBlueprintPatch(data?.tokenBlueprintPatch);

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

    productBlueprintPatch: patch,

    // ✅ tokenBlueprintPatch を持てている（backend が返せば格納される）
    tokenBlueprintPatch,

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

  return mapped;
}
