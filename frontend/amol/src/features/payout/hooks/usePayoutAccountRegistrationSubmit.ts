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
import type {
  PayoutAccount,
  PayoutAccountRegistrationDraft,
  PayoutBankAccountType,
} from "../../shared/types/payoutAccount";

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

  if (!/^\d+$/.test(draft.accountNumber)) {
    throw new Error("口座番号は数字で入力してください。");
  }

  if (!draft.accountHolderName) {
    throw new Error("口座名義を確認してください。");
  }
}

function getStripeErrorMessage(
  error: { message?: string } | null | undefined,
): string {
  const message = error?.message?.trim();

  return message || "銀行口座情報をStripeへ登録できませんでした。";
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
          throw new Error(getStripeErrorMessage(tokenResult.error));
        }

        const bankAccountToken = tokenResult.token?.id?.trim() || "";

        if (!bankAccountToken) {
          throw new Error(
            "Stripeの銀行口座トークンを取得できませんでした。",
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