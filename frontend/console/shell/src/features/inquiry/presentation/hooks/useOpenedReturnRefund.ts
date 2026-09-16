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

import {
  isOpenedReturnRefundPolicy,
} from "../../../../shared/types/inquiry";

import type {
  InquiryType,
  OpenedReturnRefundPolicy,
  ReceiveOpenedReturnResult,
  ReceiveReturnResult,
} from "../../../../shared/types/inquiry";

type ReturnInquiryType = Extract<
  InquiryType,
  "return_unopened" | "return_opened"
>;

type ReturnRefundResult =
  | ReceiveReturnResult
  | ReceiveOpenedReturnResult;

export type UseOpenedReturnRefundParams = {
  inquiryId: string;
  inquiryType: ReturnInquiryType;
  onReloadDetail: () => Promise<unknown>;
  onClearPageError: () => void;
};

export type UseOpenedReturnRefundResult = {
  selectedPolicy: OpenedReturnRefundPolicy | "";
  submitting: boolean;
  errorMessage: string | null;
  result: ReturnRefundResult | null;
  policyLocked: boolean;
  canSubmit: boolean;
  onChangePolicy: (value: string) => void;
  onSubmit: () => Promise<ReturnRefundResult | null>;
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
  inquiryType,
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
    setSelectedPolicy("");
    setErrorMessage(null);
    setResult(null);
    submittingRef.current = false;
    setSubmitting(false);
  }, [
    inquiryId,
    inquiryType,
  ]);

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

        if (!selectedPolicy) {
          setErrorMessage(
            "返金方法を選択してください。",
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
            inquiryType === "return_unopened"
              ? await receiveReturnHTTP(
                  normalizedInquiryId,
                  {
                    policy: selectedPolicy,
                  },
                )
              : await receiveOpenedReturnHTTP(
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
        inquiryId,
        inquiryType,
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