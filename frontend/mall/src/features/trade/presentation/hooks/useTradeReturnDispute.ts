// frontend/mall/src/features/trade/presentation/hooks/useTradeReturnDispute.ts

import { useCallback, useState } from "react";

import type { TradeDetail } from "../../../shared/types/trade";
import { reportTradeReturnDispute } from "../../infrastructure/tradeApi";
import { getErrorMessage } from "../util/tradeChatDetail";

type UseTradeReturnDisputeInput = {
  tradeId: string;
  trade: TradeDetail | null;
  reload: () => Promise<void>;
  blocked: boolean;
};

const MAX_DETAIL_LENGTH = 5000;

function canReportReturnDispute(
  tradeId: string,
  trade: TradeDetail | null,
  blocked: boolean,
): boolean {
  return (
    !blocked &&
    tradeId.trim() !== "" &&
    trade !== null &&
    trade.viewerSide === "buyer" &&
    trade.status === "active" &&
    !trade.isCancelled &&
    trade.isDispatched &&
    !trade.transferred &&
    trade.returnStatus === "discussing" &&
    trade.returnProposal?.agreement === "disagree"
  );
}

export function useTradeReturnDispute({
  tradeId,
  trade,
  reload,
  blocked,
}: UseTradeReturnDisputeInput) {
  const [open, setOpen] = useState(false);
  const [content, setContentState] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const normalizedContent = content.trim();
  const canSubmit =
    !submitting &&
    !blocked &&
    normalizedContent.length > 0 &&
    normalizedContent.length <= MAX_DETAIL_LENGTH &&
    canReportReturnDispute(tradeId, trade, false);

  const setContent = useCallback((value: string) => {
    setContentState(value);
    setError("");
  }, []);

  const reset = useCallback(() => {
    setContentState("");
    setError("");
  }, []);

  const openModal = useCallback(() => {
    if (!canReportReturnDispute(tradeId, trade, blocked)) {
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
      !canReportReturnDispute(tradeId, trade, blocked)
    ) {
      return;
    }

    const detail = content.trim();

    if (!detail) {
      setError("運営へ報告する内容を入力してください。");
      return;
    }

    if (detail.length > MAX_DETAIL_LENGTH) {
      setError(
        `報告内容は${MAX_DETAIL_LENGTH}文字以内で入力してください。`,
      );
      return;
    }

    setSubmitting(true);
    setError("");

    try {
      await reportTradeReturnDispute({
        tradeId,
        detail,
      });

      setOpen(false);
      reset();
      await reload();
    } catch (caught) {
      setError(
        getErrorMessage(
          caught,
          "取引を運営へ報告できませんでした。",
        ),
      );
    } finally {
      setSubmitting(false);
    }
  }, [
    blocked,
    content,
    reload,
    reset,
    submitting,
    trade,
    tradeId,
  ]);

  return {
    open,
    content,
    error,
    submitting,
    canSubmit,
    openModal,
    closeModal,
    setContent,
    submit,
  };
}

export default useTradeReturnDispute;