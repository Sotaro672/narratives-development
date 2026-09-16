// frontend/console/shell/src/features/inquiry/presentation/hooks/useOpenedReturnRefund.ts

import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import {
  receiveOpenedReturnHTTP,
  receiveReturnHTTP,
} from "../../infrastructure/inquiryRepositoryHTTP";

import type {
  InquiryType,
  ReturnRefundResult,
} from "../../../../shared/types/inquiry";

type ReturnInquiryType = Extract<
  InquiryType,
  "return_unopened" | "return_opened"
>;

export type UseOpenedReturnRefundParams = {
  inquiryId: string;
  inquiryType: ReturnInquiryType;
  merchandiseRefundMaxAmount: number;
  onReloadDetail: () => Promise<unknown>;
  onClearPageError: () => void;
};

export type UseOpenedReturnRefundResult = {
  merchandiseRefundAmount: number | "";
  refundOutboundShipping: boolean;
  coverReturnShipping: boolean;
  merchandiseRefundMaxAmount: number;
  submitting: boolean;
  errorMessage: string | null;
  result: ReturnRefundResult | null;
  selectionLocked: boolean;
  canSubmit: boolean;
  onChangeMerchandiseRefundAmount: (value: string | number) => void;
  onChangeRefundOutboundShipping: (value: boolean) => void;
  onChangeCoverReturnShipping: (value: boolean) => void;
  onSubmit: () => Promise<ReturnRefundResult | null>;
  clearErrorMessage: () => void;
};

function normalizeID(
  value: string | null | undefined,
): string {
  return String(value ?? "").trim();
}

function normalizeRefundAmount(
  value: string | number,
): number | "" {
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) {
      return "";
    }

    const normalized = Number(trimmed);
    return Number.isFinite(normalized) ? normalized : "";
  }

  return Number.isFinite(value) ? value : "";
}

function getErrorMessage(
  error: unknown,
  fallbackMessage: string,
): string {
  return error instanceof Error
    ? error.message
    : fallbackMessage;
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
    return "返金可能額を取得できません。問い合わせ詳細を再読み込みしてください。";
  }

  if (amount > maxAmount) {
    return `返金額は商品代金（税込）の上限 ${maxAmount.toLocaleString("ja-JP")}円 以下で入力してください。`;
  }

  return null;
}

