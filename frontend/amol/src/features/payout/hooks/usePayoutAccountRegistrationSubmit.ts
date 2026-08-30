// frontend/amol/src/features/payout/hooks/usePayoutAccountRegistrationSubmit.ts

import { useCallback, useState } from "react";
import { getAuth } from "firebase/auth";
import { loadStripe } from "@stripe/stripe-js";
import type { CreateTokenBankAccountData } from "@stripe/stripe-js";

import { fetchStripeConfig } from "../../payment-method/api/paymentMethodApi";
import {
  getBackendUrl,
  getStripePublishableKey,
} from "../../payment-method/utils/paymentMethodUtils";
import { registerPayoutAccount } from "../api/payoutApi";
import {
  STRIPE_TEST_ACCOUNT_NUMBER,
  STRIPE_TEST_BANK_CODE,
  STRIPE_TEST_BRANCH_CODE,
} from "./usePayoutAccountRegistrationRules";
import type {
  PayoutAccount,
  PayoutAccountRegistrationDraft,
  PayoutBankAccountType,
} from "../../shared/types/payoutAccount";

type StripeMode = "test" | "live";

type StripeTokenError = {
  type?: string;
  code?: string;
  message?: string;
};

function getStripeAccountType(
  accountType: PayoutBankAccountType,
): "futsu" | "toza" {
  switch (accountType) {
    case "ordinary":
      return "futsu";
    case "current":
      return "toza";
    default:
      throw new Error("口座種別を確認してください。");
  }
}

function normalizeDraft(
  draft: PayoutAccountRegistrationDraft,
): PayoutAccountRegistrationDraft {
  return {
    bankCode: draft.bankCode.trim(),
    bankName: draft.bankName.trim(),
    branchCode: draft.branchCode.trim(),
    branchName: draft.branchName.trim(),
    accountType: draft.accountType,
    accountNumber: draft.accountNumber.trim(),
    accountHolderName: draft.accountHolderName.trim(),
  };
}

function validateDraft(draft: PayoutAccountRegistrationDraft): void {
  if (!draft.bankCode || !draft.bankName) {
    throw new Error("金融機関を確認してください。");
  }

  if (!/^\d{4}$/.test(draft.bankCode)) {
    throw new Error("金融機関コードを確認してください。");
  }

  if (!draft.branchCode || !draft.branchName) {
    throw new Error("支店を確認してください。");
  }

  if (!/^\d{3}$/.test(draft.branchCode)) {
    throw new Error("支店コードを確認してください。");
  }

  if (!draft.accountNumber) {
    throw new Error("口座番号を確認してください。");
  }

  if (!/^\d{7}$/.test(draft.accountNumber)) {
    throw new Error("口座番号は7桁の数字で入力してください。");
  }

  if (!draft.accountHolderName) {
    throw new Error("口座名義を確認してください。");
  }
}

function resolveStripeMode(publishableKey: string): StripeMode {
  const normalizedKey = publishableKey.trim();

  if (normalizedKey.startsWith("pk_test_")) {
    return "test";
  }

  if (normalizedKey.startsWith("pk_live_")) {
    return "live";
  }

  throw new Error("Stripe 公開鍵の環境を判定できませんでした。");
}

function validateDraftForStripeMode(
  draft: PayoutAccountRegistrationDraft,
  stripeMode: StripeMode,
): void {
  if (stripeMode !== "test") {
    return;
  }

  if (draft.bankCode !== STRIPE_TEST_BANK_CODE) {
    throw new Error(
      `開発環境では金融機関コード ${STRIPE_TEST_BANK_CODE} のStripeテスト銀行を使用してください。`,
    );
  }

  if (draft.branchCode !== STRIPE_TEST_BRANCH_CODE) {
    throw new Error(
      `開発環境では支店コード ${STRIPE_TEST_BRANCH_CODE} のStripeテスト支店を使用してください。`,
    );
  }

  if (draft.accountNumber !== STRIPE_TEST_ACCOUNT_NUMBER) {
    throw new Error(
      `開発環境ではStripeテスト口座番号 ${STRIPE_TEST_ACCOUNT_NUMBER} を使用してください。`,
    );
  }
}

