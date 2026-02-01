// frontend/console/inventory/src/infrastructure/http/inventoryRepositoryHTTP.mappers.ts

import type {
  InventoryListRowDTO,
  ProductBlueprintPatchDTO,
  TokenBlueprintPatchDTO,
  ListCreateDTO,
  ListCreatePriceRowDTO,
  InventoryDetailDTO,
  InventoryDetailRowDTO,
  InventoryProductSummary,
  InventoryIDsByProductAndTokenDTO,
} from "./inventoryRepositoryHTTP.types";

import { s, n, toRgbNumberOrNull, toProductIdTagString } from "./inventoryRepositoryHTTP.utils";

// =========================================================
// ✅ B案（暫定）: /inventory だけで回す前提で縮小（実測ログ準拠）
// - 互換（揺れ吸収 / Pascal / snake / 別名）は削除
// - 実際に参照されているキーだけ読む
// - ただし別画面を壊しにくいよう、"存在している機能" は残す（縮小版）
// =========================================================

// ---------------------------------------------------------
// Inventory List Row mapper（縮小）
// 期待 row:
// { productBlueprintId, productName, tokenBlueprintId, tokenName, modelNumber, availableStock, reservedCount }
// ---------------------------------------------------------
export function normalizeInventoryListRow(raw: any): InventoryListRowDTO | null {
  const productBlueprintId = s(raw?.productBlueprintId);
  const tokenBlueprintId = s(raw?.tokenBlueprintId);

  if (!productBlueprintId || !tokenBlueprintId) return null;

  const productName = s(raw?.productName);
  const tokenName = s(raw?.tokenName);
  const modelNumber = s(raw?.modelNumber);

  const availableStock = n(raw?.availableStock); // 実測ログ: availableStock が来ている
  const reservedCount = n(raw?.reservedCount);

  return {
    productBlueprintId,
    productName,
    tokenBlueprintId,
    tokenName,
    modelNumber,
    availableStock,
    reservedCount,
  };
}

// ---------------------------------------------------------
// ProductBlueprintPatch mapper（縮小）
// 実測ログの merged vm が参照するのは productName/brandId/brandName 程度だが、
// 既存型に合わせて最小限 + productIdTag だけ維持。
// ---------------------------------------------------------
export function mapProductBlueprintPatch(raw: any): ProductBlueprintPatchDTO {
  const p = (raw ?? {}) as any;

  return {
    productName: p.productName !== undefined ? (p.productName as any) : undefined,
    brandName: p.brandName !== undefined ? (p.brandName as any) : undefined,

    // 他画面で落ちやすいので最低限は残す（互換吸収はしない）
    itemType: p.itemType !== undefined ? String(p.itemType) : undefined,
    fit: p.fit !== undefined ? (p.fit as any) : undefined,
    material: p.material !== undefined ? (p.material as any) : undefined,
    weight: p.weight !== undefined && p.weight !== null ? Number(p.weight) : undefined,
    qualityAssurance: Array.isArray(p.qualityAssurance)
      ? p.qualityAssurance.map((x: any) => String(x))
      : undefined,

    productIdTag: toProductIdTagString(p.productIdTag),
  };
}

// ---------------------------------------------------------
// TokenBlueprintPatch mapper（縮小）
// 実測ログで参照されるキー（8個）に限定:
// tokenName, symbol, brandId, brandName, description, minted, metadataUri, iconUrl
// ---------------------------------------------------------
export function mapTokenBlueprintPatch(raw: any): TokenBlueprintPatchDTO | undefined {
  if (raw === undefined || raw === null) return undefined;

  const p = raw as any;

  const mintedRaw = p?.minted;
  const minted: boolean | null | undefined =
    mintedRaw === undefined
      ? undefined
      : mintedRaw === null
        ? null
        : typeof mintedRaw === "boolean"
          ? mintedRaw
          : String(mintedRaw).trim().toLowerCase() === "true";

  const iconUrl = s(p?.iconUrl);
  const metadataUri = s(p?.metadataUri);

  return {
    tokenName: p?.tokenName !== undefined ? (p.tokenName as any) : undefined,
    symbol: p?.symbol !== undefined ? (p.symbol as any) : undefined,
    brandId: p?.brandId !== undefined ? (p.brandId as any) : undefined,
    brandName: p?.brandName !== undefined ? (p.brandName as any) : undefined,
    description: p?.description !== undefined ? (p.description as any) : undefined,
    iconUrl: iconUrl ? (iconUrl as any) : undefined,
  };
}

