// frontend/admin/shell/src/features/tokenBlueprint/presentation/hooks/useTokenBlueprintReviews.ts

import { useCallback, useEffect, useState } from "react";

import { getContractTokenBlueprintReviews } from "../../infrastructure/tokenBlueprintApi";
import type { ContractTokenBlueprintReviewResponse } from "../../model/tokenBlueprintReview";

export function useTokenBlueprintReviews(
  companyId: string | undefined,
  tokenBlueprintId: string | undefined,
  page = 1,
  perPage = 20,
) {
  const [reviews, setReviews] =
    useState<ContractTokenBlueprintReviewResponse | null>(null);
  const [loading, setLoading] = useState(
    Boolean(companyId?.trim() && tokenBlueprintId?.trim()),
  );
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    const normalizedCompanyId = companyId?.trim() ?? "";
    const normalizedTokenBlueprintId = tokenBlueprintId?.trim() ?? "";

    if (!normalizedCompanyId || !normalizedTokenBlueprintId) {
      setReviews(null);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const result = await getContractTokenBlueprintReviews(
        normalizedCompanyId,
        normalizedTokenBlueprintId,
        page,
        perPage,
      );
      setReviews(result);
    } catch (cause) {
      setReviews(null);
      setError(
        cause instanceof Error
          ? cause.message
          : "トークン設計レビューの取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [companyId, tokenBlueprintId, page, perPage]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return {
    reviews,
    loading,
    error,
    reload,
  };
}