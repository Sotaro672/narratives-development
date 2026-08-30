//frontend\amol\src\features\payout\hooks\usePayoutAccountRegistrationRules.ts
import { useCallback, useEffect, useMemo, useState } from "react";

import { fetchStripeConfig } from "../../payment-method/api/paymentMethodApi";
import {
  getBackendUrl,
  getStripePublishableKey,
} from "../../payment-method/utils/paymentMethodUtils";
import type { PayoutAccountRegistrationDraft } from "../../shared/types/payoutAccount";

export const STRIPE_TEST_BANK_CODE = "1100";
export const STRIPE_TEST_BRANCH_CODE = "000";
export const STRIPE_TEST_ACCOUNT_NUMBER = "0001234";

type StripeMode = "test" | "live";

export type PayoutAccountRegistrationRules = {
  isLoading: boolean;
  isReady: boolean;
  isTestMode: boolean;
  errorMessage: string;
  testBankCode: string;
  testBranchCode: string;
  testAccountNumber: string;
  normalizeAccountNumber: (value: string) => string;
  validateBankCode: (bankCode: string) => string;
  validateBranchCode: (
    bankCode: string,
    branchCode: string,
  ) => string;
  validateAccountNumber: (
    bankCode: string,
    branchCode: string,
    accountNumber: string,
  ) => string;
  isRegistrationInputValid: (
    draft: PayoutAccountRegistrationDraft,
  ) => boolean;
};

function resolveStripeMode(
  publishableKey: string,
): StripeMode | null {
  const normalizedKey = publishableKey.trim();

  if (normalizedKey.startsWith("pk_test_")) {
    return "test";
  }

  if (normalizedKey.startsWith("pk_live_")) {
    return "live";
  }

  return null;
}

function isValidAccountType(
  value: PayoutAccountRegistrationDraft["accountType"],
): boolean {
  return value === "ordinary" || value === "current";
}

