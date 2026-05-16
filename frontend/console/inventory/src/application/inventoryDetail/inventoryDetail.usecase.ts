// frontend/console/inventory/src/application/inventoryDetail/inventoryDetail.usecase.ts

import { getInventoryDetailRaw } from "../../infrastructure/api/inventoryApi";
import type { TokenBlueprintPatchDTO } from "../../infrastructure/http/inventoryRepositoryHTTP.types";
import {
  mapInventoryDetailDTO,
  mapTokenBlueprintPatch,
} from "../../infrastructure/http/inventoryRepositoryHTTP.mappers";
import { listModelVariationsByProductBlueprintId } from "../../../../model/src/infrastructure/repository/modelRepositoryHTTP";
import type { ModelVariationResponse } from "../../../../model/src/infrastructure/repository/modelRepositoryHTTP";

import type { InventoryDetailViewModel } from "./inventoryDetail.types";
import { buildInventoryDetailViewModel } from "./inventoryDetail.mapper";

export async function loadInventoryDetailViewModel(
  inventoryId: string,
): Promise<InventoryDetailViewModel> {
  const detailRaw = await getInventoryDetailRaw(inventoryId);
  const detail = mapInventoryDetailDTO(detailRaw, inventoryId);

  const productBlueprintId = detail.productBlueprintId;
  const tokenBlueprintId = detail.tokenBlueprintId;

  if (!productBlueprintId || !tokenBlueprintId) {
    throw new Error("inventory_detail_missing_product_or_token_blueprint_id");
  }

  let tokenBlueprintPatch: TokenBlueprintPatchDTO | undefined = undefined;
  let modelVariations: ModelVariationResponse[] = [];

  try {
    if (detailRaw?.tokenBlueprintPatch) {
      tokenBlueprintPatch = mapTokenBlueprintPatch(detailRaw.tokenBlueprintPatch);
    }
  } catch {
    tokenBlueprintPatch = undefined;
  }

  try {
    modelVariations =
      await listModelVariationsByProductBlueprintId(productBlueprintId);
  } catch {
    modelVariations = [];
  }

  return buildInventoryDetailViewModel({
    inventoryId,
    detail,
    tokenBlueprintPatch,
    modelVariations,
  });
}