// frontend/console/inventory/src/application/inventoryDetail/inventoryDetail.usecase.ts

import { getInventoryDetailRaw } from "../../infrastructure/api/inventoryApi";
import { mapInventoryDetailDTO } from "../../infrastructure/http/inventoryRepositoryHTTP.mappers";

import type { InventoryDetailViewModel } from "./inventoryDetail.types";
import { buildInventoryDetailViewModel } from "./inventoryDetail.mapper";

export async function loadInventoryDetailViewModel(
  inventoryId: string,
): Promise<InventoryDetailViewModel> {
  const id = String(inventoryId ?? "").trim();
  if (!id) {
    throw new Error("inventoryId is empty");
  }

  const detailRaw = await getInventoryDetailRaw(id);
  const detail = mapInventoryDetailDTO(detailRaw, id);

  const productBlueprintId = detail.productBlueprintId;
  const tokenBlueprintId = detail.tokenBlueprintId;

  if (!productBlueprintId || !tokenBlueprintId) {
    throw new Error("inventory_detail_missing_product_or_token_blueprint_id");
  }

  return buildInventoryDetailViewModel({
    inventoryId: id,
    detail,
  });
}