// frontend/mall/src/features/trade/presentation/hooks/useTradeReturnProposal.ts

import { useCallback, useState } from "react";

import type {
  TradeDetail,
  TradeReturnAgreement,
  TradeReturnRequirement,
} from "../../../shared/types/trade";
import { createTradeReturnProposal } from "../../infrastructure/tradeApi";
import { getErrorMessage } from "../util/tradeChatDetail";

type UseTradeReturnProposalInput = {
  tradeId: string;
  trade: TradeDetail | null;
  reload: () => Promise<void>;
  blocked: boolean;
};

function canProposeReturn(
  tradeId: string,
  trade: TradeDetail | null,
  blocked: boolean,
): boolean {
  return (
    !blocked &&
    tradeId.trim() !== "" &&
    trade !== null &&
    trade.viewerSide === "seller" &&
    trade.status === "active" &&
    !trade.isCancelled &&
    trade.isDispatched &&
    !trade.transferred &&
    trade.returnStatus === "discussing"
  );
}

export function useTradeReturnProposal({
  tradeId,
  trade,
  reload,
  blocked,
}: UseTradeReturnProposalInput) {
  const [open, setOpen] = useState(false);
  const [agreement, setAgreementState] =
    useState<TradeReturnAgreement | null>(null);
  const [returnRequirement, setReturnRequirementState] =
    useState<TradeReturnRequirement | null>(null);
  const [refundAmount, setRefundAmountState] =
    useState<number | "">("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const refundAmountMax =
    trade?.merchandiseRefundMaxAmount ?? 0;

  const reset = useCallback(() => {
    setAgreementState(null);
    setReturnRequirementState(null);
    setRefundAmountState("");
    setError("");
  }, []);

  const setAgreement = useCallback(
    (value: TradeReturnAgreement) => {
      setAgreementState(value);
      setError("");

      if (value === "disagree") {
        setReturnRequirementState(null);
        setRefundAmountState("");
      }
    },
    [],
  );

  const setReturnRequirement = useCallback(
    (value: TradeReturnRequirement) => {
      setReturnRequirementState(value);
      setError("");
    },
    [],
  );

  const setRefundAmount = useCallback(
    (value: number | "") => {
      setRefundAmountState(value);
      setError("");
    },
    [],
  );

  const openModal = useCallback(() => {
    if (!canProposeReturn(tradeId, trade, blocked)) {
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
      !canProposeReturn(tradeId, trade, blocked)
    ) {
      return;
    }

    if (!agreement) {
      setError("返品に合意するか選択してください。");
      return;
    }

    if (agreement === "agree") {
      if (!returnRequirement) {
        setError(
          "商品を返品してもらうか選択してください。",
        );
        return;
      }

      if (refundAmount === "") {
        setError("返金額を入力してください。");
        return;
      }

      if (!Number.isInteger(refundAmount)) {
        setError(
          "返金額は1円単位の整数で入力してください。",
        );
        return;
      }

      if (refundAmount <= 0) {
        setError("返金額は1円以上で入力してください。");
        return;
      }

      if (
        !Number.isInteger(refundAmountMax) ||
        refundAmountMax <= 0
      ) {
        setError(
          "返金可能額を取得できません。取引情報を再読み込みしてください。",
        );
        return;
      }

      if (refundAmount > refundAmountMax) {
        setError(
          `返金額は${refundAmountMax.toLocaleString(
            "ja-JP",
          )}円以下で入力してください。`,
        );
        return;
      }
    }

    setSubmitting(true);
    setError("");

    try {
      if (agreement === "disagree") {
        await createTradeReturnProposal({
          tradeId,
          agreement: "disagree",
        });
      } else {
        await createTradeReturnProposal({
          tradeId,
          agreement: "agree",
          returnRequirement: returnRequirement!,
          refundAmount: refundAmount as number,
        });
      }

      setOpen(false);
      reset();

      await reload();
    } catch (caught) {
      setError(
        getErrorMessage(
          caught,
          "返品相談への回答を送信できませんでした。",
        ),
      );
    } finally {
      setSubmitting(false);
    }
  }, [
    agreement,
    blocked,
    refundAmount,
    refundAmountMax,
    reload,
    reset,
    returnRequirement,
    submitting,
    trade,
    tradeId,
  ]);

  return {
    open,
    agreement,
    returnRequirement,
    refundAmount,
    refundAmountMax,
    error,
    submitting,
    openModal,
    closeModal,
    setAgreement,
    setReturnRequirement,
    setRefundAmount,
    submit,
  };
}

export default useTradeReturnProposal;