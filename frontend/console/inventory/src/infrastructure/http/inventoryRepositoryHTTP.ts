// frontend/console/inventory/src/infrastructure/http/inventoryRepositoryHTTP.ts

// ✅ API_BASE 互換が必要な箇所があるかもしれないので re-export（任意）
export { API_BASE } from "../api/inventoryApi";

import {
  getInventoryListRaw,
  getProductBlueprintRaw,
  getPrintedProductBlueprintsRaw,
  getInventoryIDsByProductAndTokenRaw,
  getTokenBlueprintPatchRaw,
  getListCreateRaw,
  getInventoryDetailRaw,
} from "../api/inventoryApi";

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
  const data = (await getInventoryListRaw()) as any;

  // ✅ 互換吸収を減らす：基本は配列を期待。どうしても違う場合のみ items を許容。
  const rawItems: any[] = Array.isArray(data)
    ? data
    : Array.isArray(data?.items)
      ? data.items
      : [];

  return rawItems
    .map(normalizeInventoryListRow)
    .filter((x): x is InventoryListRowDTO => x !== null);
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
  const pbId = s(productBlueprintId);
  if (!pbId) throw new Error("productBlueprintId is empty");

  const data = await getProductBlueprintRaw(pbId);

  return {
    id: s(data?.id),
    productName: s(data?.productName),
    brandId: s(data?.brandId),
    brandName: data?.brandName ? s(data.brandName) : undefined,
    assigneeId: s(data?.assigneeId),
    assigneeName: data?.assigneeName ? s(data.assigneeName) : undefined,
  };
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
  const data = await getPrintedProductBlueprintsRaw();
  if (!Array.isArray(data)) return [];

  return data.map((row: any) => ({
    id: s(row?.id),
    productName: s(row?.productName),
    brandId: s(row?.brandId),
    brandName: row?.brandName ? s(row.brandName) : undefined,
    assigneeId: s(row?.assigneeId),
    assigneeName: row?.assigneeName ? s(row.assigneeName) : undefined,
  }));
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

  const data = await getInventoryIDsByProductAndTokenRaw({
    productBlueprintId: pbId,
    tokenBlueprintId: tbId,
  });

  const idsRaw = Array.isArray(data) ? data : data?.inventoryIds;
  const inventoryIds = Array.isArray(idsRaw)
    ? idsRaw.map((x: any) => s(x)).filter(Boolean)
    : [];

  return {
    productBlueprintId: pbId,
    tokenBlueprintId: tbId,
    inventoryIds,
  };
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
 */
export async function fetchTokenBlueprintPatchDTO(
  tokenBlueprintId: string,
): Promise<TokenBlueprintPatchDTO> {
  const tbId = s(tokenBlueprintId);
  if (!tbId) throw new Error("tokenBlueprintId is empty");

  const data = await getTokenBlueprintPatchRaw(tbId);
  return mapTokenBlueprintPatch(data) ?? {};
}

/**
 * ✅ ListCreate DTO（出品作成画面）
 * backend/internal/application/query/dto/list_create_dto.go と対応
 */
export type ListCreateDTO = {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;

  productBrandName: string;
  productName: string;

  tokenBrandName: string;
  tokenName: string;
};

/**
 * ✅ ListCreate DTO 取得
 * GET
 * - /inventory/list-create/:inventoryId
 * - /inventory/list-create/:productBlueprintId/:tokenBlueprintId
 */
export async function fetchListCreateDTO(input: {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
}): Promise<ListCreateDTO> {
  const data = await getListCreateRaw(input);

  return {
    inventoryId: data?.inventoryId ? s(data.inventoryId) : undefined,
    productBlueprintId: data?.productBlueprintId ? s(data.productBlueprintId) : undefined,
    tokenBlueprintId: data?.tokenBlueprintId ? s(data.tokenBlueprintId) : undefined,

    productBrandName: s(data?.productBrandName),
    productName: s(data?.productName),

    tokenBrandName: s(data?.tokenBrandName),
    tokenName: s(data?.tokenName),
  };
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

  const data = await getInventoryDetailRaw(id);

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

  return {
    inventoryId: s(data?.inventoryId ?? data?.id ?? id),
    inventoryIds: Array.isArray(data?.inventoryIds)
      ? data.inventoryIds.map((x: any) => s(x)).filter(Boolean)
      : undefined,

    tokenBlueprintId: s(data?.tokenBlueprintId ?? data?.TokenBlueprintID),
    productBlueprintId: s(data?.productBlueprintId ?? data?.ProductBlueprintID),
    modelId: s(data?.modelId ?? data?.ModelID),

    productBlueprintPatch: patch,
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
}
