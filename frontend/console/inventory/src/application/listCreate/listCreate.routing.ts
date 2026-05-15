// frontend/console/inventory/src/application/listCreate/listCreate.routing.ts

import type { ListCreateDTO } from "../../infrastructure/http/inventoryRepositoryHTTP";
import type {
  ListCreateRouteParams,
  ResolvedListCreateParams,
} from "./listCreate.types";

/**
 * - UI ルートは inventoryId（= inventoryKey: "pb__tb"）のみを正とする
 * - backend fetch も inventoryId のみを使う（/inventory/list-create/:inventoryId）
 * - productBlueprintId / tokenBlueprintId は一切扱わない（互換も廃止）
 */

export function resolveListCreateParams(
  raw: ListCreateRouteParams,
): ResolvedListCreateParams {
  return {
    inventoryId: raw.inventoryId,
    raw,
  } as ResolvedListCreateParams;
}

export function canFetchListCreate(p: ResolvedListCreateParams): boolean {
  return Boolean(p.inventoryId);
}

export function buildListCreateFetchInput(p: ResolvedListCreateParams): {
  inventoryId?: string;
} {
  if (!p.inventoryId) {
    return { inventoryId: undefined };
  }

  return {
    inventoryId: p.inventoryId,
  };
}

export function getInventoryIdFromDTO(
  dto: ListCreateDTO | null | undefined,
): string {
  return dto?.inventoryId ?? "";
}

/**
 * リダイレクトは不要
 */
export function shouldRedirectToInventoryIdRoute(_: {
  currentInventoryId: string;
  gotInventoryId: string;
  alreadyRedirected: boolean;
}): boolean {
  return false;
}

export function buildInventoryDetailPath(inventoryId: string): string {
  if (!inventoryId) return "/inventory";
  return `/inventory/detail/${encodeURIComponent(inventoryId)}`;
}

export function buildInventoryListCreatePath(inventoryId: string): string {
  if (!inventoryId) return "/inventory/list/create";
  return `/inventory/list/create/${encodeURIComponent(inventoryId)}`;
}

export function buildBackPath(p: ResolvedListCreateParams): string {
  if (p.inventoryId) return buildInventoryDetailPath(p.inventoryId);
  return "/inventory";
}

export function buildAfterCreatePath(p: ResolvedListCreateParams): string {
  if (p.inventoryId) return buildInventoryDetailPath(p.inventoryId);
  return "/inventory";
}