export function usePayoutAccountRegistrationRules(): PayoutAccountRegistrationRules {
  const [stripeMode, setStripeMode] = useState<StripeMode | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");

  useEffect(() => {
    let cancelled = false;

    const loadRules = async () => {
      try {
        setIsLoading(true);
        setErrorMessage("");

        const backendUrl = getBackendUrl();

        if (!backendUrl) {
          throw new Error(
            "VITE_API_BASE_URL が設定されていません。",
          );
        }

        const stripeConfig = await fetchStripeConfig(backendUrl);

        if (cancelled) {
          return;
        }

        const publishableKey =
          getStripePublishableKey(stripeConfig);

        if (!publishableKey) {
          throw new Error(
            "Stripe 公開鍵を取得できませんでした。",
          );
        }

        const mode = resolveStripeMode(publishableKey);

        if (!mode) {
          throw new Error(
            "Stripe 公開鍵の環境を判定できませんでした。",
          );
        }

        if (cancelled) {
          return;
        }

        setStripeMode(mode);
      } catch (error) {
        if (cancelled) {
          return;
        }

        console.error(
          "failed to load payout account registration rules:",
          error,
        );

        setStripeMode(null);
        setErrorMessage(
          error instanceof Error
            ? error.message
            : "Stripe設定を確認できませんでした。",
        );
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    void loadRules();

    return () => {
      cancelled = true;
    };
  }, []);

  const isReady =
    !isLoading &&
    stripeMode !== null &&
    !errorMessage;

  const isTestMode = stripeMode === "test";

  const normalizeAccountNumber = useCallback(
    (value: string): string =>
      value.replace(/\D/g, "").slice(0, 7),
    [],
  );

  const validateBankCode = useCallback(
    (bankCodeInput: string): string => {
      const bankCode = bankCodeInput.trim();

      if (!bankCode) {
        return "金融機関を選択してください。";
      }

      if (!/^\d{4}$/.test(bankCode)) {
        return "金融機関コードを確認してください。";
      }

      if (!isReady) {
        return "Stripe設定を確認できませんでした。";
      }

      if (
        isTestMode &&
        bankCode !== STRIPE_TEST_BANK_CODE
      ) {
        return "開発環境ではStripeテスト銀行を選択してください。";
      }

      return "";
    },
    [isReady, isTestMode],
  );

  const validateBranchCode = useCallback(
    (
      bankCodeInput: string,
      branchCodeInput: string,
    ): string => {
      const bankCode = bankCodeInput.trim();
      const branchCode = branchCodeInput.trim();

      if (!branchCode) {
        return "支店を選択してください。";
      }

      if (!/^\d{3}$/.test(branchCode)) {
        return "支店コードを確認してください。";
      }

      if (!isReady) {
        return "Stripe設定を確認できませんでした。";
      }

      if (isTestMode) {
        if (bankCode !== STRIPE_TEST_BANK_CODE) {
          return "開発環境ではStripeテスト銀行を選択してください。";
        }

        if (branchCode !== STRIPE_TEST_BRANCH_CODE) {
          return "開発環境ではStripeテスト支店を選択してください。";
        }
      }

      return "";
    },
    [isReady, isTestMode],
  );

  const validateAccountNumber = useCallback(
    (
      bankCodeInput: string,
      branchCodeInput: string,
      accountNumberInput: string,
    ): string => {
      const bankCode = bankCodeInput.trim();
      const branchCode = branchCodeInput.trim();
      const accountNumber = accountNumberInput.trim();

      if (!accountNumber) {
        return "口座番号を入力してください。";
      }

      if (!/^\d{7}$/.test(accountNumber)) {
        return "口座番号は7桁の数字で入力してください。";
      }

      if (!isReady) {
        return "Stripe設定を確認できませんでした。";
      }

      if (isTestMode) {
        if (bankCode !== STRIPE_TEST_BANK_CODE) {
          return "開発環境ではStripeテスト銀行を選択してください。";
        }

        if (branchCode !== STRIPE_TEST_BRANCH_CODE) {
          return "開発環境ではStripeテスト支店を選択してください。";
        }

        if (
          accountNumber !== STRIPE_TEST_ACCOUNT_NUMBER
        ) {
          return "開発環境ではStripeテスト口座番号 0001234 を使用してください。";
        }
      }

      return "";
    },
    [isReady, isTestMode],
  );

  const isRegistrationInputValid = useCallback(
    (
      draft: PayoutAccountRegistrationDraft,
    ): boolean => {
      if (!isReady) {
        return false;
      }

      const bankCode = draft.bankCode.trim();
      const bankName = draft.bankName.trim();
      const branchCode = draft.branchCode.trim();
      const branchName = draft.branchName.trim();
      const accountNumber = draft.accountNumber.trim();
      const accountHolderName =
        draft.accountHolderName.trim();

      if (!bankName || !branchName || !accountHolderName) {
        return false;
      }

      if (!isValidAccountType(draft.accountType)) {
        return false;
      }

      if (validateBankCode(bankCode)) {
        return false;
      }

      if (validateBranchCode(bankCode, branchCode)) {
        return false;
      }

      if (
        validateAccountNumber(
          bankCode,
          branchCode,
          accountNumber,
        )
      ) {
        return false;
      }

      return true;
    },
    [
      isReady,
      validateAccountNumber,
      validateBankCode,
      validateBranchCode,
    ],
  );

  return useMemo(
    () => ({
      isLoading,
      isReady,
      isTestMode,
      errorMessage,
      testBankCode: STRIPE_TEST_BANK_CODE,
      testBranchCode: STRIPE_TEST_BRANCH_CODE,
      testAccountNumber: STRIPE_TEST_ACCOUNT_NUMBER,
      normalizeAccountNumber,
      validateBankCode,
      validateBranchCode,
      validateAccountNumber,
      isRegistrationInputValid,
    }),
    [
      errorMessage,
      isLoading,
      isReady,
      isTestMode,
      isRegistrationInputValid,
      normalizeAccountNumber,
      validateAccountNumber,
      validateBankCode,
      validateBranchCode,
    ],
  );
}