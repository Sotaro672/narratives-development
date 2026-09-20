// frontend/mall/src/features/order/hooks/useOrderTradeNavigation.ts

import { useCallback, useState } from "react";
import { useNavigate } from "react-router-dom";

import { fetchTradeByOrderItem } from "../../trade/infrastructure/tradeApi";

export function useOrderTradeNavigation() {
  const navigate = useNavigate();

  const [openingItemIndex, setOpeningItemIndex] = useState<number | null>(null);
  const [error, setError] = useState("");

  const openTrade = useCallback(
    async (orderId: string, itemIndex: number) => {
      const normalizedOrderId = orderId.trim();

      if (
        !normalizedOrderId ||
        !Number.isInteger(itemIndex) ||
        itemIndex < 0 ||
        openingItemIndex !== null
      ) {
        return;
      }

      setOpeningItemIndex(itemIndex);
      setError("");

      try {
        const trade = await fetchTradeByOrderItem({
          orderId: normalizedOrderId,
          orderItemIndex: itemIndex,
          limit: 1,
        });

        if (!trade.id) {
          throw new Error("取引が見つかりません。");
        }

        navigate(`/chats/trades/${encodeURIComponent(trade.id)}`, {
          state: {
            trade,
          },
        });
      } catch (caught) {
        setError(
          caught instanceof Error
            ? caught.message
            : "取引を開けませんでした。",
        );
      } finally {
        setOpeningItemIndex(null);
      }
    },
    [navigate, openingItemIndex],
  );

  return {
    openingItemIndex,
    error,
    openTrade,
  };
}