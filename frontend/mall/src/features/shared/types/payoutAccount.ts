// frontend/amol/src/features/shared/types/payoutAccount.ts

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
  bankAccount: PayoutBankAccount;
};

export type PayoutAccountResponse = {
  data?: PayoutAccount | null;
  error?: string;
};

export type PayoutAccountRegistrationDraft = {
  bankCode: string;
  bankName: string;
  branchCode: string;
  branchName: string;
  accountType: PayoutBankAccountType;
  accountNumber: string;
  accountHolderName: string;
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