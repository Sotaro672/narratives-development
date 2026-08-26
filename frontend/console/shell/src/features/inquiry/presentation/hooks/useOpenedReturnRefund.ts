// frontend/console/shell/src/features/inquiry/presentation/hooks/useOpenedReturnRefund.ts

import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import {
  receiveOpenedReturnHTTP,
} from "../../infrastructure/inquiryRepositoryHTTP";

import {
  isOpenedReturnRefundPolicy,
} from "../../../../shared/types/inquiry";

import type {
  OpenedReturnRefundPolicy,
  ReceiveOpenedReturnResult,
} from "../../../../shared/types/inquiry";

export type UseOpenedReturnRefundParams = {
  inquiryId: string;
  onReloadDetail: () => Promise<unknown>;
  onClearPageError: () => void;
};

export type UseOpenedReturnRefundResult = {
  selectedPolicy: OpenedReturnRefundPolicy | "";
  submitting: boolean;
  errorMessage: string | null;
  result: ReceiveOpenedReturnResult | null;

  policyLocked: boolean;
  canSubmit: boolean;

  onChangePolicy: (value: string) => void;
  onSubmit: () => Promise<ReceiveOpenedReturnResult | null>;
  clearErrorMessage: () => void;
};

function normalizeID(
  value: string | null | undefined,
): string {
  return String(value ?? "").trim();
}

function getErrorMessage(
  error: unknown,
  fallbackMessage: string,
): string {
  return error instanceof Error
    ? error.message
    : fallbackMessage;
}

export function useOpenedReturnRefund({
  inquiryId,
  onReloadDetail,
  onClearPageError,
}: UseOpenedReturnRefundParams): UseOpenedReturnRefundResult {
  const [selectedPolicy, setSelectedPolicy] =
    useState<OpenedReturnRefundPolicy | "">("");

  const [submitting, setSubmitting] =
    useState(false);

  const [errorMessage, setErrorMessage] =
    useState<string | null>(null);

  const [result, setResult] =
    useState<ReceiveOpenedReturnResult | null>(null);

  const mountedRef = useRef(false);
  const submittingRef = useRef(false);

  useEffect(() => {
    mountedRef.current = true;

    return () => {
      mountedRef.current = false;
    };
  }, []);

  const policyLocked =
    result !== null;

  const canSubmit =
    normalizeID(inquiryId) !== "" &&
    selectedPolicy !== "" &&
    !submitting &&
    !result?.financiallyCompleted;

  const clearErrorMessage =
    useCallback((): void => {
      setErrorMessage(null);
    }, []);

  const onChangePolicy =
    useCallback(
      (
        value: string,
      ): void => {
        if (
          submittingRef.current ||
          policyLocked
        ) {
          return;
        }

        if (!value) {
          setSelectedPolicy("");
          setErrorMessage(null);
          return;
        }

        if (
          !isOpenedReturnRefundPolicy(
            value,
          )
        ) {
          setErrorMessage(
            "返金方法が不正です。",
          );
          return;
        }

        setSelectedPolicy(value);
        setErrorMessage(null);
      },
      [
        policyLocked,
      ],
    );

  const onSubmit =
    useCallback(
      async (): Promise<ReceiveOpenedReturnResult | null> => {
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

        if (!selectedPolicy) {
          setErrorMessage(
            "返金方法を選択してください。",
          );
          return null;
        }

        if (
          result?.financiallyCompleted
        ) {
          return result;
        }

        submittingRef.current = true;

        if (mountedRef.current) {
          setSubmitting(true);
          setErrorMessage(null);
        }

        onClearPageError();

        try {
          const response =
            await receiveOpenedReturnHTTP(
              normalizedInquiryId,
              {
                policy: selectedPolicy,
              },
            );

          if (!mountedRef.current) {
            return response;
          }

          // Backend が返した Policy を権威値として保持する。
          //
          // 202 pending の場合も同じ Refund が既に作成されているため、
          // 以降の再試行で別 Policy に変更できないようロックする。
          setSelectedPolicy(
            response.policy,
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

          if (
            response.inquiryResolved
          ) {
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
                "開封後返品の返金処理に失敗しました",
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
        inquiryId,
        onClearPageError,
        onReloadDetail,
        result,
        selectedPolicy,
      ],
    );

  return {
    selectedPolicy,
    submitting,
    errorMessage,
    result,

    policyLocked,
    canSubmit,

    onChangePolicy,
    onSubmit,
    clearErrorMessage,
  };
}