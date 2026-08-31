// frontend/amol/src/features/payout/hooks/usePayoutAccountRegistrationSubmit.ts

import { useCallback, useState } from "react";
import { getAuth } from "firebase/auth";

import { registerPayoutAccount } from "../api/payoutApi";
import type {
  PayoutAccount,
  PayoutAccountRegistrationDraft,
} from "../../shared/types/payoutAccount";

function hasNonWhitespace(value: string): boolean {
  return /\S/.test(value);
}

function validateDraft(draft: PayoutAccountRegistrationDraft): void {
  if (!/^\d{4}$/.test(draft.bankCode) || !hasNonWhitespace(draft.bankName)) {
    throw new Error("金融機関を確認してください。");
  }

  if (!/^\d{3}$/.test(draft.branchCode) || !hasNonWhitespace(draft.branchName)) {
    throw new Error("支店を確認してください。");
  }

  if (draft.accountType !== "ordinary" && draft.accountType !== "current") {
    throw new Error("口座種別を確認してください。");
  }

  if (!/^\d{7}$/.test(draft.accountNumber)) {
    throw new Error("口座番号は7桁の数字で入力してください。");
  }

  if (!hasNonWhitespace(draft.accountHolderName)) {
    throw new Error("口座名義を確認してください。");
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
      draft: PayoutAccountRegistrationDraft,
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

        validateDraft(draft);

        const idToken = await currentUser.getIdToken(true);

        return await registerPayoutAccount({
          idToken,
          input: {
            bankCode: draft.bankCode,
            bankName: draft.bankName,
            branchCode: draft.branchCode,
            branchName: draft.branchName,
            accountType: draft.accountType,
            accountNumber: draft.accountNumber,
            accountHolderName: draft.accountHolderName,
          },
        });
      } catch (error) {
        console.error("failed to register payout account:", error);

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