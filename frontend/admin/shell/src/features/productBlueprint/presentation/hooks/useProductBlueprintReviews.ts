// frontend/admin/shell/src/features/productBlueprint/presentation/hooks/useProductBlueprintReviews.ts

import { useCallback, useEffect, useState } from "react";

import { getContractProductBlueprintReviews } from "../../infrastructure/productBlueprintApi";
import type {
  ContractProductBlueprintReviewResponse,
  ContractProductBlueprintReviewStatus,
} from "../../model/productBlueprintReview";

export function useProductBlueprintReviews(
  companyId: string | undefined,
  productBlueprintId: string | undefined,
  status: ContractProductBlueprintReviewStatus = "PUBLISHED",
  page = 1,
  perPage = 20,
) {
  const [reviews, setReviews] =
    useState<ContractProductBlueprintReviewResponse | null>(null);
  const [loading, setLoading] = useState(
    Boolean(companyId?.trim() && productBlueprintId?.trim()),
  );
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    const normalizedCompanyId = companyId?.trim() ?? "";
    const normalizedProductBlueprintId = productBlueprintId?.trim() ?? "";

    if (!normalizedCompanyId || !normalizedProductBlueprintId) {
      setReviews(null);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const result = await getContractProductBlueprintReviews(
        normalizedCompanyId,
        normalizedProductBlueprintId,
        status,
        page,
        perPage,
      );
      setReviews(result);
    } catch (cause) {
      setReviews(null);
      setError(
        cause instanceof Error
          ? cause.message
          : "商品設計レビューの取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [companyId, productBlueprintId, status, page, perPage]);

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