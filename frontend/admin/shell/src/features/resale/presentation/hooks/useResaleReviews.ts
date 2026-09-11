// frontend/admin/shell/src/features/resale/presentation/hooks/useResaleReviews.ts

import { useCallback, useEffect, useState } from "react";

import type { ResaleReviewResponse } from "../../../../shared/type/resaleReview";
import { getResaleReviews } from "../../infrastructure/resaleApi";

export function useResaleReviews(
  avatarId: string | undefined,
  resaleId: string | undefined,
  page = 1,
  perPage = 20,
) {
  const [reviews, setReviews] = useState<ResaleReviewResponse | null>(null);
  const [loading, setLoading] = useState(
    Boolean(avatarId?.trim() && resaleId?.trim()),
  );
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    const normalizedAvatarId = avatarId?.trim() ?? "";
    const normalizedResaleId = resaleId?.trim() ?? "";

    if (!normalizedAvatarId || !normalizedResaleId) {
      setReviews(null);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const result = await getResaleReviews(
        normalizedAvatarId,
        normalizedResaleId,
        page,
        perPage,
      );
      setReviews(result);
    } catch (cause) {
      setReviews(null);
      setError(
        cause instanceof Error
          ? cause.message
          : "リセールレビューの取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [avatarId, resaleId, page, perPage]);

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