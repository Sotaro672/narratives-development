// frontend/mall/src/features/trade/presentation/hooks/useTradeThread.ts

import { useCallback, useEffect, useMemo, useState } from "react";

import type { TradeDetail } from "../../../shared/types/trade";
import {
  fetchTradeById,
  markTradeMessagesAsRead,
} from "../../infrastructure/tradeApi";
import {
  getErrorMessage,
  sortTradeMessages,
} from "../util/tradeChatDetail";

const MESSAGE_LIMIT = 100;

export function useTradeThread(tradeId: string) {
  const normalizedTradeId = tradeId.trim();

  const [trade, setTrade] = useState<TradeDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const reload = useCallback(async (): Promise<void> => {
    if (!normalizedTradeId) {
      setTrade(null);
      setError("取引IDが見つかりません。");
      setLoading(false);
      return;
    }

    setLoading(true);
    setError("");

    try {
      const loadedTrade = await fetchTradeById({
        tradeId: normalizedTradeId,
        limit: MESSAGE_LIMIT,
      });

      setTrade({
        ...loadedTrade,
        messages: sortTradeMessages(loadedTrade.messages),
      });

      try {
        await markTradeMessagesAsRead({
          tradeId: normalizedTradeId,
        });
      } catch (caught) {
        setError(
          getErrorMessage(
            caught,
            "メッセージの既読状態の更新に失敗しました。",
          ),
        );
      }
    } catch (caught) {
      setTrade(null);
      setError(
        getErrorMessage(
          caught,
          "取引の取得に失敗しました。",
        ),
      );
    } finally {
      setLoading(false);
    }
  }, [normalizedTradeId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const messages = useMemo(
    () => sortTradeMessages(trade?.messages ?? []),
    [trade?.messages],
  );

  return {
    tradeId: normalizedTradeId,
    trade,
    setTrade,
    messages,
    loading,
    error,
    reload,
  };
}

export default useTradeThread;