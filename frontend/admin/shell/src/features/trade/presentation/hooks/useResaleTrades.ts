// frontend/admin/shell/src/features/trade/presentation/hooks/useResaleTrades.ts

import { useCallback, useEffect, useState } from "react";

import type { ResaleTradeListResponse } from "../../../../shared/type/trade";
import { getResaleTrades } from "../../infrastructure/tradeApi";

export function useResaleTrades(
  resaleId: string | undefined,
) {
  const [trades, setTrades] = useState<ResaleTradeListResponse | null>(null);
  const [loading, setLoading] = useState(Boolean(resaleId?.trim()));
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    const normalizedResaleId = resaleId?.trim() ?? "";

    if (!normalizedResaleId) {
      setTrades(null);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const result = await getResaleTrades(normalizedResaleId);
      setTrades(result);
    } catch (cause) {
      setTrades(null);
      setError(
        cause instanceof Error
          ? cause.message
          : "取引情報の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [resaleId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return {
    trades,
    loading,
    error,
    reload,
  };
}