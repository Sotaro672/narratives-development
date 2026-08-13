// frontend/console/shell/src/features/inventory/application/inventoryDetail/inventoryDetailService.tsx

import {
  getInventoryDetailRaw,
} from "../infrastructure/inventoryApi";

import type {
  InventoryDetailDTO,
  InventoryDetailViewModel,
} from "../../../shared/types/inventory";

function buildInventoryDetailViewModel(
  detail: InventoryDetailDTO,
): InventoryDetailViewModel {
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

export async function loadInventoryDetailViewModel(
  inventoryId: string,
): Promise<InventoryDetailViewModel> {
  const id =
    inventoryId.trim();

  if (!id) {
    throw new Error(
      "inventoryId is empty",
    );
  }

  const detail =
    await getInventoryDetailRaw(
      id,
    );

  if (!detail.inventoryId) {
    throw new Error(
      "inventory_detail_missing_inventory_id",
    );
  }

  if (
    !detail.productBlueprintId ||
    !detail.tokenBlueprintId
  ) {
    throw new Error(
      "inventory_detail_missing_product_or_token_blueprint_id",
    );
  }

  return buildInventoryDetailViewModel(
    detail,
  );
}