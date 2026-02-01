// frontend/console/inventory/src/application/listCreate/listCreate.routing.ts

import type { ListCreateDTO } from "../../infrastructure/http/inventoryRepositoryHTTP";
import type { ListCreateRouteParams, ResolvedListCreateParams } from "./listCreate.types";
import { normalizeInventoryId, s } from "./listCreate.utils";

/**
 *
 * - UI ルートは inventoryId（= inventoryKey: "pb__tb"）のみを正とする
 * - backend fetch も inventoryId のみを使う（/inventory/list-create/:inventoryId）
 * - productBlueprintId / tokenBlueprintId は一切扱わない（互換も廃止）
 */

export function resolveListCreateParams(raw: ListCreateRouteParams): ResolvedListCreateParams {
  const inventoryId = normalizeInventoryId(raw?.inventoryId);

  return {
    inventoryId: inventoryId || "",
    // productBlueprintId: "",
    // tokenBlueprintId: "",
    raw,
  } as ResolvedListCreateParams;
}

export function computeListCreateTitle(inventoryId: string): string {
  return inventoryId ? `出品作成（inventoryId: ${inventoryId}）` : "出品作成";
}

export function canFetchListCreate(p: ResolvedListCreateParams): boolean {
  // ✅ inventoryId があれば fetch 可
  return Boolean(s((p as any)?.inventoryId));
}

export function buildListCreateFetchInput(p: ResolvedListCreateParams): {
  inventoryId?: string;
} {
  const inventoryId = s((p as any)?.inventoryId);
  if (!inventoryId) {
    return { inventoryId: undefined };
  }
  return { inventoryId };
}

export function getInventoryIdFromDTO(dto: ListCreateDTO | null | undefined): string {
  return normalizeInventoryId((dto as any)?.inventoryId ?? (dto as any)?.InventoryID);
}

/**
 * ✅ リダイレクトは不要
 */
export function shouldRedirectToInventoryIdRoute(_: {
  currentInventoryId: string;
  gotInventoryId: string;
  alreadyRedirected: boolean;
}): boolean {
  return false;
}

export function buildInventoryDetailPath(inventoryId: string): string {
  const id = normalizeInventoryId(inventoryId);
  if (!id) return "/inventory";
  // 既存が /inventory/detail/:pb/:tb のままならここは変更が必要
  return `/inventory/detail/${encodeURIComponent(id)}`;
}

export function buildInventoryListCreatePath(inventoryId: string): string {
  const id = normalizeInventoryId(inventoryId);
  if (!id) return "/inventory/list/create";
  return `/inventory/list/create/${encodeURIComponent(id)}`;
}

export function buildBackPath(p: ResolvedListCreateParams): string {
  const inventoryId = s((p as any)?.inventoryId);
  if (inventoryId) return buildInventoryDetailPath(inventoryId);
  return "/inventory";
}

export function buildAfterCreatePath(p: ResolvedListCreateParams): string {
  const inventoryId = s((p as any)?.inventoryId);
  if (inventoryId) return buildInventoryDetailPath(inventoryId);
  return "/inventory";
}
