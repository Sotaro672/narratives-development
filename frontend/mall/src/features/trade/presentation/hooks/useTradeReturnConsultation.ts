// frontend/mall/src/features/trade/presentation/hooks/useTradeReturnConsultation.ts

import { useCallback, useState } from "react";

import type {
  TradeDetail,
  TradeReturnConsultationReason,
} from "../../../shared/types/trade";
import { createTradeReturnConsultation } from "../../infrastructure/tradeApi";
import { getErrorMessage } from "../util/tradeChatDetail";

type UseTradeReturnConsultationInput = {
  tradeId: string;
  trade: TradeDetail | null;
  reload: () => Promise<void>;
  blocked: boolean;
};

const MAX_DETAIL_LENGTH = 5000;

export function useTradeReturnConsultation({
  tradeId,
  trade,
  reload,
  blocked,
}: UseTradeReturnConsultationInput) {
  const [open, setOpen] = useState(false);
  const [reason, setReasonState] =
    useState<TradeReturnConsultationReason | null>(null);
  const [detail, setDetailState] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const setReason = useCallback(
    (value: TradeReturnConsultationReason | null) => {
      setReasonState(value);
      setError("");
    },
    [],
  );

  const setDetail = useCallback((value: string) => {
    setDetailState(value);
    setError("");
  }, []);

  const reset = useCallback(() => {
    setReasonState(null);
    setDetailState("");
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
      trade.returnStatus !== "none" ||
      !tradeId
    ) {
      return;
    }

    reset();
    setOpen(true);
  }, [blocked, reset, trade, tradeId]);

  const closeModal = useCallback(() => {
    if (submitting) {
      return;
    }

    setOpen(false);
    reset();
  }, [reset, submitting]);

  const submit = useCallback(async (): Promise<void> => {
    if (
      submitting ||
      blocked ||
      !trade ||
      trade.viewerSide !== "buyer" ||
      trade.status !== "active" ||
      trade.isCancelled ||
      !trade.isDispatched ||
      trade.transferred ||
      trade.returnStatus !== "none" ||
      !tradeId
    ) {
      return;
    }

    if (!reason) {
      setError("返品理由を選択してください。");
      return;
    }

    const normalizedDetail = detail.trim();

    if (!normalizedDetail) {
      setError("詳細を入力してください。");
      return;
    }

    if (normalizedDetail.length > MAX_DETAIL_LENGTH) {
      setError(
        `詳細は${MAX_DETAIL_LENGTH}文字以内で入力してください。`,
      );
      return;
    }

    setSubmitting(true);
    setError("");

    try {
      await createTradeReturnConsultation({
        tradeId,
        reason,
        detail: normalizedDetail,
      });

      setOpen(false);
      reset();
      await reload();
    } catch (caught) {
      setError(
        getErrorMessage(
          caught,
          "返品についての相談を開始できませんでした。",
        ),
      );
    } finally {
      setSubmitting(false);
    }
  }, [
    blocked,
    detail,
    reason,
    reload,
    reset,
    submitting,
    trade,
    tradeId,
  ]);

  return {
    open,
    reason,
    detail,
    error,
    submitting,
    openModal,
    closeModal,
    setReason,
    setDetail,
    submit,
  };
}

export default useTradeReturnConsultation;