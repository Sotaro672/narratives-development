// frontend/admin/shell/src/features/resale/presentation/hooks/useResaleDetail.ts

import { useCallback, useEffect, useState } from "react";

import type { Resale } from "../../../../shared/type/resale";
import { getResaleDetail } from "../../infrastructure/resaleApi";

export function useResaleDetail(avatarId: string, resaleId: string) {
  const [resale, setResale] = useState<Resale | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    if (!avatarId || !resaleId) {
      setResale(null);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      setResale(await getResaleDetail(avatarId, resaleId));
    } catch (cause) {
      setResale(null);
      setError(
        cause instanceof Error
          ? cause.message
          : "Resaleの取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [avatarId, resaleId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { resale, loading, error, reload };
}