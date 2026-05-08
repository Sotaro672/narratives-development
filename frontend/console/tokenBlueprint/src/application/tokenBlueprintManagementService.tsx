// frontend/console/tokenBlueprint/src/application/tokenBlueprintManagementService.tsx

import type { TokenBlueprint } from "../../../shell/src/shared/types/tokenBlueprint";
import { listTokenBlueprintsByCompanyId } from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";

/** ISO8601 → timestamp（不正値は 0 扱い） */
const toTs = (iso: string): number => {
  if (!iso) return 0;
  const t = Date.parse(iso);
  return Number.isNaN(t) ? 0 : t;
};

export type SortKey = "createdAt" | null;
export type SortDir = "asc" | "desc" | null;

export type TokenBlueprintFilterState = {
  brandFilter: string[];
  assigneeFilter: string[];
  sortKey: SortKey;
  sortDir: SortDir;
};

/**
 * currentMember.companyId を指定してトークン設計一覧を取得
 * - backend 側では companyId は context から取得される想定
 *
 * ★変更点:
 * - minted による絞り込み（minted=false のみ表示）を廃止
 * - minted:true/false 両方を返す
 */
export async function fetchTokenBlueprintsForCompany(
  companyId: string,
): Promise<TokenBlueprint[]> {
  const cid = companyId.trim();
  if (!cid) return [];

  // backend は companyId を context で見ているので、
  // ここでは呼び出しのトリガとして companyId を受け取るだけでOK
  const all = await listTokenBlueprintsByCompanyId(cid);

  // ★ minted でのフィルタはしない（true/false 両方表示）
  return all;
}

/**
 * 一覧から brand / assignee のフィルタオプションを生成
 */
export function buildOptionsFromTokenBlueprints(rows: TokenBlueprint[]): {
  brandOptions: { value: string; label: string }[];
  assigneeOptions: { value: string; label: string }[];
} {
  const brandSet = new Set<string>();
  const assigneeSet = new Set<string>();

  for (const r of rows) {
    if (r.brandId) brandSet.add(r.brandId);
    if (r.assigneeId) assigneeSet.add(r.assigneeId);
  }

  const brandOptions = Array.from(brandSet).map((v) => ({
    value: v,
    label: v,
  }));

  const assigneeOptions = Array.from(assigneeSet).map((v) => ({
    value: v,
    label: v,
  }));

  return { brandOptions, assigneeOptions };
}

/**
 * フィルタ＋ソートを適用した TokenBlueprint 一覧を返す
 *
 * ★変更点:
 * - minted による篩い分けは廃止
 * - brand / assignee のフィルタは従来通り有効
 */
export function filterAndSortTokenBlueprints(
  rows: TokenBlueprint[],
  state: TokenBlueprintFilterState,
): TokenBlueprint[] {
  const { brandFilter, assigneeFilter, sortKey, sortDir } = state;

  // ★ minted の絞り込みはしない
  let data = (rows ?? []).filter(
    (r) =>
      (brandFilter.length === 0 || brandFilter.includes(r.brandId)) &&
      (assigneeFilter.length === 0 || assigneeFilter.includes(r.assigneeId)),
  );

  if (sortKey && sortDir) {
    data = [...data].sort((a, b) => {
      const av = toTs((a as any)[sortKey] as string);
      const bv = toTs((b as any)[sortKey] as string);
      return sortDir === "asc" ? av - bv : bv - av;
    });
  }

  return data;
}
