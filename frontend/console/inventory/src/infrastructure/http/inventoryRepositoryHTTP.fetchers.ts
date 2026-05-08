// frontend/console/inventory/src/infrastructure/http/inventoryRepositoryHTTP.fetchers.ts

import {
  getInventoryListRaw,
  getTokenBlueprintPatchRaw,
  getInventoryDetailRaw,
} from "../api/inventoryApi";
import type {
  InventoryListRowDTO,
  TokenBlueprintPatchDTO,
  InventoryDetailDTO,
} from "./inventoryRepositoryHTTP.types";
import { s } from "./inventoryRepositoryHTTP.utils";

import {
  normalizeInventoryListRow,
  mapTokenBlueprintPatch,
  mapInventoryDetailDTO,
} from "./inventoryRepositoryHTTP.mappers";
/**
 * ✅ Inventory 一覧DTO
 * - 戻り値は "必ず tokenBlueprintId を含む" 正規化済み配列
 * - reservedCount / availableStock を落とさず返す
 */
export async function fetchInventoryListDTO(): Promise<InventoryListRowDTO[]> {
  const data = (await getInventoryListRaw()) as any;

  // ✅ 互換吸収を減らす：基本は配列を期待。どうしても違う場合のみ items を許容。
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
 * ✅ TokenBlueprint Patch DTO
 * GET /token-blueprints/{tokenBlueprintId}/patch
 */
export async function fetchTokenBlueprintPatchDTO(
  tokenBlueprintId: string,
): Promise<TokenBlueprintPatchDTO> {
  const tbId = s(tokenBlueprintId);
  if (!tbId) throw new Error("tokenBlueprintId is empty");

  const data = await getTokenBlueprintPatchRaw(tbId);
  return mapTokenBlueprintPatch(data) ?? {};
}

/**
 * ✅ Inventory Detail DTO
 * GET /inventory/{inventoryId}
 *
 * NOTE:
 * - productName / brandName 等は detail.productBlueprintPatch に含まれる前提
 * - brandId / assigneeId は不要のため取得しない
 */
export async function fetchInventoryDetailDTO(
  inventoryId: string,
): Promise<InventoryDetailDTO> {
  const id = s(inventoryId);
  if (!id) throw new Error("inventoryId is empty");

  const data = await getInventoryDetailRaw(id);
  return mapInventoryDetailDTO(data, id);
}
