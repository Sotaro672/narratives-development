// frontend/console/inventory/src/application/listCreate/listCreate.routing.ts

import type { ListCreateDTO } from "../../infrastructure/http/inventoryRepositoryHTTP";
import type { ListCreateRouteParams, ResolvedListCreateParams } from "./listCreate.types";
import { normalizeInventoryId, s } from "./listCreate.utils";

/**
 * ✅ param 解決
 * - inventoryId が来ていればそれを優先（pb__tbを維持）
 * - inventoryId が無ければ pbId + tbId から pb__tb を合成
 * - inventoryId しか無い場合は pb/tb を補完（抽出のために split はするが inventoryId は変えない）
 */
export function resolveListCreateParams(raw: ListCreateRouteParams): ResolvedListCreateParams {
  const inv = normalizeInventoryId(raw?.inventoryId);
  const pbRaw = s(raw?.productBlueprintId);
  const tbRaw = s(raw?.tokenBlueprintId);

  // inventoryId が無いなら pb/tb から合成
  const inventoryId = inv || (pbRaw && tbRaw ? `${pbRaw}__${tbRaw}` : "");

  // pb/tb が無いなら inventoryId から補完（※抽出のための split）
  let productBlueprintId = pbRaw;
  let tokenBlueprintId = tbRaw;
  if ((!productBlueprintId || !tokenBlueprintId) && inventoryId.includes("__")) {
    const parts = inventoryId.split("__");
    const pb = s(parts[0]);
    const tb = s(parts[1]);
    if (!productBlueprintId) productBlueprintId = pb;
    if (!tokenBlueprintId) tokenBlueprintId = tb;
  }

  return {
    inventoryId,
    productBlueprintId,
    tokenBlueprintId,
    raw,
  };
}

export function computeListCreateTitle(inventoryId: string): string {
  return inventoryId ? `出品作成（inventoryId: ${inventoryId}）` : "出品作成";
}

export function canFetchListCreate(p: ResolvedListCreateParams): boolean {
  // ✅ 方針A: inventoryId（pb__tb）があれば取得できる
  return Boolean(p.inventoryId);
}

export function buildListCreateFetchInput(p: ResolvedListCreateParams): {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
} {
  // ✅ 方針A: backend は inventoryId（pb__tb）を期待
  return {
    inventoryId: p.inventoryId || undefined,
    productBlueprintId: undefined,
    tokenBlueprintId: undefined,
  };
}

export function getInventoryIdFromDTO(dto: ListCreateDTO | null | undefined): string {
  return normalizeInventoryId((dto as any)?.inventoryId ?? (dto as any)?.InventoryID);
}

export function shouldRedirectToInventoryIdRoute(args: {
  currentInventoryId: string;
  gotInventoryId: string;
  alreadyRedirected: boolean;
}): boolean {
  return !args.alreadyRedirected && !args.currentInventoryId && Boolean(args.gotInventoryId);
}

export function buildInventoryDetailPath(pbId: string, tbId: string): string {
  const pb = s(pbId);
  const tb = s(tbId);
  if (!pb || !tb) return "/inventory";
  return `/inventory/detail/${encodeURIComponent(pb)}/${encodeURIComponent(tb)}`;
}

export function buildInventoryListCreatePath(inventoryId: string): string {
  const id = normalizeInventoryId(inventoryId);
  if (!id) return "/inventory/list/create";
  // ✅ pb__tb をそのまま URL に入れる
  return `/inventory/list/create/${encodeURIComponent(id)}`;
}

export function buildBackPath(p: ResolvedListCreateParams): string {
  if (p.productBlueprintId && p.tokenBlueprintId) {
    return buildInventoryDetailPath(p.productBlueprintId, p.tokenBlueprintId);
  }
  // pb/tb が補完できない場合は一覧へ
  return "/inventory";
}

export function buildAfterCreatePath(p: ResolvedListCreateParams): string {
  if (p.productBlueprintId && p.tokenBlueprintId) {
    return buildInventoryDetailPath(p.productBlueprintId, p.tokenBlueprintId);
  }
  return "/inventory";
}
