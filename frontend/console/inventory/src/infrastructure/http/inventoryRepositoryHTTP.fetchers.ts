// frontend/console/inventory/src/infrastructure/http/inventoryRepositoryHTTP.fetchers.ts

import {
  getInventoryListRaw,
  getProductBlueprintRaw,
  getPrintedProductBlueprintsRaw,
  getInventoryIDsByProductAndTokenRaw,
  getTokenBlueprintPatchRaw,
  getListCreateRaw,
  getInventoryDetailRaw,
} from "../api/inventoryApi";

import type {
  InventoryListRowDTO,
  InventoryProductSummary,
  InventoryIDsByProductAndTokenDTO,
  TokenBlueprintPatchDTO,
  ListCreateDTO,
  InventoryDetailDTO,
} from "./inventoryRepositoryHTTP.types";

import { s } from "./inventoryRepositoryHTTP.utils";

import {
  normalizeInventoryListRow,
  mapInventoryProductSummary,
  mapPrintedInventorySummaries,
  mapInventoryIDsByProductAndToken,
  mapTokenBlueprintPatch,
  mapListCreateDTO,
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
 * 在庫詳細画面用：
 * ProductBlueprint ID から productName / brandId / assigneeId を取得
 *
 * GET /product-blueprints/{id}
 */
export async function fetchInventoryProductSummary(
  productBlueprintId: string,
): Promise<InventoryProductSummary> {
  const pbId = s(productBlueprintId);
  if (!pbId) throw new Error("productBlueprintId is empty");

  const data = await getProductBlueprintRaw(pbId);
  return mapInventoryProductSummary(data);
}

/**
 * 在庫一覧（ヘッダー用）:
 * printed == "printed" の ProductBlueprint 一覧を取得
 *
 * GET /product-blueprints/printed
 */
export async function fetchPrintedInventorySummaries(): Promise<InventoryProductSummary[]> {
  const data = await getPrintedProductBlueprintsRaw();
  return mapPrintedInventorySummaries(data);
}

/**
 * ✅ inventoryIds 解決 DTO（方針A）
 * GET /inventory/ids?productBlueprintId=...&tokenBlueprintId=...
 */
export async function fetchInventoryIDsByProductAndTokenDTO(
  productBlueprintId: string,
  tokenBlueprintId: string,
): Promise<InventoryIDsByProductAndTokenDTO> {
  const pbId = s(productBlueprintId);
  const tbId = s(tokenBlueprintId);
  if (!pbId) throw new Error("productBlueprintId is empty");
  if (!tbId) throw new Error("tokenBlueprintId is empty");

  const data = await getInventoryIDsByProductAndTokenRaw({
    productBlueprintId: pbId,
    tokenBlueprintId: tbId,
  });

  return mapInventoryIDsByProductAndToken(pbId, tbId, data);
}

/**
 * ✅ NEW: TokenBlueprint Patch DTO
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
 * ✅ ListCreate DTO 取得
 * GET
 * - /inventory/list-create/:inventoryId
 * - /inventory/list-create/:productBlueprintId/:tokenBlueprintId
 */
export async function fetchListCreateDTO(input: {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
}): Promise<ListCreateDTO> {
  const data = await getListCreateRaw(input);
  return mapListCreateDTO(data);
}

/**
 * ✅ Inventory Detail DTO
 * GET /inventory/{inventoryId}
 */
export async function fetchInventoryDetailDTO(
  inventoryId: string,
): Promise<InventoryDetailDTO> {
  const id = s(inventoryId);
  if (!id) throw new Error("inventoryId is empty");

  const data = await getInventoryDetailRaw(id);
  return mapInventoryDetailDTO(data, id);
}
