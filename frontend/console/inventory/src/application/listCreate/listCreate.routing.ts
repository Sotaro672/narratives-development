// frontend/console/inventory/src/application/listCreate/listCreate.routing.ts

import type { ListCreateDTO } from "../../infrastructure/http/inventoryRepositoryHTTP";
import type { ListCreateRouteParams, ResolvedListCreateParams } from "./listCreate.types";
import { normalizeInventoryId, s } from "./listCreate.utils";

/**
 * ✅ 方針（無駄削除 + 互換維持）
 *
 * Firestore の inventory ドキュメントは常に
 * - productBlueprintId
 * - tokenBlueprintId
 * を持ち、ドキュメントIDは pb__tb 形式（inventoryKey）で運用されている前提。
 *
 * そのため、
 * - UI ルートは inventoryKey（pb__tb）を正とする
 * - backend fetch は常に pbId/tbId を使う（/inventory/list-create/:pbId/:tbId）
 *
 * ただし互換のために、
 * - pbId/tbId だけで遷移してきた場合は inventoryKey を合成
 * - inventoryId に pb__tb が来た場合は split して pb/tb を補完
 * を残す。
 */

const SEP = "__";

function splitInventoryKey(inventoryKey: string): { pbId: string; tbId: string } | null {
  const key = normalizeInventoryId(inventoryKey);
  if (!key || !key.includes(SEP)) return null;

  const parts = key.split(SEP);
  const pbId = s(parts[0]);
  const tbId = s(parts[1]);

  if (!pbId || !tbId) return null;
  return { pbId, tbId };
}

function makeInventoryKey(pbId: string, tbId: string): string {
  const pb = s(pbId);
  const tb = s(tbId);
  return pb && tb ? `${pb}${SEP}${tb}` : "";
}

/**
 * ✅ param 解決
 * - inventoryId（UIルートの :inventoryId）= inventoryKey（pb__tb）として扱う
 * - 互換: pb/tb だけで来た場合は inventoryKey を合成
 * - 互換: inventoryKey から pb/tb を split で補完
 */
export function resolveListCreateParams(raw: ListCreateRouteParams): ResolvedListCreateParams {
  const invKeyRaw = normalizeInventoryId(raw?.inventoryId);
  const pbRaw = s(raw?.productBlueprintId);
  const tbRaw = s(raw?.tokenBlueprintId);

  // 1) まず UI キー（inventoryKey）を確定（最優先は route param）
  const inventoryId = invKeyRaw || makeInventoryKey(pbRaw, tbRaw);

  // 2) pb/tb を確定（最優先は query/params の pb/tb。無ければ inventoryKey から split）
  let productBlueprintId = pbRaw;
  let tokenBlueprintId = tbRaw;

  if ((!productBlueprintId || !tokenBlueprintId) && inventoryId) {
    const split = splitInventoryKey(inventoryId);
    if (split) {
      if (!productBlueprintId) productBlueprintId = split.pbId;
      if (!tokenBlueprintId) tokenBlueprintId = split.tbId;
    }
  }

  return {
    inventoryId, // UI routing key (pb__tb)
    productBlueprintId,
    tokenBlueprintId,
    raw,
  };
}

export function computeListCreateTitle(inventoryId: string): string {
  return inventoryId ? `出品作成（inventoryId: ${inventoryId}）` : "出品作成";
}

export function canFetchListCreate(p: ResolvedListCreateParams): boolean {
  // ✅ backend fetch は pb/tb が揃っていることが前提
  // inventoryKey から補完できるので、基本的には true になるはず
  return Boolean(s(p.productBlueprintId) && s(p.tokenBlueprintId));
}

export function buildListCreateFetchInput(p: ResolvedListCreateParams): {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
} {
  // ✅ 無駄削除:
  // Firestore 実データ前提では inventory は必ず pb/tb を持つため、
  // backend list-create 取得は常に pb/tb ルートで叩く。
  const pb = s(p.productBlueprintId);
  const tb = s(p.tokenBlueprintId);

  if (pb && tb) {
    return {
      inventoryId: undefined,
      productBlueprintId: pb,
      tokenBlueprintId: tb,
    };
  }

  // 互換の最後の保険（通常ここには来ない）
  const split = splitInventoryKey(p.inventoryId);
  if (split) {
    return {
      inventoryId: undefined,
      productBlueprintId: split.pbId,
      tokenBlueprintId: split.tbId,
    };
  }

  // どうしても復元できない場合は空で返す（呼び出し側で missing params 扱い）
  return {
    inventoryId: undefined,
    productBlueprintId: undefined,
    tokenBlueprintId: undefined,
  };
}

export function getInventoryIdFromDTO(dto: ListCreateDTO | null | undefined): string {
  return normalizeInventoryId((dto as any)?.inventoryId ?? (dto as any)?.InventoryID);
}

/**
 * ✅ 互換削除:
 * 以前は「currentInventoryId が空なら gotInventoryId へリダイレクト」していたが、
 * 現在は UI ルートは inventoryKey（pb__tb）を正とするため不要。
 */
export function shouldRedirectToInventoryIdRoute(_: {
  currentInventoryId: string;
  gotInventoryId: string;
  alreadyRedirected: boolean;
}): boolean {
  return false;
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
  // ✅ UI ルートは pb__tb をそのまま URL に入れる（これが正）
  return `/inventory/list/create/${encodeURIComponent(id)}`;
}

export function buildBackPath(p: ResolvedListCreateParams): string {
  // pb/tb があれば詳細へ
  if (p.productBlueprintId && p.tokenBlueprintId) {
    return buildInventoryDetailPath(p.productBlueprintId, p.tokenBlueprintId);
  }

  // 互換: inventoryKey から復元できれば詳細へ
  const split = splitInventoryKey(p.inventoryId);
  if (split) return buildInventoryDetailPath(split.pbId, split.tbId);

  // 復元できない場合は一覧へ
  return "/inventory";
}

export function buildAfterCreatePath(p: ResolvedListCreateParams): string {
  // pb/tb があれば詳細へ
  if (p.productBlueprintId && p.tokenBlueprintId) {
    return buildInventoryDetailPath(p.productBlueprintId, p.tokenBlueprintId);
  }

  // 互換: inventoryKey から復元できれば詳細へ
  const split = splitInventoryKey(p.inventoryId);
  if (split) return buildInventoryDetailPath(split.pbId, split.tbId);

  return "/inventory";
}
