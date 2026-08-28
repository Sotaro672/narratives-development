// frontend/amol/src/features/shared/types/payoutAccount.ts

export type PayoutAccountStatus = "unregistered" | "pending" | "registered" | "restricted";

export type PayoutBankAccountType = "ordinary" | "current";

export type PayoutBankAccount = {
  bankCode: string;
  bankName: string;
  branchCode: string;
  branchName: string;
  accountType: PayoutBankAccountType;
  last4: string;
  accountHolderName: string;
};

export type PayoutAccount = {
  status: PayoutAccountStatus;
  payoutReady: boolean;
  bankAccount?: PayoutBankAccount | null;
};

export type PayoutAccountResponse = {
  data?: PayoutAccount | null;
  error?: string;
};

export type PayoutAccountRegistrationInput = {
  bankCode: string;
  bankName: string;
  branchCode: string;
  branchName: string;
  accountType: PayoutBankAccountType;
  accountNumber: string;
  accountHolderName: string;
};

export type PayoutAccountRegistrationResponse = {
  data?: PayoutAccount | null;
  error?: string;
};