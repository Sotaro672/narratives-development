// frontend/admin/shell/src/features/avatar/presentation/hooks/useAvatarResales.ts

import { useCallback, useEffect, useState } from "react";

import type { Resale } from "../../../../shared/type/resale";
import { listAvatarResales } from "../../infrastructure/avatarApi";

export function useAvatarResales(avatarId: string) {
  const [resales, setResales] = useState<Resale[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    if (!avatarId) {
      setResales([]);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      setResales(await listAvatarResales(avatarId));
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Resale一覧の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [avatarId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { resales, loading, error, reload };
}