// frontend/amol/src/features/payout/context/PayoutAccountRegistrationProvider.tsx

import * as React from "react";

import type {
  PayoutAccountRegistrationInput,
  PayoutBankAccountType,
} from "../../shared/types/payoutAccount";

type PayoutBankSelection = {
  bankCode: string;
  bankName: string;
};

type PayoutBranchSelection = {
  branchCode: string;
  branchName: string;
};

type PayoutBankAccountDetails = {
  accountType: PayoutBankAccountType;
  accountNumber: string;
  accountHolderName: string;
};

type PayoutAccountRegistrationContextValue = {
  draft: PayoutAccountRegistrationInput;
  setBank: (bank: PayoutBankSelection) => void;
  setBranch: (branch: PayoutBranchSelection) => void;
  setAccountDetails: (details: PayoutBankAccountDetails) => void;
  resetDraft: () => void;
  isComplete: boolean;
};

const initialDraft: PayoutAccountRegistrationInput = {
  bankCode: "",
  bankName: "",
  branchCode: "",
  branchName: "",
  accountType: "ordinary",
  accountNumber: "",
  accountHolderName: "",
};

const PayoutAccountRegistrationContext =
  React.createContext<PayoutAccountRegistrationContextValue | undefined>(
    undefined
  );

export const PayoutAccountRegistrationProvider: React.FC<{
  children: React.ReactNode;
}> = ({ children }) => {
  const [draft, setDraft] =
    React.useState<PayoutAccountRegistrationInput>(initialDraft);

  const setBank = React.useCallback((bank: PayoutBankSelection) => {
    setDraft((current) => {
      const bankChanged = current.bankCode !== bank.bankCode;

      return {
        ...current,
        bankCode: bank.bankCode,
        bankName: bank.bankName,
        branchCode: bankChanged ? "" : current.branchCode,
        branchName: bankChanged ? "" : current.branchName,
      };
    });
  }, []);

  const setBranch = React.useCallback((branch: PayoutBranchSelection) => {
    setDraft((current) => ({
      ...current,
      branchCode: branch.branchCode,
      branchName: branch.branchName,
    }));
  }, []);

  const setAccountDetails = React.useCallback(
    (details: PayoutBankAccountDetails) => {
      setDraft((current) => ({
        ...current,
        accountType: details.accountType,
        accountNumber: details.accountNumber,
        accountHolderName: details.accountHolderName,
      }));
    },
    []
  );

  const resetDraft = React.useCallback(() => {
    setDraft(initialDraft);
  }, []);

  const isComplete = React.useMemo(
    () =>
      Boolean(
        draft.bankCode.trim() &&
          draft.bankName.trim() &&
          draft.branchCode.trim() &&
          draft.branchName.trim() &&
          draft.accountNumber.trim() &&
          draft.accountHolderName.trim()
      ),
    [draft]
  );

  const contextValue = React.useMemo<PayoutAccountRegistrationContextValue>(
    () => ({
      draft,
      setBank,
      setBranch,
      setAccountDetails,
      resetDraft,
      isComplete,
    }),
    [draft, setBank, setBranch, setAccountDetails, resetDraft, isComplete]
  );

  return (
    <PayoutAccountRegistrationContext.Provider value={contextValue}>
      {children}
    </PayoutAccountRegistrationContext.Provider>
  );
};

export function usePayoutAccountRegistration(): PayoutAccountRegistrationContextValue {
  const context = React.useContext(PayoutAccountRegistrationContext);

  if (!context) {
    throw new Error(
      "usePayoutAccountRegistration must be used within PayoutAccountRegistrationProvider"
    );
  }

  return context;
}