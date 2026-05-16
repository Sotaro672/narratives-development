// frontend/console/inventory/src/infrastructure/http/inventoryRepositoryHTTP.fetchers.ts

import { getInventoryListRaw, getInventoryDetailRaw } from "../api/inventoryApi";
import type {
  InventoryListRowDTO,
  TokenBlueprintPatchDTO,
  InventoryDetailDTO,
} from "./inventoryRepositoryHTTP.types";

import {
  normalizeInventoryListRow,
  mapTokenBlueprintPatch,
  mapInventoryDetailDTO,
} from "./inventoryRepositoryHTTP.mappers";

/**
 * Inventory 一覧DTO
 * - 戻り値は "必ず tokenBlueprintId を含む" 正規化済み配列
 * - reservedCount / availableStock を落とさず返す
 */
export async function fetchInventoryListDTO(): Promise<InventoryListRowDTO[]> {
  const data = (await getInventoryListRaw()) as any;

  // 互換吸収を減らす：基本は配列を期待。どうしても違う場合のみ items を許容。
  const rawItems: any[] = Array.isArray(data)
    ? data
    : Array.isArray(data?.items)
      ? data.items
      : [];

  return rawItems
    .map(normalizeInventoryListRow)
    .filter((x): x is InventoryListRowDTO => x !== null);
}

/**
 * TokenBlueprint Patch DTO
 *
 * NOTE:
 * - Inventory Detail では GET /inventory/{inventoryId} の tokenBlueprintPatch を正とする
 * - ここでは追加で GET /token-blueprints/{tokenBlueprintId}/patch を呼ばない
 * - 後方互換用に raw detail から tokenBlueprintPatch を取り出す fetcher として扱う
 */
export function fetchTokenBlueprintPatchDTOFromInventoryDetailRaw(
  detailRaw: any,
): TokenBlueprintPatchDTO {
  return mapTokenBlueprintPatch(detailRaw?.tokenBlueprintPatch) ?? {};
}

/**
 * Inventory Detail に含まれる TokenBlueprint Patch DTO
 * GET /inventory/{inventoryId}
 *
 * NOTE:
 * - GET /token-blueprints/{tokenBlueprintId}/patch は呼ばない
 * - inventoryId は `${productBlueprintId}__${tokenBlueprintId}` 形式を想定
 */
export async function fetchTokenBlueprintPatchDTOByInventoryId(
  inventoryId: string,
): Promise<TokenBlueprintPatchDTO> {
  const id = String(inventoryId ?? "").trim();
  if (!id) {
    throw new Error("inventoryId is empty");
  }

  const data = await getInventoryDetailRaw(id);
  return mapTokenBlueprintPatch(data?.tokenBlueprintPatch) ?? {};
}

/**
 * Inventory Detail DTO
 * GET /inventory/{inventoryId}
 *
 * NOTE:
 * - productName / brandName 等は detail.productBlueprintPatch に含まれる前提
 * - tokenBlueprintPatch も detail.tokenBlueprintPatch に含まれる前提
 * - brandId / assigneeId は不要のため追加取得しない
 */
export async function fetchInventoryDetailDTO(
  inventoryId: string,
): Promise<InventoryDetailDTO> {
  const id = String(inventoryId ?? "").trim();
  if (!id) {
    throw new Error("inventoryId is empty");
  }

  const data = await getInventoryDetailRaw(id);
  return mapInventoryDetailDTO(data, id);
}