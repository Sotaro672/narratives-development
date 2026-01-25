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

// ---------------------------------------------------------
// Inventory List Row mapper
// ---------------------------------------------------------
export function normalizeInventoryListRow(raw: any): InventoryListRowDTO | null {
  const productBlueprintId = s(raw?.productBlueprintId ?? raw?.productBlueprintID);
  const productName = s(raw?.productName);

  const tokenBlueprintId = s(raw?.tokenBlueprintId ?? raw?.tokenBlueprintID);
  const tokenName = s(raw?.tokenName);

  const modelNumber = s(raw?.modelNumber ?? raw?.modelNum);

  // ✅ 受け取り揺れ吸収（camel / Pascal / snake）
  const hasAvailableStock =
    raw?.availableStock !== undefined ||
    raw?.AvailableStock !== undefined ||
    raw?.available_stock !== undefined;

  const availableStockRaw = n(
    raw?.availableStock ?? raw?.AvailableStock ?? raw?.available_stock,
  );

  const stockRaw = n(raw?.stock ?? raw?.Stock ?? raw?.stock_count);

  const reservedCount = n(
    raw?.reservedCount ?? raw?.ReservedCount ?? raw?.reserved_count,
  );

  // ✅ 方針A: pbId/tbId は必須。ここで落とす（"-" 埋めはしない）
  if (!productBlueprintId || !tokenBlueprintId) return null;

  // ✅ stock は互換のため残すが、「在庫数(表示)」= availableStock を正とする
  // - availableStock が来ているならそれを採用（0でも採用）
  // - 来ていない場合のみ stock を availableStock とみなす
  const availableStock = hasAvailableStock ? availableStockRaw : stockRaw;

  return {
    productBlueprintId,
    productName,
    tokenBlueprintId,
    tokenName,
    modelNumber,

    // 互換: stock は availableStock と同義で返す
    stock: availableStock,

    availableStock,
    reservedCount,
  };
}

// ---------------------------------------------------------
// ProductBlueprintPatch mapper
// ---------------------------------------------------------
export function mapProductBlueprintPatch(raw: any): ProductBlueprintPatchDTO {
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

// ---------------------------------------------------------
// TokenBlueprintPatch mapper
// ---------------------------------------------------------
export function mapTokenBlueprintPatch(raw: any): TokenBlueprintPatchDTO | undefined {
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

// ---------------------------------------------------------
// Product summary mapper
// ---------------------------------------------------------
export function mapInventoryProductSummary(data: any): InventoryProductSummary {
  return {
    id: s(data?.id),
    productName: s(data?.productName),
    brandId: s(data?.brandId),
    brandName: data?.brandName ? s(data.brandName) : undefined,
    assigneeId: s(data?.assigneeId),
    assigneeName: data?.assigneeName ? s(data.assigneeName) : undefined,
  };
}

export function mapPrintedInventorySummaries(data: any): InventoryProductSummary[] {
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
// Inventory IDs mapper
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
// ListCreate mapper
// ---------------------------------------------------------
export function mapListCreateDTO(data: any): ListCreateDTO {
  const rawRows: any[] = Array.isArray(data?.priceRows)
    ? data.priceRows
    : Array.isArray(data?.PriceRows)
      ? data.PriceRows
      : [];

  // ✅ null を返さない（flatMap で安全に配列化）
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
      ...(rgbVal === undefined ? {} : { rgb: rgbVal }), // ✅ rgb が undefined のときはプロパティ自体を持たない
      ...(price === undefined ? {} : { price }), // ✅ price も同様
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

    // ✅ NEW
    priceRows,
    totalStock:
      totalStockRaw === undefined || totalStockRaw === null
        ? undefined
        : n(totalStockRaw),
  };
}

// ---------------------------------------------------------
// InventoryDetail mapper
// ---------------------------------------------------------
export function mapInventoryDetailDTO(data: any, requestedId: string): InventoryDetailDTO {
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
