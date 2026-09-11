// frontend/admin/shell/src/features/trade/presentation/hooks/useTradeMessages.ts

import { useCallback, useEffect, useState } from "react";

import type { TradeMessageListResponse } from "../../../../shared/type/trade";
import { getTradeMessages } from "../../infrastructure/tradeApi";

export function useTradeMessages(
  tradeId: string | undefined,
) {
  const [messages, setMessages] = useState<TradeMessageListResponse | null>(null);
  const [loading, setLoading] = useState(Boolean(tradeId?.trim()));
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    const normalizedTradeId = tradeId?.trim() ?? "";

    if (!normalizedTradeId) {
      setMessages(null);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const result = await getTradeMessages(normalizedTradeId);
      setMessages(result);
    } catch (cause) {
      setMessages(null);
      setError(
        cause instanceof Error
          ? cause.message
          : "取引メッセージの取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [tradeId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return {
    messages,
    loading,
    error,
    reload,
  };
}