function isValidBankAccountToken(value: string): boolean {
  if (!value || /\s/.test(value)) {
    return false;
  }

  return value.startsWith("btok_") || value.startsWith("tok_");
}

function getStripeErrorMessage(
  error: StripeTokenError | null | undefined,
): string {
  const code = error?.code?.trim();

  switch (code) {
    case "routing_number_invalid":
      return "金融機関コードまたは支店コードを確認してください。";
    case "account_number_invalid":
      return "口座番号を確認してください。";
    default:
      return "銀行口座情報をStripeへ登録できませんでした。入力内容を確認してください。";
  }
}

export function usePayoutAccountRegistrationSubmit() {
  const auth = getAuth();

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const clearError = useCallback(() => {
    setErrorMessage("");
  }, []);

  const submitPayoutAccountRegistration = useCallback(
    async (
      draftInput: PayoutAccountRegistrationDraft,
    ): Promise<PayoutAccount> => {
      const currentUser = auth.currentUser;

      if (!currentUser) {
        const error = new Error("ログイン情報を確認できませんでした。");
        setErrorMessage(error.message);
        throw error;
      }

      try {
        setIsSubmitting(true);
        setErrorMessage("");

        const draft = normalizeDraft(draftInput);
        validateDraft(draft);

        const backendUrl = getBackendUrl();

        if (!backendUrl) {
          throw new Error("VITE_API_BASE_URL が設定されていません。");
        }

        const stripeConfig = await fetchStripeConfig(backendUrl);
        const publishableKey = getStripePublishableKey(stripeConfig);

        if (!publishableKey) {
          throw new Error("Stripe 公開鍵を取得できませんでした。");
        }

        const stripeMode = resolveStripeMode(publishableKey);
        validateDraftForStripeMode(draft, stripeMode);

        const stripe = await loadStripe(publishableKey);

        if (!stripe) {
          throw new Error("Stripeを初期化できませんでした。");
        }

        const bankAccountData: CreateTokenBankAccountData = {
          country: "JP",
          currency: "jpy",
          routing_number: `${draft.bankCode}${draft.branchCode}`,
          account_number: draft.accountNumber,
          account_holder_name: draft.accountHolderName,
          account_holder_type: "individual",
          account_type: getStripeAccountType(draft.accountType),
        };

        const tokenResult = await stripe.createToken(
          "bank_account",
          bankAccountData,
        );

        if (tokenResult.error) {
          console.error("Stripe bank account tokenization failed:", {
            type: tokenResult.error.type,
            code: tokenResult.error.code,
          });

          throw new Error(getStripeErrorMessage(tokenResult.error));
        }

        const bankAccountToken = tokenResult.token?.id?.trim() || "";

        if (!bankAccountToken) {
          throw new Error(
            "Stripeの銀行口座トークンを取得できませんでした。",
          );
        }

        if (!isValidBankAccountToken(bankAccountToken)) {
          console.error(
            "Stripe returned an unsupported bank account token format.",
          );

          throw new Error(
            "Stripeの銀行口座トークン形式を確認できませんでした。",
          );
        }

        const idToken = await currentUser.getIdToken(true);

        return await registerPayoutAccount({
          idToken,
          input: {
            bankAccountToken,
          },
        });
      } catch (error) {
        console.error(
          "failed to register payout account:",
          error,
        );

        const message =
          error instanceof Error
            ? error.message
            : "売上受取口座の登録に失敗しました。";

        setErrorMessage(message);

        if (error instanceof Error) {
          throw error;
        }

        throw new Error(message);
      } finally {
        setIsSubmitting(false);
      }
    },
    [auth],
  );

  return {
    isSubmitting,
    errorMessage,
    clearError,
    submitPayoutAccountRegistration,
  };
}