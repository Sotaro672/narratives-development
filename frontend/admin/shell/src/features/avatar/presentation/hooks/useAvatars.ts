// frontend/admin/shell/src/features/avatar/presentation/hooks/useAvatars.ts

import { useCallback, useEffect, useState } from "react";

import type { Avatar } from "../../../../shared/type/avatar";
import { listAvatars } from "../../infrastructure/avatarApi";

export function useAvatars() {
  const [avatars, setAvatars] = useState<Avatar[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);

    try {
      const result = await listAvatars();
      setAvatars(result);
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "アバター一覧の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  return {
    avatars,
    loading,
    error,
    reload,
  };
}