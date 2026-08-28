// frontend/amol/src/features/shared/types/payoutAccount.ts

export type PayoutBankAccount = {
  bankName?: string;
  last4?: string;
};

export type PayoutAccount = {
  stripeAccountId: string;
  detailsSubmitted: boolean;
  payoutsEnabled: boolean;
  bankAccount?: PayoutBankAccount | null;
};

export type PayoutAccountResponse = {
  data?: PayoutAccount | null;
  error?: string;
};

export type PayoutAccountSession = {
  clientSecret: string;
};

export type PayoutAccountSessionResponse = {
  data?: PayoutAccountSession;
  error?: string;
};