// ---------------------------------------------------------
// Product summary mapper（縮小）
// B案: /inventory の row から printed summaries を作る前提
// 期待 row: { productBlueprintId, productName }
// ---------------------------------------------------------
export function mapPrintedInventorySummaries(data: any): InventoryProductSummary[] {
  if (!Array.isArray(data)) return [];

  const byPbId = new Map<string, InventoryProductSummary>();

  for (const row of data) {
    const id = s(row?.productBlueprintId);
    if (!id) continue;

    if (!byPbId.has(id)) {
      byPbId.set(id, {
        id,
        productName: s(row?.productName) || "-",
      });
    }
  }

  return Array.from(byPbId.values());
}

// ---------------------------------------------------------
// Inventory IDs mapper（縮小）
// 期待 shape: { inventoryIds: string[] }
// ---------------------------------------------------------
export function mapInventoryIDsByProductAndToken(
  productBlueprintId: string,
  tokenBlueprintId: string,
  data: any,
): InventoryIDsByProductAndTokenDTO {
  const idsRaw = data?.inventoryIds;
  const inventoryIds = Array.isArray(idsRaw)
    ? idsRaw.map((x: any) => s(x)).filter(Boolean)
    : [];

  return {
    productBlueprintId,
    tokenBlueprintId,
    inventoryIds,
  };
}

// ---------------------------------------------------------
// ListCreate mapper（縮小）
// 互換吸収を削除し、想定キーのみ読む
// ---------------------------------------------------------
export function mapListCreateDTO(data: any): ListCreateDTO {
  const rawRows: any[] = Array.isArray(data?.priceRows) ? data.priceRows : [];

  const priceRows: ListCreatePriceRowDTO[] = rawRows.flatMap((r: any) => {
    const modelId = s(r?.modelId);
    if (!modelId) return [];

    const rgbVal = toRgbNumberOrNull(r?.rgb);
    const stock = n(r?.stock);

    const hasPriceField = r?.price !== undefined;
    const rawPrice = r?.price;
    const price: number | null | undefined =
      !hasPriceField ? undefined : rawPrice === null ? null : n(rawPrice);

    const row: ListCreatePriceRowDTO = {
      modelId,
      size: s(r?.size) || "-",
      color: s(r?.color) || "-",
      stock,
      ...(rgbVal === undefined ? {} : { rgb: rgbVal }),
      ...(price === undefined ? {} : { price }),
    };

    return [row];
  });

  const totalStockRaw = data?.totalStock;

  return {
    inventoryId: data?.inventoryId ? s(data.inventoryId) : undefined,
    productBlueprintId: data?.productBlueprintId ? s(data.productBlueprintId) : undefined,
    tokenBlueprintId: data?.tokenBlueprintId ? s(data.tokenBlueprintId) : undefined,

    productBrandName: s(data?.productBrandName),
    productName: s(data?.productName),

    tokenBrandName: s(data?.tokenBrandName),
    tokenName: s(data?.tokenName),

    priceRows,
    totalStock:
      totalStockRaw === undefined || totalStockRaw === null ? undefined : n(totalStockRaw),
  };
}

// ---------------------------------------------------------
// InventoryDetail mapper（縮小）
// 実測ログで rows は { token:'-', modelNumber,size,color,rgb,stock } を参照。
// tokenBlueprintId は undefined でもOK（実測ログで undefined）
// ---------------------------------------------------------
export function mapInventoryDetailDTO(data: any, requestedId: string): InventoryDetailDTO {
  const patch = mapProductBlueprintPatch(data?.productBlueprintPatch);
  const tokenBlueprintPatch = mapTokenBlueprintPatch(data?.tokenBlueprintPatch);

  const rows: InventoryDetailRowDTO[] = Array.isArray(data?.rows)
    ? data.rows.map((r: any) => ({
        tokenBlueprintId: r?.tokenBlueprintId ? s(r.tokenBlueprintId) : undefined,
        token: r?.token ? s(r.token) : undefined,
        modelNumber: s(r?.modelNumber),
        size: s(r?.size),
        color: s(r?.color),
        rgb: toRgbNumberOrNull(r?.rgb),
        stock: n(r?.stock),
      }))
    : [];

  return {
    inventoryId: s(data?.inventoryId ?? requestedId),
    inventoryIds: Array.isArray(data?.inventoryIds)
      ? data.inventoryIds.map((x: any) => s(x)).filter(Boolean)
      : undefined,

    tokenBlueprintId: s(data?.tokenBlueprintId),
    productBlueprintId: s(data?.productBlueprintId),
    modelId: s(data?.modelId),

    productBlueprintPatch: patch,
    tokenBlueprintPatch,

    // ✅ この2つは実測ログ上、vm 側で必須ではないので「そのまま通す」だけ
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
    totalStock: n(data?.totalStock),
    updatedAt: data?.updatedAt ? String(data.updatedAt) : undefined,
  };
}
