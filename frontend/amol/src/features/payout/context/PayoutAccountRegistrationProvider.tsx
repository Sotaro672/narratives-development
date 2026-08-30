// frontend/amol/src/features/payout/context/PayoutAccountRegistrationProvider.tsx

import * as React from "react";
import { useLocation } from "react-router-dom";

import type {
  PayoutAccountRegistrationDraft,
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

export type PayoutRegistrationReturnTarget = {
  pathname: "/resale";
  state?: unknown;
};

type PayoutAccountRegistrationContextValue = {
  draft: PayoutAccountRegistrationDraft;
  returnAfterRegistration: PayoutRegistrationReturnTarget | null;
  setBank: (bank: PayoutBankSelection) => void;
  setBranch: (branch: PayoutBranchSelection) => void;
  setAccountDetails: (details: PayoutBankAccountDetails) => void;
  resetDraft: () => void;
  isComplete: boolean;
};

const initialDraft: PayoutAccountRegistrationDraft = {
  bankCode: "",
  bankName: "",
  branchCode: "",
  branchName: "",
  accountType: "ordinary",
  accountNumber: "",
  accountHolderName: "",
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function parseReturnAfterRegistration(
  value: unknown,
): PayoutRegistrationReturnTarget | null {
  if (!isRecord(value)) {
    return null;
  }

  const target = value.returnAfterRegistration;

  if (!isRecord(target) || target.pathname !== "/resale") {
    return null;
  }

  return {
    pathname: "/resale",
    state: target.state,
  };
}

const PayoutAccountRegistrationContext =
  React.createContext<PayoutAccountRegistrationContextValue | undefined>(
    undefined,
  );

export const PayoutAccountRegistrationProvider: React.FC<{
  children: React.ReactNode;
}> = ({ children }) => {
  const location = useLocation();

  const [draft, setDraft] =
    React.useState<PayoutAccountRegistrationDraft>(initialDraft);
  const [returnAfterRegistration, setReturnAfterRegistration] =
    React.useState<PayoutRegistrationReturnTarget | null>(() =>
      parseReturnAfterRegistration(location.state),
    );

  React.useEffect(() => {
    const target = parseReturnAfterRegistration(location.state);

    if (target) {
      setReturnAfterRegistration(target);
    }
  }, [location.state]);

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
    [],
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
          draft.accountHolderName.trim(),
      ),
    [draft],
  );

  const contextValue = React.useMemo<PayoutAccountRegistrationContextValue>(
    () => ({
      draft,
      returnAfterRegistration,
      setBank,
      setBranch,
      setAccountDetails,
      resetDraft,
      isComplete,
    }),
    [
      draft,
      returnAfterRegistration,
      setBank,
      setBranch,
      setAccountDetails,
      resetDraft,
      isComplete,
    ],
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
      "usePayoutAccountRegistration must be used within PayoutAccountRegistrationProvider",
    );
  }

  return context;
}