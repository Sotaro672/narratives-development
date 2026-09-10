// frontend/admin/shell/src/features/mint/hooks/useMintDetail.ts

import { useCallback, useEffect, useState } from "react";

import type { MintDetail } from "../../../shared/type/mint";
import { getMintDetail } from "../infrastructure/mintApi";

export function useMintDetail(mintId: string | undefined) {
  const [detail, setDetail] = useState<MintDetail | null>(null);
  const [loading, setLoading] = useState(Boolean(mintId?.trim()));
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    const normalizedMintId = mintId?.trim() ?? "";

    if (!normalizedMintId) {
      setDetail(null);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);
    setDetail(null);

    try {
      const result = await getMintDetail(normalizedMintId);
      setDetail(result);
    } catch (cause) {
      setDetail(null);
      setError(
        cause instanceof Error
          ? cause.message
          : "Mint詳細の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [mintId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return {
    detail,
    loading,
    error,
    reload,
  };
}