export function useOpenedReturnRefund({
  inquiryId,
  inquiryType,
  merchandiseRefundMaxAmount,
  onReloadDetail,
  onClearPageError,
}: UseOpenedReturnRefundParams): UseOpenedReturnRefundResult {
  const [merchandiseRefundAmount, setMerchandiseRefundAmount] =
    useState<number | "">("");
  const [refundOutboundShipping, setRefundOutboundShipping] =
    useState(false);
  const [coverReturnShipping, setCoverReturnShipping] =
    useState(false);
  const [submitting, setSubmitting] =
    useState(false);
  const [errorMessage, setErrorMessage] =
    useState<string | null>(null);
  const [result, setResult] =
    useState<ReturnRefundResult | null>(null);

  const mountedRef = useRef(false);
  const submittingRef = useRef(false);

  useEffect(() => {
    mountedRef.current = true;

    return () => {
      mountedRef.current = false;
    };
  }, []);

  useEffect(() => {
    setMerchandiseRefundAmount("");
    setRefundOutboundShipping(false);
    setCoverReturnShipping(false);
    setErrorMessage(null);
    setResult(null);
    submittingRef.current = false;
    setSubmitting(false);
  }, [
    inquiryId,
    inquiryType,
  ]);

  const selectionLocked =
    result !== null;

  const amountValidationError =
    validateMerchandiseRefundAmount(
      merchandiseRefundAmount,
      merchandiseRefundMaxAmount,
    );

  const canSubmit =
    normalizeID(inquiryId) !== "" &&
    amountValidationError === null &&
    !submitting &&
    !result?.financiallyCompleted;

  const clearErrorMessage =
    useCallback((): void => {
      setErrorMessage(null);
    }, []);

  const onChangeMerchandiseRefundAmount =
    useCallback(
      (
        value: string | number,
      ): void => {
        if (
          submittingRef.current ||
          selectionLocked
        ) {
          return;
        }

        const normalized =
          normalizeRefundAmount(value);

        setMerchandiseRefundAmount(normalized);

        if (normalized === "") {
          setErrorMessage(null);
          return;
        }

        const validationError =
          validateMerchandiseRefundAmount(
            normalized,
            merchandiseRefundMaxAmount,
          );

        setErrorMessage(validationError);
      },
      [
        merchandiseRefundMaxAmount,
        selectionLocked,
      ],
    );

  const onChangeRefundOutboundShipping =
    useCallback(
      (
        value: boolean,
      ): void => {
        if (
          submittingRef.current ||
          selectionLocked
        ) {
          return;
        }

        setRefundOutboundShipping(value);
        setErrorMessage(null);
      },
      [
        selectionLocked,
      ],
    );

  const onChangeCoverReturnShipping =
    useCallback(
      (
        value: boolean,
      ): void => {
        if (
          submittingRef.current ||
          selectionLocked
        ) {
          return;
        }

        setCoverReturnShipping(value);
        setErrorMessage(null);
      },
      [
        selectionLocked,
      ],
    );

  const onSubmit =
    useCallback(
      async (): Promise<ReturnRefundResult | null> => {
        if (submittingRef.current) {
          return null;
        }

        const normalizedInquiryId =
          normalizeID(inquiryId);

        if (!normalizedInquiryId) {
          setErrorMessage(
            "問い合わせIDが指定されていません。",
          );
          return null;
        }

        if (
          inquiryType !== "return_unopened" &&
          inquiryType !== "return_opened"
        ) {
          setErrorMessage(
            "返品種別が不正です。",
          );
          return null;
        }

        if (merchandiseRefundAmount === "") {
          setErrorMessage(
            "返金額を入力してください。",
          );
          return null;
        }

        const validationError =
          validateMerchandiseRefundAmount(
            merchandiseRefundAmount,
            merchandiseRefundMaxAmount,
          );

        if (validationError) {
          setErrorMessage(validationError);
          return null;
        }

        if (result?.financiallyCompleted) {
          return result;
        }

        const selection = {
          merchandiseRefundAmount,
          refundOutboundShipping,
          coverReturnShipping,
        };

        submittingRef.current = true;

        if (mountedRef.current) {
          setSubmitting(true);
          setErrorMessage(null);
        }

        onClearPageError();

        try {
          const response =
            inquiryType === "return_unopened"
              ? await receiveReturnHTTP(
                  normalizedInquiryId,
                  selection,
                )
              : await receiveOpenedReturnHTTP(
                  normalizedInquiryId,
                  selection,
                );

          if (!mountedRef.current) {
            return response;
          }

          // Backend が返した値を権威値として保持する。
          //
          // 202 pending の場合でも Refund は既に同じ Selection で作成されているため、
          // 以降の再試行で返金額・往路送料・復路送料を変更できないようロックする。
          setMerchandiseRefundAmount(
            response.merchandiseRefundAmount,
          );
          setRefundOutboundShipping(
            response.refundOutboundShipping,
          );
          setCoverReturnShipping(
            response.coverReturnShipping,
          );
          setResult(response);

          try {
            await onReloadDetail();
          } catch (
            reloadError: unknown
          ) {
            if (mountedRef.current) {
              setErrorMessage(
                getErrorMessage(
                  reloadError,
                  "返金処理後の問い合わせ詳細の再取得に失敗しました",
                ),
              );
            }
          }

          if (response.inquiryResolved) {
            window.dispatchEvent(
              new Event(
                "inquiry:status-changed",
              ),
            );
          }

          return response;
        } catch (
          error: unknown
        ) {
          if (mountedRef.current) {
            setErrorMessage(
              getErrorMessage(
                error,
                "返品の返金処理に失敗しました",
              ),
            );
          }

          return null;
        } finally {
          submittingRef.current = false;

          if (mountedRef.current) {
            setSubmitting(false);
          }
        }
      },
      [
        coverReturnShipping,
        inquiryId,
        inquiryType,
        merchandiseRefundAmount,
        merchandiseRefundMaxAmount,
        onClearPageError,
        onReloadDetail,
        refundOutboundShipping,
        result,
      ],
    );

  return {
    merchandiseRefundAmount,
    refundOutboundShipping,
    coverReturnShipping,
    merchandiseRefundMaxAmount,
    submitting,
    errorMessage,
    result,
    selectionLocked,
    canSubmit,
    onChangeMerchandiseRefundAmount,
    onChangeRefundOutboundShipping,
    onChangeCoverReturnShipping,
    onSubmit,
    clearErrorMessage,
  };
}