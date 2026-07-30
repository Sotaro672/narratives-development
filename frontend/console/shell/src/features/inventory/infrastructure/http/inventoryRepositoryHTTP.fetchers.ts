// frontend/console/shell/src/features/inventory/infrastructure/http/inventoryRepositoryHTTP.fetchers.ts

import {
  getInventoryDetailRaw,
  getInventoryListRaw,
} from "../api/inventoryApi";

import type {
  InventoryDetailDTO,
  InventoryListRowDTO,
} from "./inventoryRepositoryHTTP.types";

import {
  mapInventoryDetailDTO,
} from "./inventoryRepositoryHTTP.mappers";

/**
 * Inventory 一覧DTO
 *
 * GET /inventory
 *
 * 前提:
 * - backend response は InventoryListRowDTO[]。
 * - 旧 items wrapper などの後方互換は扱わない。
 * - backend response のフィールド名と型を正とする。
 * - frontend では一覧行の normalize を行わない。
 */
export async function fetchInventoryListDTO(): Promise<
  InventoryListRowDTO[]
> {
  const data =
    await getInventoryListRaw();

  if (!Array.isArray(data)) {
    throw new Error(
      "inventory list response must be an array",
    );
  }

  return data as InventoryListRowDTO[];
}

/**
 * Inventory Detail DTO
 *
 * GET /inventory/{inventoryId}
 *
 * 前提:
 * - Inventory Detail 画面はこの API だけを正とする。
 * - /models/by-blueprint/{productBlueprintId}/variations は呼ばない。
 * - productBlueprintPatch / tokenBlueprintPatch / rows は detail response に含まれる。
 * - rows は backend 側で productBlueprintCategory.Kind に応じた完成形になっている。
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