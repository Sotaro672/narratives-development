// frontend/console/shell/src/features/tokenBlueprintReview/application/tokenBlueprintReviewManagementService.tsx

import type { TokenBlueprintReviewAggregate } from "../../../shared/types/tokenBlueprintReview";
import { listTokenBlueprintReviewAggregates } from "../infrastructure/tokenBlueprintReviewRepositoryHTTP";

export type SortKey = "createdAt" | "updatedAt" | null;
export type SortDir = "asc" | "desc" | null;

type TokenBlueprintReviewSortState = {
  sortKey: SortKey;
  sortDir: SortDir;
};

/**
 * トークン設計レビュー一覧（Aggregate）を取得する。
 * companyId は backend の認証 context から解決する。
 */
export async function fetchTokenBlueprintReviews(): Promise<TokenBlueprintReviewAggregate[]> {
  return listTokenBlueprintReviewAggregates();
}

/**
 * TokenBlueprintReview 一覧を作成日時または更新日時で並び替える。
 * brandName によるフィルター処理は Management hook 側で適用する。
 */
export function filterAndSortTokenBlueprintReviews(
  rows: TokenBlueprintReviewAggregate[],
  state: TokenBlueprintReviewSortState,
): TokenBlueprintReviewAggregate[] {
  const { sortKey, sortDir } = state;

  if (!sortKey || !sortDir) {
    return rows;
  }

  return [...rows].sort((a, b) => {
    const leftTimestamp = Date.parse(a[sortKey]);
    const rightTimestamp = Date.parse(b[sortKey]);

    return sortDir === "asc"
      ? leftTimestamp - rightTimestamp
      : rightTimestamp - leftTimestamp;
  });
}