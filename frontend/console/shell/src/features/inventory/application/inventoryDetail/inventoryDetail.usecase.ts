// frontend/console/shell/src/features/inventory/application/inventoryDetail/inventoryDetail.usecase.ts

import {
  getInventoryDetailRaw,
} from "../../infrastructure/api/inventoryApi";

import type {
  InventoryDetailViewModel,
} from "./inventoryDetail.types";

import {
  buildInventoryDetailViewModel,
} from "./inventoryDetail.mapper";

export async function loadInventoryDetailViewModel(
  inventoryId: string,
): Promise<InventoryDetailViewModel> {
  const id =
    String(
      inventoryId ?? "",
    ).trim();

  if (!id) {
    throw new Error(
      "inventoryId is empty",
    );
  }

  const detail =
    await getInventoryDetailRaw(
      id,
    );

  const detailInventoryId =
    detail.inventoryId;

  const productBlueprintId =
    detail.productBlueprintId;

  const tokenBlueprintId =
    detail.tokenBlueprintId;

  if (!detailInventoryId) {
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

  return buildInventoryDetailViewModel({
    inventoryId:
      detailInventoryId,

    detail,
  });
}