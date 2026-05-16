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
import type {
  ProductBlueprintCategoryKind,
  ProductBlueprintCategorySnapshot,
} from "../../../../productBlueprint/src/domain/entity/productBlueprintCategory";

// =========================================================
// /inventory を正とする mapper
//
// 方針:
// - 後方互換の揺れ吸収はしない。
// - snake_case / 旧別名 / 旧 variation merge 前提は扱わない。
// - Inventory Detail は GET /inventory/{inventoryId} の response を唯一の正とする。
// - /models/by-blueprint/{productBlueprintId}/variations は呼ばない。
// =========================================================

function mapProductBlueprintCategory(
  raw: any,
): ProductBlueprintCategorySnapshot | null {
  if (!raw) return null;

  return {
    id: raw.ID,
    code: raw.Code,
    nameJa: raw.NameJa,
    nameEn: raw.NameEn,
    kind: raw.Kind as ProductBlueprintCategoryKind,
    path: Array.isArray(raw.Path) ? raw.Path : [],
    parentId: raw.ParentID ?? null,
  };
}

function mapProductIdTag(raw: any): { type?: string } | null {
  if (!raw) return null;

  return {
    type: raw.Type,
  };
}

// ---------------------------------------------------------
// Inventory List Row mapper
//
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
  const productBlueprintId = raw.productBlueprintId;
  const tokenBlueprintId = raw.tokenBlueprintId;

  if (!productBlueprintId || !tokenBlueprintId) return null;

  return {
    productBlueprintId,
    productName: raw.productName,
    tokenBlueprintId,
    tokenName: raw.tokenName,
    modelNumber: raw.modelNumber,
    availableStock: raw.availableStock,
    reservedCount: raw.reservedCount,
  };
}

// ---------------------------------------------------------
// ProductBlueprintPatch mapper
//
// backend raw:
// - productBlueprintCategory: ID / Code / NameJa / NameEn / Kind / Path
// - productIdTag: Type
// - modelRefs: ModelID / DisplayOrder
//
// frontend DTO:
// - productBlueprintCategory: id / code / nameJa / nameEn / kind / path
// - productIdTag: type
// - modelRefs: modelId / displayOrder
// ---------------------------------------------------------
export function mapProductBlueprintPatch(raw: any): ProductBlueprintPatchDTO {
  const p = raw ?? {};

  return {
    productName: p.productName,
    description: p.description,

    brandId: p.brandId,
    brandName: p.brandName,
    companyId: p.companyId,

    productBlueprintCategory: mapProductBlueprintCategory(
      p.productBlueprintCategory,
    ),
    categoryFields: p.categoryFields ?? null,

    fit: p.fit,
    material: p.material,
    weight: p.weight,
    qualityAssurance: p.qualityAssurance,

    productIdTag: mapProductIdTag(p.productIdTag),

    modelRefs: Array.isArray(p.modelRefs)
      ? p.modelRefs.map((r: any) => ({
          modelId: r.ModelID,
          displayOrder: r.DisplayOrder,
        }))
      : null,
  };
}

// ---------------------------------------------------------
// TokenBlueprintPatch mapper
//
// 期待 raw:
// {
//   tokenName,
//   symbol,
//   brandId,
//   brandName,
//   description,
//   iconUrl
// }
// ---------------------------------------------------------
export function mapTokenBlueprintPatch(
  raw: any,
): TokenBlueprintPatchDTO | undefined {
  if (raw === undefined || raw === null) return undefined;

  return {
    tokenName: raw.tokenName,
    symbol: raw.symbol,
    brandId: raw.brandId,
    brandName: raw.brandName,
    description: raw.description,
    iconUrl: raw.iconUrl,
  };
}

// ---------------------------------------------------------
// Product summary mapper
//
// 期待 row:
// {
//   productBlueprintId,
//   productName
// }
// ---------------------------------------------------------
export function mapPrintedInventorySummaries(
  data: any,
): InventoryProductSummary[] {
  if (!Array.isArray(data)) return [];

  const byPbId = new Map<string, InventoryProductSummary>();

  for (const row of data) {
    const id = row.productBlueprintId;
    if (!id) continue;

    if (!byPbId.has(id)) {
      byPbId.set(id, {
        id,
        productName: row.productName || "-",
      });
    }
  }

  return Array.from(byPbId.values());
}

// ---------------------------------------------------------
// Inventory IDs mapper
//
// NOTE:
// 後方互換削除後、Inventory Detail では使用しない。
// 他画面で未使用なら、この mapper も削除可能。
// ---------------------------------------------------------
export function mapInventoryIDsByProductAndToken(
  productBlueprintId: string,
  tokenBlueprintId: string,
  data: any,
): InventoryIDsByProductAndTokenDTO {
  if (!Array.isArray(data.inventoryIds)) {
    throw new Error("inventoryIds response must contain inventoryIds array");
  }

  return {
    productBlueprintId,
    tokenBlueprintId,
    inventoryIds: data.inventoryIds,
  };
}

// ---------------------------------------------------------
// InventoryDetail mapper
//
// GET /inventory/{inventoryId} の response を唯一の正とする。
// rows は backend 側で productBlueprintCategory.Kind に応じて完成済み。
//
// apparel row:
// {
//   modelId,
//   kind,
//   modelNumber,
//   stock,
//   size,
//   color,
//   rgb
// }
//
// alcohol row:
// {
//   modelId,
//   kind,
//   modelNumber,
//   stock,
//   volumeValue,
//   volumeUnit
// }
// ---------------------------------------------------------
export function mapInventoryDetailDTO(
  data: any,
  requestedId: string,
): InventoryDetailDTO {
  if (!data) {
    throw new Error("inventory detail response is empty");
  }

  if (!Array.isArray(data.rows)) {
    throw new Error("inventory detail rows must be an array");
  }

  const patch = mapProductBlueprintPatch(data.productBlueprintPatch);
  const tokenBlueprintPatch = mapTokenBlueprintPatch(data.tokenBlueprintPatch);

  const rows: InventoryDetailRowDTO[] = data.rows.map((r: any) => ({
    modelId: r.modelId,
    kind: r.kind ?? null,

    modelNumber: r.modelNumber,
    stock: r.stock,

    size: r.size ?? null,
    color: r.color ?? null,
    rgb: r.rgb ?? null,

    volumeValue: r.volumeValue ?? null,
    volumeUnit: r.volumeUnit ?? null,
  }));

  return {
    inventoryId: data.inventoryId ?? requestedId,

    tokenBlueprintId: data.tokenBlueprintId,
    productBlueprintId: data.productBlueprintId,

    productBlueprintPatch: patch,
    tokenBlueprintPatch,

    tokenBlueprint: data.tokenBlueprint
      ? {
          id: data.tokenBlueprint.id,
          name: data.tokenBlueprint.name,
          symbol: data.tokenBlueprint.symbol,
        }
      : undefined,

    productBlueprint: data.productBlueprint
      ? {
          id: data.productBlueprint.id,
          name: data.productBlueprint.name,
        }
      : undefined,

    rows,
    totalStock: data.totalStock,
    updatedAt: data.updatedAt,
  };
}