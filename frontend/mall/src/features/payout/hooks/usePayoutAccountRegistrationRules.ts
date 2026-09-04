// frontend/amol/src/features/payout/hooks/usePayoutAccountRegistrationRules.ts

import { useCallback, useMemo } from "react";

import type { PayoutAccountRegistrationDraft } from "../../shared/types/payoutAccount";

export type PayoutAccountRegistrationRules = {
  validateBankCode: (bankCode: string) => string;
  validateBranchCode: (branchCode: string) => string;
  validateAccountNumber: (accountNumber: string) => string;
  isRegistrationInputValid: (draft: PayoutAccountRegistrationDraft) => boolean;
};

function hasNonWhitespace(value: string): boolean {
  return /\S/.test(value);
}

function isValidAccountType(
  value: PayoutAccountRegistrationDraft["accountType"],
): boolean {
  return value === "ordinary" || value === "current";
}

export function usePayoutAccountRegistrationRules(): PayoutAccountRegistrationRules {
  const validateBankCode = useCallback((bankCode: string): string => {
    if (!bankCode) {
      return "金融機関を選択してください。";
    }

    if (!/^\d{4}$/.test(bankCode)) {
      return "金融機関コードを確認してください。";
    }

    return "";
  }, []);

  const validateBranchCode = useCallback((branchCode: string): string => {
    if (!branchCode) {
      return "支店を選択してください。";
    }

    if (!/^\d{3}$/.test(branchCode)) {
      return "支店コードを確認してください。";
    }

    return "";
  }, []);

  const validateAccountNumber = useCallback((accountNumber: string): string => {
    if (!accountNumber) {
      return "口座番号を入力してください。";
    }

    if (!/^\d{7}$/.test(accountNumber)) {
      return "口座番号は7桁の数字で入力してください。";
    }

    return "";
  }, []);

  const isRegistrationInputValid = useCallback(
    (draft: PayoutAccountRegistrationDraft): boolean => {
      if (!hasNonWhitespace(draft.bankName) || !hasNonWhitespace(draft.branchName)) {
        return false;
      }

      if (!isValidAccountType(draft.accountType)) {
        return false;
      }

      if (!hasNonWhitespace(draft.accountHolderName)) {
        return false;
      }

      if (validateBankCode(draft.bankCode)) {
        return false;
      }

      if (validateBranchCode(draft.branchCode)) {
        return false;
      }

      if (validateAccountNumber(draft.accountNumber)) {
        return false;
      }

      return true;
    },
    [validateAccountNumber, validateBankCode, validateBranchCode],
  );

  return useMemo(
    () => ({
      validateBankCode,
      validateBranchCode,
      validateAccountNumber,
      isRegistrationInputValid,
    }),
    [
      isRegistrationInputValid,
      validateAccountNumber,
      validateBankCode,
      validateBranchCode,
    ],
  );
}