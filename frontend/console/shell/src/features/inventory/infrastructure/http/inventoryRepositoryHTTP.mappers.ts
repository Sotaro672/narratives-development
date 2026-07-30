// frontend/console/shell/src/features/inventory/infrastructure/http/inventoryRepositoryHTTP.mappers.ts

import type {
  InventoryDetailDTO,
  InventoryDetailRowDTO,
  ProductBlueprintPatchDTO,
  TokenBlueprintPatchDTO,
} from "./inventoryRepositoryHTTP.types";

import type {
  ProductBlueprintCategoryKind,
  ProductBlueprintCategorySnapshot,
} from "../../../productBlueprint/domain/productBlueprintCategory";

// =========================================================
// Inventory Detail mapper
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
  if (!raw) {
    return null;
  }

  return {
    id: raw.ID,
    code: raw.Code,
    nameJa: raw.NameJa,
    nameEn: raw.NameEn,
    kind: raw.Kind as ProductBlueprintCategoryKind,
    path: Array.isArray(raw.Path)
      ? raw.Path
      : [],
    parentId:
      raw.ParentID ?? null,
  };
}

function mapProductIdTag(
  raw: any,
): {
  type?: string;
} | null {
  if (!raw) {
    return null;
  }

  return {
    type: raw.Type,
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

export function mapProductBlueprintPatch(
  raw: any,
): ProductBlueprintPatchDTO {
  const patch =
    raw ?? {};

  return {
    productName:
      patch.productName,

    description:
      patch.description,

    brandId:
      patch.brandId,

    brandName:
      patch.brandName,

    companyId:
      patch.companyId,

    productBlueprintCategory:
      mapProductBlueprintCategory(
        patch.productBlueprintCategory,
      ),

    categoryFields:
      patch.categoryFields ?? null,

    fit:
      patch.fit,

    material:
      patch.material,

    weight:
      patch.weight,

    qualityAssurance:
      patch.qualityAssurance,

    productIdTag:
      mapProductIdTag(
        patch.productIdTag,
      ),

    modelRefs:
      Array.isArray(
        patch.modelRefs,
      )
        ? patch.modelRefs.map(
            (ref: any) => ({
              modelId:
                ref.ModelID,

              displayOrder:
                ref.DisplayOrder,
            }),
          )
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
  if (
    raw === undefined ||
    raw === null
  ) {
    return undefined;
  }

  return {
    tokenName:
      raw.tokenName,

    symbol:
      raw.symbol,

    brandId:
      raw.brandId,

    brandName:
      raw.brandName,

    description:
      raw.description,

    iconUrl:
      raw.iconUrl,
  };
}

// ---------------------------------------------------------
// Inventory Detail mapper
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
    throw new Error(
      "inventory detail response is empty",
    );
  }

  if (!Array.isArray(data.rows)) {
    throw new Error(
      "inventory detail rows must be an array",
    );
  }

  const productBlueprintPatch =
    mapProductBlueprintPatch(
      data.productBlueprintPatch,
    );

  const tokenBlueprintPatch =
    mapTokenBlueprintPatch(
      data.tokenBlueprintPatch,
    );

  const rows:
    InventoryDetailRowDTO[] =
    data.rows.map(
      (row: any) => ({
        modelId:
          row.modelId,

        kind:
          row.kind ?? null,

        modelNumber:
          row.modelNumber,

        stock:
          row.stock,

        size:
          row.size ?? null,

        color:
          row.color ?? null,

        rgb:
          row.rgb ?? null,

        volumeValue:
          row.volumeValue ?? null,

        volumeUnit:
          row.volumeUnit ?? null,
      }),
    );

  return {
    inventoryId:
      data.inventoryId ??
      requestedId,

    tokenBlueprintId:
      data.tokenBlueprintId,

    productBlueprintId:
      data.productBlueprintId,

    productBlueprintPatch,
    tokenBlueprintPatch,

    rows,

    totalStock:
      data.totalStock,

    updatedAt:
      data.updatedAt,
  };
}