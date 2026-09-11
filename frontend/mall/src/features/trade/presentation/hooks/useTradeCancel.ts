// frontend/mall/src/features/trade/presentation/hooks/useTradeCancel.ts

import { useCallback, useState } from "react";

import type { TradeDetail } from "../../../shared/types/trade";
import {
  cancelTradeOrderItem,
  createTradeMessage,
} from "../../infrastructure/tradeApi";
import { getErrorMessage } from "../util/tradeChatDetail";

type UseTradeCancelInput = {
  tradeId: string;
  trade: TradeDetail | null;
  reload: () => Promise<void>;
};

export function useTradeCancel({
  tradeId,
  trade,
  reload,
}: UseTradeCancelInput) {
  const [open, setOpen] = useState(false);
  const [message, setMessageState] = useState("");
  const [error, setError] = useState("");
  const [cancelling, setCancelling] = useState(false);

  const canSubmit = /\S/u.test(message);

  const setMessage = useCallback((value: string) => {
    setMessageState(value);
    setError("");
  }, []);

  const openModal = useCallback(() => {
    if (
      !trade ||
      trade.viewerSide !== "buyer" ||
      trade.status !== "active" ||
      trade.isCancelled ||
      trade.isDispatched ||
      trade.transferred
    ) {
      return;
    }

    setMessageState("");
    setError("");
    setOpen(true);
  }, [trade]);

  const closeModal = useCallback(() => {
    if (cancelling) {
      return;
    }

    setOpen(false);
    setMessageState("");
    setError("");
  }, [cancelling]);

  const submit = useCallback(async (): Promise<void> => {
    if (
      cancelling ||
      !trade ||
      trade.viewerSide !== "buyer" ||
      trade.status !== "active" ||
      trade.isCancelled ||
      trade.isDispatched ||
      trade.transferred ||
      !tradeId
    ) {
      return;
    }

    const normalizedMessage = message.trim();
    if (!normalizedMessage) {
      setError("キャンセル理由のメッセージを入力してください。");
      return;
    }

    setCancelling(true);
    setError("");

    try {
      await createTradeMessage({
        tradeId,
        content: normalizedMessage,
      });

      await cancelTradeOrderItem({
        orderId: trade.orderId,
        orderItemIndex: trade.orderItemIndex,
      });

      setOpen(false);
      setMessageState("");
      setError("");

      await reload();
    } catch (caught) {
      setError(
        getErrorMessage(
          caught,
          "注文のキャンセルに失敗しました。",
        ),
      );
    } finally {
      setCancelling(false);
    }
  }, [
    cancelling,
    message,
    reload,
    trade,
    tradeId,
  ]);

  return {
    open,
    message,
    error,
    cancelling,
    canSubmit,
    openModal,
    closeModal,
    setMessage,
    submit,
  };
}

export default useTradeCancel;