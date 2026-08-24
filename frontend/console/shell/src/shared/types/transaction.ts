// frontend/console/shell/src/shared/types/transaction.ts

export type TransactionType =
  | "receive"
  | "send";

export type Transaction = {
  id: string;
  settlementId: string;
  orderId: string;
  paymentId: string;
  accountId: string;
  type: TransactionType;
  amount: number;
  currency: string;
  description: string;
  status: string;
  stripeTransferId?: string;
  stripeTransferReversalId?: string;
  timestamp: string;
};

export type TransactionManagementRowDTO =
  Transaction;