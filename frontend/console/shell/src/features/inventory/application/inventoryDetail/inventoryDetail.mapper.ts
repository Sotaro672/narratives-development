// frontend/console/shell/src/features/inventory/application/inventoryDetail/inventoryDetail.mapper.ts

import type {
  InventoryDetailDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP.types";

import type {
  InventoryDetailViewModel,
} from "./inventoryDetail.types";

export function buildInventoryDetailViewModel(args: {
  inventoryId: string;
  detail: InventoryDetailDTO;
}): InventoryDetailViewModel {
  const {
    detail,
  } = args;

  const inventoryId =
    detail.inventoryId;

  const productBlueprintId =
    detail.productBlueprintId;

  const tokenBlueprintId =
    detail.tokenBlueprintId;

  const productBlueprintPatch =
    detail.productBlueprintPatch;

  const tokenBlueprintPatch =
    detail.tokenBlueprintPatch;

  if (!inventoryId) {
    throw new Error(
      "inventory_detail_missing_inventory_id",
    );
  }

  if (
    !productBlueprintId ||
    !tokenBlueprintId
  ) {
    throw new Error(
      "inventory_detail_missing_product_or_token_blueprint_id",
    );
  }

  if (!tokenBlueprintPatch) {
    throw new Error(
      "inventory_detail_missing_token_blueprint_patch",
    );
  }

  const productName =
    productBlueprintPatch.productName;

  const tokenName =
    tokenBlueprintPatch.tokenName;

  if (!productName) {
    throw new Error(
      "inventory_detail_missing_product_name",
    );
  }

  if (!tokenName) {
    throw new Error(
      "inventory_detail_missing_token_name",
    );
  }

  const category =
    productBlueprintPatch.productBlueprintCategory;

  if (!category) {
    throw new Error(
      "inventory_detail_missing_product_blueprint_category",
    );
  }

  return {
    inventoryId,

    productBlueprintId,
    tokenBlueprintId,

    productName,
    tokenName,

    headerTitle:
      `${productName} / ${tokenName}`,

    productBlueprintCategoryName:
      category.nameJa,

    productBlueprintCategoryCode:
      category.code,

    productBlueprintCategoryKind:
      category.kind,

    categoryFields:
      productBlueprintPatch.categoryFields,

    productBlueprintPatch,
    tokenBlueprintPatch,

    updatedAt:
      detail.updatedAt,

    totalStock:
      detail.totalStock,

    rows:
      detail.rows,
  };
}