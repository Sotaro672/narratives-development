// frontend/console/inventory/src/application/inventoryDetail/inventoryDetail.usecase.ts

import {
  getInventoryDetailRaw,
  getTokenBlueprintPatchRaw,
} from "../../infrastructure/api/inventoryApi";
import type {
  InventoryDetailDTO,
  TokenBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP.types";
import type { InventoryDetailViewModel } from "./inventoryDetail.types";
import {
  buildInventoryDetailViewModel,
  mapTokenBlueprintPatch,
} from "./inventoryDetail.mapper";

export async function loadInventoryDetailViewModel(
  inventoryId: string,
): Promise<InventoryDetailViewModel> {
  const detailRaw = (await getInventoryDetailRaw(inventoryId)) as any;
  const detail = detailRaw as InventoryDetailDTO;

  const tokenBlueprintId = (detail as any)?.tokenBlueprintId;

  if (!tokenBlueprintId) {
    throw new Error("inventory_detail_missing_product_or_token_blueprint_id");
  }

  let tokenBlueprintPatch: TokenBlueprintPatchDTO | undefined = undefined;

  try {
    const patchRaw = await getTokenBlueprintPatchRaw(tokenBlueprintId);
    tokenBlueprintPatch = mapTokenBlueprintPatch(patchRaw);
  } catch {
    tokenBlueprintPatch = undefined;
  }

  return buildInventoryDetailViewModel({
    inventoryId,
    detail,
    tokenBlueprintPatch,
  });
}