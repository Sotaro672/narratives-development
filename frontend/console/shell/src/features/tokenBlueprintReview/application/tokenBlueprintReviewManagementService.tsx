// frontend/console/shell/src/features/tokenBlueprintReview/application/tokenBlueprintReviewManagementService.tsx

import type { TokenBlueprintReviewAggregate } from "../../../shared/types/tokenBlueprintReview";
import { listTokenBlueprintReviewAggregatesByCompanyId } from "../infrastructure/tokenBlueprintReviewRepositoryHTTP";

/** ISO8601 -> timestamp（不正値は 0 扱い） */
function toTimestamp(iso: string): number {
  if (!iso) {
    return 0;
  }

  const timestamp = Date.parse(iso);

  return Number.isNaN(timestamp) ? 0 : timestamp;
}

export type SortKey = "createdAt" | "updatedAt" | null;
export type SortDir = "asc" | "desc" | null;

type TokenBlueprintReviewSortState = {
  sortKey: SortKey;
  sortDir: SortDir;
};

/**
 * トークン設計レビュー一覧（Aggregate）を取得する。
 *
 * backend側では、認証情報からcompanyIdを解決する。
 * 現在は既存の呼び出し元との互換性を保つため、companyIdを受け取る。
 */
export async function fetchTokenBlueprintReviewsForCompany(
  companyId: string,
): Promise<TokenBlueprintReviewAggregate[]> {
  return listTokenBlueprintReviewAggregatesByCompanyId(
    String(companyId ?? ""),
  );
}

/**
 * TokenBlueprintReview一覧を作成日時または更新日時で並び替える。
 *
 * brandNameによるフィルター処理は、Management hook側で適用する。
 */
export function filterAndSortTokenBlueprintReviews(
  rows: TokenBlueprintReviewAggregate[],
  state: TokenBlueprintReviewSortState,
): TokenBlueprintReviewAggregate[] {
  const data = rows ?? [];
  const { sortKey, sortDir } = state;

  if (!sortKey || !sortDir) {
    return data;
  }

  return [...data].sort((a, b) => {
    const leftTimestamp = toTimestamp(String(a[sortKey] ?? ""));
    const rightTimestamp = toTimestamp(String(b[sortKey] ?? ""));

    return sortDir === "asc"
      ? leftTimestamp - rightTimestamp
      : rightTimestamp - leftTimestamp;
  });
}