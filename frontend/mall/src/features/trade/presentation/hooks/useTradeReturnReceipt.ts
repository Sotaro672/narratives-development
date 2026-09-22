// frontend/mall/src/features/trade/presentation/hooks/useTradeReturnReceipt.ts

import {
  useCallback,
  useEffect,
  useState,
} from "react";

import type {
  ReceiveTradeReturnResult,
  TradeDetail,
  TradeReturnProposal,
} from "../../../shared/types/trade";
import { receiveTradeReturn } from "../../infrastructure/tradeApi";
import { getErrorMessage } from "../util/tradeChatDetail";

type UseTradeReturnReceiptInput = {
  tradeId: string;
  trade: TradeDetail | null;
  reload: () => Promise<void>;
  blocked: boolean;
};

function getAcceptedPhysicalReturnProposal(
  trade: TradeDetail | null,
): TradeReturnProposal | null {
  const proposal = trade?.returnProposal;

  if (
    !proposal ||
    proposal.id.trim() === "" ||
    proposal.agreement !== "agree" ||
    proposal.rejectedAt ||
    proposal.returnRequirement !== "required" ||
    proposal.refundAmount === undefined ||
    !Number.isInteger(proposal.refundAmount) ||
    proposal.refundAmount <= 0
  ) {
    return null;
  }

  return proposal;
}

function isReceivableReturnStatus(
  trade: TradeDetail,
): boolean {
  switch (trade.returnStatus) {
    case "agreed":
    case "return_shipped":
    case "return_received":
    case "refund_processing":
      return true;

    case "none":
    case "discussing":
    case "proposed":
    case "completed":
    case "disputed":
    case undefined:
      return false;
  }
}

function canReceiveReturn(
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
    isReceivableReturnStatus(trade) &&
    getAcceptedPhysicalReturnProposal(trade) !== null
  );
}

export function useTradeReturnReceipt({
  tradeId,
  trade,
  reload,
  blocked,
}: UseTradeReturnReceiptInput) {
  const [open, setOpen] = useState(false);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] =
    useState<ReceiveTradeReturnResult | null>(null);

  const proposal =
    getAcceptedPhysicalReturnProposal(trade);

  const refundAmount =
    proposal?.refundAmount ?? 0;

  const canSubmit =
    canReceiveReturn(
      tradeId,
      trade,
      blocked,
    ) &&
    !submitting &&
    !result?.returnCompleted;

  useEffect(() => {
    setOpen(false);
    setError("");
    setSubmitting(false);
    setResult(null);
  }, [tradeId]);

  const openModal = useCallback(() => {
    if (
      !canReceiveReturn(
        tradeId,
        trade,
        blocked,
      ) ||
      result?.returnCompleted
    ) {
      return;
    }

    setError("");
    setOpen(true);
  }, [
    blocked,
    result,
    trade,
    tradeId,
  ]);

  const closeModal = useCallback(() => {
    if (submitting) {
      return;
    }

    setOpen(false);
    setError("");
  }, [submitting]);

  const submit =
    useCallback(
      async (): Promise<ReceiveTradeReturnResult | null> => {
        if (
          submitting ||
          !canReceiveReturn(
            tradeId,
            trade,
            blocked,
          )
        ) {
          return null;
        }

        if (result?.returnCompleted) {
          return result;
        }

        setSubmitting(true);
        setError("");

        try {
          const response =
            await receiveTradeReturn({
              tradeId,
            });

          setResult(response);

          try {
            await reload();
          } catch (reloadError) {
            setError(
              getErrorMessage(
                reloadError,
                "返品処理後の取引情報の再取得に失敗しました。",
              ),
            );

            return response;
          }

          if (
            response.financiallyCompleted &&
            response.returnCompleted
          ) {
            setOpen(false);
            setError("");
          } else {
            setError(
              "返金処理を受け付けました。金融処理が完了していないため、再実行できます。",
            );
          }

          return response;
        } catch (caught) {
          setError(
            getErrorMessage(
              caught,
              "返品の受領・返金処理に失敗しました。",
            ),
          );

          return null;
        } finally {
          setSubmitting(false);
        }
      },
      [
        blocked,
        reload,
        result,
        submitting,
        trade,
        tradeId,
      ],
    );

  return {
    open,
    proposal,
    refundAmount,
    error,
    submitting,
    result,
    canSubmit,
    openModal,
    closeModal,
    submit,
  };
}

export default useTradeReturnReceipt;