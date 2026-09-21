// frontend/mall/src/features/trade/presentation/hooks/useTradeReturnAgreement.ts

import { useCallback, useState } from "react";

import type {
  TradeDetail,
  TradeReturnProposal,
} from "../../../shared/types/trade";
import {
  acceptTradeReturnProposal,
  rejectTradeReturnProposal,
} from "../../infrastructure/tradeApi";
import { getErrorMessage } from "../util/tradeChatDetail";

type UseTradeReturnAgreementInput = {
  tradeId: string;
  trade: TradeDetail | null;
  reload: () => Promise<void>;
  blocked: boolean;
};

type TradeReturnAgreementAction = "accept" | "reject";

function getReviewableProposal(
  tradeId: string,
  trade: TradeDetail | null,
  blocked: boolean,
): TradeReturnProposal | null {
  if (
    blocked ||
    tradeId.trim() === "" ||
    trade === null ||
    trade.viewerSide !== "buyer" ||
    trade.status !== "active" ||
    trade.isCancelled ||
    !trade.isDispatched ||
    trade.transferred ||
    trade.returnStatus !== "proposed"
  ) {
    return null;
  }

  const proposal = trade.returnProposal;

  if (
    !proposal ||
    proposal.id.trim() === "" ||
    proposal.agreement !== "agree" ||
    proposal.rejectedAt
  ) {
    return null;
  }

  if (
    proposal.returnRequirement !== "required" &&
    proposal.returnRequirement !== "not_required"
  ) {
    return null;
  }

  if (
    proposal.refundAmount === undefined ||
    !Number.isInteger(proposal.refundAmount) ||
    proposal.refundAmount <= 0
  ) {
    return null;
  }

  return proposal;
}

export function useTradeReturnAgreement({
  tradeId,
  trade,
  reload,
  blocked,
}: UseTradeReturnAgreementInput) {
  const [open, setOpen] = useState(false);
  const [error, setError] = useState("");
  const [submittingAction, setSubmittingAction] =
    useState<TradeReturnAgreementAction | null>(null);

  const proposal = getReviewableProposal(
    tradeId,
    trade,
    blocked,
  );

  const submitting = submittingAction !== null;
  const accepting = submittingAction === "accept";
  const rejecting = submittingAction === "reject";

  const reset = useCallback(() => {
    setError("");
    setSubmittingAction(null);
  }, []);

  const openModal = useCallback(() => {
    const currentProposal = getReviewableProposal(
      tradeId,
      trade,
      blocked,
    );

    if (!currentProposal) {
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

  const accept = useCallback(async (): Promise<void> => {
    if (submitting) {
      return;
    }

    const currentProposal = getReviewableProposal(
      tradeId,
      trade,
      blocked,
    );

    if (!currentProposal) {
      setError(
        "返品条件を確認できません。取引情報を再読み込みしてください。",
      );
      return;
    }

    if (
      trade &&
      currentProposal.refundAmount !== undefined &&
      Number.isInteger(trade.merchandiseRefundMaxAmount) &&
      trade.merchandiseRefundMaxAmount > 0 &&
      currentProposal.refundAmount >
        trade.merchandiseRefundMaxAmount
    ) {
      setError(
        "提示された返金額を確認できません。取引情報を再読み込みしてください。",
      );
      return;
    }

    setSubmittingAction("accept");
    setError("");

    try {
      await acceptTradeReturnProposal({
        tradeId,
        proposalId: currentProposal.id,
      });

      setOpen(false);
      setSubmittingAction(null);
      await reload();
    } catch (caught) {
      setError(
        getErrorMessage(
          caught,
          "返品条件に同意できませんでした。",
        ),
      );
      setSubmittingAction(null);
    }
  }, [
    blocked,
    reload,
    submitting,
    trade,
    tradeId,
  ]);

  const reject = useCallback(async (): Promise<void> => {
    if (submitting) {
      return;
    }

    const currentProposal = getReviewableProposal(
      tradeId,
      trade,
      blocked,
    );

    if (!currentProposal) {
      setError(
        "返品条件を確認できません。取引情報を再読み込みしてください。",
      );
      return;
    }

    setSubmittingAction("reject");
    setError("");

    try {
      await rejectTradeReturnProposal({
        tradeId,
        proposalId: currentProposal.id,
      });

      setOpen(false);
      setSubmittingAction(null);
      await reload();
    } catch (caught) {
      setError(
        getErrorMessage(
          caught,
          "返品条件への回答を送信できませんでした。",
        ),
      );
      setSubmittingAction(null);
    }
  }, [
    blocked,
    reload,
    submitting,
    trade,
    tradeId,
  ]);

  return {
    open,
    proposal,
    error,
    submitting,
    submittingAction,
    accepting,
    rejecting,
    openModal,
    closeModal,
    accept,
    reject,
  };
}

export default useTradeReturnAgreement;