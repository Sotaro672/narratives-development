// frontend/amol/src/features/catalog/infrastructure/catalogResponseMapper.ts

import type { CatalogResponse } from "../types";

export function mapCatalogResponse(raw: Partial<CatalogResponse>): CatalogResponse {
  if (!raw.list || typeof raw.list.title !== "string") {
    throw new Error("カタログ詳細APIのlistが不正です。");
  }

  if (!raw.productBlueprint || typeof raw.productBlueprint.id !== "string") {
    throw new Error("カタログ詳細APIのproductBlueprintが不正です。");
  }

  return {
    ...raw,
    list: {
      ...raw.list,
      prices: Array.isArray(raw.list.prices) ? raw.list.prices : [],
      description: raw.list.description ?? "",
      image: raw.list.image ?? "",
      inventoryId: raw.list.inventoryId ?? "",
    },
    listImages: Array.isArray(raw.listImages) ? raw.listImages : [],
    inventory: {
      ...raw.inventory,
      id: raw.inventory?.id ?? raw.list.inventoryId ?? "",
      productBlueprintId: raw.inventory?.productBlueprintId ?? "",
      tokenBlueprintId: raw.inventory?.tokenBlueprintId ?? "",
      modelIds: Array.isArray(raw.inventory?.modelIds)
        ? raw.inventory.modelIds
        : [],
      stock: raw.inventory?.stock ?? {},
    },
    productBlueprint: {
      ...raw.productBlueprint,
      productName: raw.productBlueprint.productName ?? "",
      brandId: raw.productBlueprint.brandId ?? "",
      companyId: raw.productBlueprint.companyId ?? "",
      brandName: raw.productBlueprint.brandName ?? "",
      companyName: raw.productBlueprint.companyName ?? "",
      printed: Boolean(raw.productBlueprint.printed),
      productIdTagType: raw.productBlueprint.productIdTagType ?? "",

      productBlueprintCategoryId:
        raw.productBlueprint.productBlueprintCategoryId ?? null,
      productBlueprintCategoryCode:
        raw.productBlueprint.productBlueprintCategoryCode ?? null,
      productBlueprintCategoryKind:
        raw.productBlueprint.productBlueprintCategoryKind ?? null,
      productBlueprintCategoryNameEn:
        raw.productBlueprint.productBlueprintCategoryNameEn ?? null,
      productBlueprintCategoryNameJa:
        raw.productBlueprint.productBlueprintCategoryNameJa ?? null,
      productBlueprintCategoryPath: Array.isArray(
        raw.productBlueprint.productBlueprintCategoryPath,
      )
        ? raw.productBlueprint.productBlueprintCategoryPath
        : null,

      categoryFields: raw.productBlueprint.categoryFields ?? null,

      modelRefs: Array.isArray(raw.productBlueprint.modelRefs)
        ? raw.productBlueprint.modelRefs
        : [],
    },
    tokenBlueprint: {
      ...raw.tokenBlueprint,
      id: raw.tokenBlueprint?.id ?? "",
      tokenName: raw.tokenBlueprint?.tokenName ?? "",
      symbol: raw.tokenBlueprint?.symbol ?? "",
      brandId: raw.tokenBlueprint?.brandId ?? "",
      brandName: raw.tokenBlueprint?.brandName ?? "",
      companyName: raw.tokenBlueprint?.companyName ?? "",
      description: raw.tokenBlueprint?.description ?? "",
      tokenIcon: raw.tokenBlueprint?.tokenIcon ?? "",
    },
    modelVariations: Array.isArray(raw.modelVariations)
      ? raw.modelVariations
      : [],
    productReviewSummary: raw.productReviewSummary ?? {
      productBlueprintId: raw.productBlueprint.id,
      status: "PUBLISHED",
      totalCount: 0,
      averageRating: 0,
      rating5Count: 0,
      rating4Count: 0,
      rating3Count: 0,
      rating2Count: 0,
      rating1Count: 0,
    },
  } as CatalogResponse;
}