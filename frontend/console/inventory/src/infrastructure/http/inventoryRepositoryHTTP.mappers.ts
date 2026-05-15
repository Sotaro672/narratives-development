// frontend/console/inventory/src/infrastructure/http/inventoryRepositoryHTTP.mappers.ts

import type {
  InventoryListRowDTO,
  ProductBlueprintPatchDTO,
  TokenBlueprintPatchDTO,
  InventoryDetailDTO,
  InventoryDetailRowDTO,
  InventoryProductSummary,
  InventoryIDsByProductAndTokenDTO,
} from "./inventoryRepositoryHTTP.types";

// =========================================================
// B案: /inventory だけで回す前提で縮小（実測ログ準拠）
// - 互換（揺れ吸収 / Pascal / snake / 別名）は削除
// - 実際に参照されている lower camel case のキーだけ読む
// - inventoryRepositoryHTTP.utils.ts への依存は廃止
// =========================================================

// ---------------------------------------------------------
// Inventory List Row mapper（縮小）
// 期待 row:
// {
//   productBlueprintId,
//   productName,
//   tokenBlueprintId,
//   tokenName,
//   modelNumber,
//   availableStock,
//   reservedCount
// }
// ---------------------------------------------------------
export function normalizeInventoryListRow(raw: any): InventoryListRowDTO | null {
  const productBlueprintId = raw?.productBlueprintId;
  const tokenBlueprintId = raw?.tokenBlueprintId;

  if (!productBlueprintId || !tokenBlueprintId) return null;

  return {
    productBlueprintId,
    productName: raw?.productName,
    tokenBlueprintId,
    tokenName: raw?.tokenName,
    modelNumber: raw?.modelNumber,
    availableStock: raw?.availableStock,
    reservedCount: raw?.reservedCount,
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
    productName: p.productName,
    brandName: p.brandName,

    // 他画面で落ちやすいので最低限は残す（互換吸収はしない）
    itemType: p.itemType,
    fit: p.fit,
    material: p.material,
    weight: p.weight,
    qualityAssurance: p.qualityAssurance,

    productIdTag: p.productIdTag,
  };
}

// ---------------------------------------------------------
// TokenBlueprintPatch mapper（縮小）
// 実測ログで参照されるキーに限定:
// tokenName, symbol, brandId, brandName, description, minted, metadataUri, iconUrl
// ---------------------------------------------------------
export function mapTokenBlueprintPatch(
  raw: any,
): TokenBlueprintPatchDTO | undefined {
  if (raw === undefined || raw === null) return undefined;

  const p = raw as any;

  return {
    tokenName: p.tokenName,
    symbol: p.symbol,
    brandId: p.brandId,
    brandName: p.brandName,
    description: p.description,
    iconUrl: p.iconUrl,
  };
}

// ---------------------------------------------------------
// Product summary mapper（縮小）
// B案: /inventory の row から printed summaries を作る前提
// 期待 row: { productBlueprintId, productName }
// ---------------------------------------------------------
export function mapPrintedInventorySummaries(
  data: any,
): InventoryProductSummary[] {
  if (!Array.isArray(data)) return [];

  const byPbId = new Map<string, InventoryProductSummary>();

  for (const row of data) {
    const id = row?.productBlueprintId;
    if (!id) continue;

    if (!byPbId.has(id)) {
      byPbId.set(id, {
        id,
        productName: row?.productName || "-",
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
    ? idsRaw.filter(Boolean)
    : [];

  return {
    productBlueprintId,
    tokenBlueprintId,
    inventoryIds,
  };
}

// ---------------------------------------------------------
// InventoryDetail mapper（縮小）
// 実測ログで rows は { token:'-', modelNumber, size, color, rgb, stock } を参照。
// tokenBlueprintId は undefined でもOK（実測ログで undefined）
// ---------------------------------------------------------
export function mapInventoryDetailDTO(
  data: any,
  requestedId: string,
): InventoryDetailDTO {
  const patch = mapProductBlueprintPatch(data?.productBlueprintPatch);
  const tokenBlueprintPatch = mapTokenBlueprintPatch(data?.tokenBlueprintPatch);

  const rows: InventoryDetailRowDTO[] = Array.isArray(data?.rows)
    ? data.rows.map((r: any) => ({
        tokenBlueprintId: r?.tokenBlueprintId,
        token: r?.token,
        modelNumber: r?.modelNumber,
        size: r?.size,
        color: r?.color,
        rgb: r?.rgb ?? null,
        stock: r?.stock,
      }))
    : [];

  return {
    inventoryId: data?.inventoryId ?? requestedId,
    inventoryIds: Array.isArray(data?.inventoryIds)
      ? data.inventoryIds.filter(Boolean)
      : undefined,

    tokenBlueprintId: data?.tokenBlueprintId,
    productBlueprintId: data?.productBlueprintId,
    modelId: data?.modelId,

    productBlueprintPatch: patch,
    tokenBlueprintPatch,

    // この2つは実測ログ上、vm 側で必須ではないので「そのまま通す」だけ
    tokenBlueprint: data?.tokenBlueprint
      ? {
          id: data.tokenBlueprint.id,
          name: data.tokenBlueprint.name,
          symbol: data.tokenBlueprint.symbol,
        }
      : undefined,

    productBlueprint: data?.productBlueprint
      ? {
          id: data.productBlueprint.id,
          name: data.productBlueprint.name,
        }
      : undefined,

    rows,
    totalStock: data?.totalStock,
    updatedAt: data?.updatedAt,
  };
}