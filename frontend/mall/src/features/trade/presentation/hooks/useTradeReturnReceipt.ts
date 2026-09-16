// frontend/mall/src/features/trade/presentation/hooks/useTradeReturnReceipt.ts

import {
  useCallback,
  useEffect,
  useState,
} from "react";

import type {
  ReceiveTradeReturnResult,
  TradeDetail,
} from "../../../shared/types/trade";
import { receiveTradeReturn } from "../../infrastructure/tradeApi";
import { getErrorMessage } from "../util/tradeChatDetail";

type UseTradeReturnReceiptInput = {
  tradeId: string;
  trade: TradeDetail | null;
  reload: () => Promise<void>;
  blocked: boolean;
};

function normalizeRefundAmount(
  value: string | number,
): number | "" {
  if (typeof value === "string") {
    const trimmed = value.trim();

    if (!trimmed) {
      return "";
    }

    const normalized = Number(trimmed);
    return Number.isFinite(normalized)
      ? normalized
      : "";
  }

  return Number.isFinite(value)
    ? value
    : "";
}

function validateMerchandiseRefundAmount(
  amount: number | "",
  maxAmount: number,
): string | null {
  if (amount === "") {
    return "返金額を入力してください。";
  }

  if (!Number.isInteger(amount)) {
    return "返金額は1円単位の整数で入力してください。";
  }

  if (amount <= 0) {
    return "返金額は1円以上で入力してください。";
  }

  if (
    !Number.isInteger(maxAmount) ||
    maxAmount <= 0
  ) {
    return "返金可能額を取得できません。取引情報を再読み込みしてください。";
  }

  if (amount > maxAmount) {
    return `返金額は商品代金（税込）の上限 ${maxAmount.toLocaleString("ja-JP")}円 以下で入力してください。`;
  }

  return null;
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
    trade.isReturnRequested &&
    !trade.isReturnCompleted
  );
}

export function useTradeReturnReceipt({
  tradeId,
  trade,
  reload,
  blocked,
}: UseTradeReturnReceiptInput) {
  const [open, setOpen] = useState(false);
  const [
    merchandiseRefundAmount,
    setMerchandiseRefundAmountState,
  ] = useState<number | "">("");
  const [
    refundOutboundShipping,
    setRefundOutboundShippingState,
  ] = useState(false);
  const [
    coverReturnShipping,
    setCoverReturnShippingState,
  ] = useState(false);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] =
    useState<ReceiveTradeReturnResult | null>(null);

  const merchandiseRefundMaxAmount =
    trade?.merchandiseRefundMaxAmount ?? 0;

  const selectionLocked =
    result !== null;

  const amountValidationError =
    validateMerchandiseRefundAmount(
      merchandiseRefundAmount,
      merchandiseRefundMaxAmount,
    );

  const canSubmit =
    canReceiveReturn(
      tradeId,
      trade,
      blocked,
    ) &&
    amountValidationError === null &&
    !submitting &&
    !result?.financiallyCompleted;

  useEffect(() => {
    setOpen(false);
    setMerchandiseRefundAmountState("");
    setRefundOutboundShippingState(false);
    setCoverReturnShippingState(false);
    setError("");
    setSubmitting(false);
    setResult(null);
  }, [tradeId]);

  const setMerchandiseRefundAmount =
    useCallback(
      (value: string | number): void => {
        if (submitting || selectionLocked) {
          return;
        }

        const normalized =
          normalizeRefundAmount(value);

        setMerchandiseRefundAmountState(
          normalized,
        );

        if (normalized === "") {
          setError("");
          return;
        }

        setError(
          validateMerchandiseRefundAmount(
            normalized,
            merchandiseRefundMaxAmount,
          ) ?? "",
        );
      },
      [
        merchandiseRefundMaxAmount,
        selectionLocked,
        submitting,
      ],
    );

  const setRefundOutboundShipping =
    useCallback(
      (value: boolean): void => {
        if (submitting || selectionLocked) {
          return;
        }

        setRefundOutboundShippingState(value);
        setError("");
      },
      [
        selectionLocked,
        submitting,
      ],
    );

  const setCoverReturnShipping =
    useCallback(
      (value: boolean): void => {
        if (submitting || selectionLocked) {
          return;
        }

        setCoverReturnShippingState(value);
        setError("");
      },
      [
        selectionLocked,
        submitting,
      ],
    );

  const openModal = useCallback(() => {
    if (
      !canReceiveReturn(
        tradeId,
        trade,
        blocked,
      ) ||
      result?.financiallyCompleted
    ) {
      return;
    }

    if (!selectionLocked) {
      setMerchandiseRefundAmountState("");
      setRefundOutboundShippingState(false);
      setCoverReturnShippingState(false);
    }

    setError("");
    setOpen(true);
  }, [
    blocked,
    result,
    selectionLocked,
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

        if (merchandiseRefundAmount === "") {
          setError("返金額を入力してください。");
          return null;
        }

        const validationError =
          validateMerchandiseRefundAmount(
            merchandiseRefundAmount,
            merchandiseRefundMaxAmount,
          );

        if (validationError) {
          setError(validationError);
          return null;
        }

        if (result?.financiallyCompleted) {
          return result;
        }

        setSubmitting(true);
        setError("");

        try {
          const response =
            await receiveTradeReturn({
              tradeId,
              merchandiseRefundAmount,
              refundOutboundShipping,
              coverReturnShipping,
            });

          // 202相当で金融処理が未完了の場合でも、RefundはこのSelectionで
          // 作成済みとなるため、以後の再試行で条件を変更できないようにする。
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

          if (response.financiallyCompleted) {
            setOpen(false);
            setError("");
          } else {
            setError(
              "返金処理を受け付けました。金融処理が完了していないため、同じ返金条件で再実行できます。",
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
        coverReturnShipping,
        merchandiseRefundAmount,
        merchandiseRefundMaxAmount,
        refundOutboundShipping,
        reload,
        result,
        submitting,
        trade,
        tradeId,
      ],
    );

  return {
    open,
    merchandiseRefundAmount,
    merchandiseRefundMaxAmount,
    refundOutboundShipping,
    coverReturnShipping,
    error,
    submitting,
    result,
    selectionLocked,
    canSubmit,
    openModal,
    closeModal,
    setMerchandiseRefundAmount,
    setRefundOutboundShipping,
    setCoverReturnShipping,
    submit,
  };
}

export default useTradeReturnReceipt;