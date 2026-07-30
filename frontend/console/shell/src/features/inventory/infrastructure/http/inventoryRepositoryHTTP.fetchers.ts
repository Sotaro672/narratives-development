// frontend/console/shell/src/features/inventory/infrastructure/http/inventoryRepositoryHTTP.fetchers.ts

import {
  getInventoryDetailRaw,
} from "../api/inventoryApi";

import type {
  InventoryDetailDTO,
} from "./inventoryRepositoryHTTP.types";

import {
  mapInventoryDetailDTO,
} from "./inventoryRepositoryHTTP.mappers";

/**
 * Inventory Detail DTO
 *
 * GET /inventory/{inventoryId}
 *
 * 前提:
 * - Inventory Detail 画面はこのAPIだけを正とする。
 * - /models/by-blueprint/{productBlueprintId}/variationsは呼ばない。
 * - productBlueprintPatch / tokenBlueprintPatch / rowsはdetail responseに含まれる。
 * - rowsはbackend側でproductBlueprintCategory.Kindに応じた完成形になっている。
 */
export async function fetchInventoryDetailDTO(
  inventoryId: string,
): Promise<InventoryDetailDTO> {
  const id =
    String(
      inventoryId ?? "",
    ).trim();

  if (!id) {
    throw new Error(
      "inventoryId is empty",
    );
  }

  const data =
    await getInventoryDetailRaw(
      id,
    );

  return mapInventoryDetailDTO(
    data,
    id,
  );
}