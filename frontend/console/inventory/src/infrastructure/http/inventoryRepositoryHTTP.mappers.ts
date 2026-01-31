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

import {
  s,
  n,
  toOptionalString,
  toRgbNumberOrNull,
  toProductIdTagString,
} from "./inventoryRepositoryHTTP.utils";

// =========================================================
// ✅ B案（暫定）: /inventory だけで回す前提で縮小
// - 返却の shape が安定している前提で「揺れ吸収」を削除
// - ただし Detail / ListCreate は現状維持（縮小しすぎると別画面が壊れやすい）
// =========================================================

// ---------------------------------------------------------
// Inventory List Row mapper（縮小）
// ---------------------------------------------------------
export function normalizeInventoryListRow(raw: any): InventoryListRowDTO | null {
  const productBlueprintId = s(raw?.productBlueprintId);
  const tokenBlueprintId = s(raw?.tokenBlueprintId);

  // ✅ 必須
  if (!productBlueprintId || !tokenBlueprintId) return null;

  const productName = s(raw?.productName);
  const tokenName = s(raw?.tokenName);
  const modelNumber = s(raw?.modelNumber);

  // ✅ 在庫数(表示)は availableStock を正（無ければ stock にフォールバック）
  const availableStock = n(raw?.availableStock ?? raw?.stock);
  const reservedCount = n(raw?.reservedCount);

  return {
    productBlueprintId,
    productName,
    tokenBlueprintId,
    tokenName,
    modelNumber,
    stock: availableStock, // 互換: stock は (= availableStock)
    availableStock,
    reservedCount,
  };
}

// ---------------------------------------------------------
// ProductBlueprintPatch mapper（現状維持）
// ---------------------------------------------------------
export function mapProductBlueprintPatch(raw: any): ProductBlueprintPatchDTO {
  const patchRaw = (raw ?? {}) as any;

  return {
    productName:
      patchRaw.productName !== undefined ? (patchRaw.productName as any) : undefined,

    brandId: patchRaw.brandId !== undefined ? (patchRaw.brandId as any) : undefined,
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

    productIdTag: toProductIdTagString(patchRaw.productIdTag),

    assigneeId:
      patchRaw.assigneeId !== undefined ? (patchRaw.assigneeId as any) : undefined,
  };
}

// ---------------------------------------------------------
// TokenBlueprintPatch mapper（現状維持）
// ---------------------------------------------------------
export function mapTokenBlueprintPatch(raw: any): TokenBlueprintPatchDTO | undefined {
  if (raw === undefined || raw === null) return undefined;

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

// ---------------------------------------------------------
// Product summary mapper（縮小）
// - B案: /inventory の row から printed summaries を作る前提
// - types 的に brandId / assigneeId は必須なので "" を入れる
// ---------------------------------------------------------
export function mapInventoryProductSummary(data: any): InventoryProductSummary {
  // 互換用の単体mapper（今はほぼ使わない想定）
  return {
    id: s(data?.id ?? data?.productBlueprintId),
    productName: s(data?.productName),
    brandId: s(data?.brandId),
    brandName: data?.brandName ? s(data.brandName) : undefined,
    assigneeId: s(data?.assigneeId),
    assigneeName: data?.assigneeName ? s(data.assigneeName) : undefined,
  };
}

/**
 * B案: GET /inventory の配列を受け取り、
 * productBlueprintId 単位で dedup した product summaries を返す。
 *
 * 期待 row:
 * { productBlueprintId, productName, ... }
 */
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
        brandId: "",
        assigneeId: "",
      });
    }
  }

  return Array.from(byPbId.values());
}

// ---------------------------------------------------------
// Inventory IDs mapper（現状維持）
// ---------------------------------------------------------
export function mapInventoryIDsByProductAndToken(
  productBlueprintId: string,
  tokenBlueprintId: string,
  data: any,
): InventoryIDsByProductAndTokenDTO {
  const idsRaw = Array.isArray(data) ? data : data?.inventoryIds;
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
// ListCreate mapper（現状維持）
// ---------------------------------------------------------
export function mapListCreateDTO(data: any): ListCreateDTO {
  const rawRows: any[] = Array.isArray(data?.priceRows)
    ? data.priceRows
    : Array.isArray(data?.PriceRows)
      ? data.PriceRows
      : [];

  const priceRows: ListCreatePriceRowDTO[] = rawRows.flatMap((r: any) => {
    const modelId = s(r?.modelId ?? r?.ModelID ?? r?.modelID);
    if (!modelId) return [];

    const rgbVal = toRgbNumberOrNull(r?.rgb ?? r?.RGB);
    const stock = n(r?.stock ?? r?.Stock);

    const rawPrice = r?.price ?? r?.Price;
    const hasPriceField = r?.price !== undefined || r?.Price !== undefined;
    const price: number | null | undefined =
      !hasPriceField ? undefined : rawPrice === null ? null : n(rawPrice);

    const row: ListCreatePriceRowDTO = {
      modelId,
      size: s(r?.size ?? r?.Size) || "-",
      color: s(r?.color ?? r?.Color) || "-",
      stock,
      ...(rgbVal === undefined ? {} : { rgb: rgbVal }),
      ...(price === undefined ? {} : { price }),
    };

    return [row];
  });

  const totalStockRaw = data?.totalStock ?? data?.TotalStock;

  return {
    inventoryId: data?.inventoryId
      ? s(data.inventoryId)
      : data?.InventoryID
        ? s(data.InventoryID)
        : undefined,
    productBlueprintId: data?.productBlueprintId
      ? s(data.productBlueprintId)
      : data?.ProductBlueprintID
        ? s(data.ProductBlueprintID)
        : undefined,
    tokenBlueprintId: data?.tokenBlueprintId
      ? s(data.tokenBlueprintId)
      : data?.TokenBlueprintID
        ? s(data.TokenBlueprintID)
        : undefined,

    productBrandName: s(data?.productBrandName ?? data?.ProductBrandName),
    productName: s(data?.productName ?? data?.ProductName),

    tokenBrandName: s(data?.tokenBrandName ?? data?.TokenBrandName),
    tokenName: s(data?.tokenName ?? data?.TokenName),

    priceRows,
    totalStock:
      totalStockRaw === undefined || totalStockRaw === null
        ? undefined
        : n(totalStockRaw),
  };
}

// ---------------------------------------------------------
// InventoryDetail mapper（現状維持）
// ---------------------------------------------------------
export function mapInventoryDetailDTO(
  data: any,
  requestedId: string,
): InventoryDetailDTO {
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
    inventoryId: s(data?.inventoryId ?? data?.id ?? requestedId),
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
