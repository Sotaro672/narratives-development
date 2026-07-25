// frontend/console/tokenBlueprintReview/src/application/tokenBlueprintReviewManagementService.tsx

import type { TokenBlueprintReviewAggregate } from "../domain/entity";
import { listTokenBlueprintReviewAggregatesByCompanyId } from "../infrastructure/tokenBlueprintReviewRepositoryHTTP";

/** ISO8601 -> timestamp（不正値は 0 扱い） */
const toTs = (iso: string): number => {
  if (!iso) return 0;
  const t = Date.parse(iso);
  return Number.isNaN(t) ? 0 : t;
};

export type SortKey = "createdAt" | "updatedAt" | null;
export type SortDir = "asc" | "desc" | null;

export type TokenBlueprintReviewFilterState = {
  brandFilter?: string[];
  assigneeFilter?: string[];
  sortKey: SortKey;
  sortDir: SortDir;
};

/**
 * トークン設計レビュー一覧（Aggregate）を取得
 * - backend 側で companyId は auth context から解決される想定
 * - フロントでは companyId の有無でガードしない
 */
export async function fetchTokenBlueprintReviewsForCompany(
  companyId: string,
): Promise<TokenBlueprintReviewAggregate[]> {
  const all = await listTokenBlueprintReviewAggregatesByCompanyId(String(companyId ?? ""));
  return all;
}

/**
 * 一覧から brand / assignee のフィルタオプションを生成
 *
 * NOTE:
 * TokenBlueprintReviewAggregate 自体には brandId / assigneeId が無い設計なので、
 * repository から返ってくる行に brandId / assigneeId が載っている前提で拾う（無ければ空になる）。
 */
export function buildOptionsFromTokenBlueprintReviews(
  rows: TokenBlueprintReviewAggregate[],
): {
  brandOptions: { value: string; label: string }[];
  assigneeOptions: { value: string; label: string }[];
} {
  const brandSet = new Set<string>();
  const assigneeSet = new Set<string>();

  for (const r of rows) {
    const row = r as TokenBlueprintReviewAggregate & {
      brandId?: string;
      assigneeId?: string;
    };

    const bid = String(row.brandId ?? "");
    const aid = String(row.assigneeId ?? "");

    if (bid) brandSet.add(bid);
    if (aid) assigneeSet.add(aid);
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
 * フィルタ＋ソートを適用した TokenBlueprintReview 一覧を返す
 * - createdAt / updatedAt は camelCase の domain model を使う
 */
export function filterAndSortTokenBlueprintReviews(
  rows: TokenBlueprintReviewAggregate[],
  state: TokenBlueprintReviewFilterState,
): TokenBlueprintReviewAggregate[] {
  const brandFilter = state.brandFilter ?? [];
  const assigneeFilter = state.assigneeFilter ?? [];
  const { sortKey, sortDir } = state;

  let data = (rows ?? []).filter((r) => {
    const row = r as TokenBlueprintReviewAggregate & {
      brandId?: string;
      assigneeId?: string;
    };

    const bid = String(row.brandId ?? "");
    const aid = String(row.assigneeId ?? "");

    return (
      (brandFilter.length === 0 || (bid && brandFilter.includes(bid))) &&
      (assigneeFilter.length === 0 || (aid && assigneeFilter.includes(aid)))
    );
  });

  if (sortKey && sortDir) {
    data = [...data].sort((a, b) => {
      const av = toTs(String(a[sortKey] ?? ""));
      const bv = toTs(String(b[sortKey] ?? ""));
      return sortDir === "asc" ? av - bv : bv - av;
    });
  }

  return data;
}