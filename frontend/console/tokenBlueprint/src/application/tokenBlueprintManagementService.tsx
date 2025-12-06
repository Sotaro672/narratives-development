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
 * minted = "notYet" の行だけを抽出するユーティリティ
 * - backend の ListMintedNotYet と同等のフィルタをフロント側でも行う
 */
export function listNotYet(rows: TokenBlueprint[]): TokenBlueprint[] {
  return rows.filter((r) => r.minted === "notYet");
}

/**
 * currentMember.companyId を指定してトークン設計一覧を取得
 * - backend 側では ListByCompanyID usecase → GetNameByID が裏側で利用される想定
 * - ここで minted = "notYet" のみを画面に渡す
 */
export async function fetchTokenBlueprintsForCompany(
  companyId: string,
): Promise<TokenBlueprint[]> {
  const cid = companyId.trim();
  if (!cid) return [];
  const all = await listTokenBlueprintsByCompanyId(cid);
  return listNotYet(all);
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
 * - minted = "notYet" のみを対象にする（ListMintedNotYet 相当のフィルタ）
 */
export function filterAndSortTokenBlueprints(
  rows: TokenBlueprint[],
  state: TokenBlueprintFilterState,
): TokenBlueprint[] {
  const { brandFilter, assigneeFilter, sortKey, sortDir } = state;

  // ★ まず minted = "notYet" のみを対象に絞り込む
  let data = listNotYet(rows).filter(
    (r) =>
      (brandFilter.length === 0 || brandFilter.includes(r.brandId)) &&
      (assigneeFilter.length === 0 || assigneeFilter.includes(r.assigneeId)),
  );

  if (sortKey && sortDir) {
    data = [...data].sort((a, b) => {
      const av = toTs(a[sortKey] as string);
      const bv = toTs(b[sortKey] as string);
      return sortDir === "asc" ? av - bv : bv - av;
    });
  }

  return data;
}
