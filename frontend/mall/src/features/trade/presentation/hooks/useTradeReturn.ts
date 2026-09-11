// frontend/mall/src/features/trade/presentation/hooks/useTradeReturn.ts

import { useCallback, useState } from "react";

import { getApiBaseUrl } from "../../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../../lib/authToken";
import { returnOrderItem } from "../../../order/api/orderDetailApi";
import type { ReturnPackageState } from "../../../order/components/ReturnRequestModal";
import type { TradeDetail } from "../../../shared/types/trade";
import { createTradeMessage } from "../../infrastructure/tradeApi";
import { getErrorMessage } from "../util/tradeChatDetail";

type UseTradeReturnInput = {
  tradeId: string;
  trade: TradeDetail | null;
  reload: () => Promise<void>;
  blocked: boolean;
};

export function useTradeReturn({
  tradeId,
  trade,
  reload,
  blocked,
}: UseTradeReturnInput) {
  const [open, setOpen] = useState(false);
  const [packageState, setPackageStateState] =
    useState<ReturnPackageState | null>(null);
  const [reason, setReasonState] = useState("");
  const [error, setError] = useState("");
  const [returning, setReturning] = useState(false);

  const setPackageState = useCallback(
    (value: ReturnPackageState | null) => {
      setPackageStateState(value);
      setError("");
    },
    [],
  );

  const setReason = useCallback((value: string) => {
    setReasonState(value);
    setError("");
  }, []);

  const openModal = useCallback(() => {
    if (
      blocked ||
      !trade ||
      trade.viewerSide !== "buyer" ||
      trade.status !== "active" ||
      trade.isCancelled ||
      !trade.isDispatched ||
      trade.transferred ||
      trade.isReturnRequested ||
      trade.isReturnCompleted ||
      !tradeId
    ) {
      return;
    }

    setPackageStateState(null);
    setReasonState("");
    setError("");
    setOpen(true);
  }, [blocked, trade, tradeId]);

  const closeModal = useCallback(() => {
    if (returning) {
      return;
    }

    setOpen(false);
    setPackageStateState(null);
    setReasonState("");
    setError("");
  }, [returning]);

  const submit = useCallback(async (): Promise<void> => {
    if (
      returning ||
      blocked ||
      !trade ||
      trade.viewerSide !== "buyer" ||
      trade.status !== "active" ||
      trade.isCancelled ||
      !trade.isDispatched ||
      trade.transferred ||
      trade.isReturnRequested ||
      trade.isReturnCompleted ||
      !tradeId
    ) {
      return;
    }

    if (
      packageState !== "unopened" &&
      packageState !== "opened"
    ) {
      setError("商品の開封状態を選択してください。");
      return;
    }

    const normalizedReason = reason.trim();
    if (!normalizedReason) {
      setError("返品理由を入力してください。");
      return;
    }

    setReturning(true);
    setError("");

    try {
      const backendUrl = getApiBaseUrl();
      if (!backendUrl) {
        throw new Error("VITE_API_BASE_URLが設定されていません。");
      }

      const idToken = await getFirebaseIdToken();

      await createTradeMessage({
        tradeId,
        content: normalizedReason,
      });

      await returnOrderItem({
        backendUrl,
        idToken,
        orderId: trade.orderId,
        itemIndex: trade.orderItemIndex,
        packageState,
        reason: normalizedReason,
      });

      setOpen(false);
      setPackageStateState(null);
      setReasonState("");
      setError("");

      await reload();
    } catch (caught) {
      setError(
        getErrorMessage(
          caught,
          "商品の返品受付に失敗しました。",
        ),
      );
    } finally {
      setReturning(false);
    }
  }, [
    blocked,
    packageState,
    reason,
    reload,
    returning,
    trade,
    tradeId,
  ]);

  return {
    open,
    packageState,
    reason,
    error,
    returning,
    openModal,
    closeModal,
    setPackageState,
    setReason,
    submit,
  };
}

export default useTradeReturn;