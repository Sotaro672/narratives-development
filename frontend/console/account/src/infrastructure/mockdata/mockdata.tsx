// frontend/account/src/infrastructure/mockdata/mockdata.tsx
import {
  Account,
  AccountStatus,
  AccountType,
  DEFAULT_CURRENCY, // ← type ではなく value として通常import
} from "../../../../shell/src/shared/types/account";

export const ACCOUNTS: Account[] = [
  {
    id: "account_001",
    memberId: "LUMINA_ADMIN",
    bankName: "三菱UFJ銀行",
    branchName: "渋谷支店",
    accountNumber: 12345678,
    accountType: "普通" as AccountType,
    currency: DEFAULT_CURRENCY,
    status: "active" as AccountStatus,
    createdAt: "2024-05-20T00:00:00Z",
    createdBy: "system",
    updatedAt: "2024-05-20T00:00:00Z",
  },
  {
    id: "account_002",
    memberId: "LUMINA_MANAGER",
    bankName: "みずほ銀行",
    branchName: "新宿支店",
    accountNumber: 87654321,
    accountType: "当座" as AccountType,
    currency: DEFAULT_CURRENCY,
    status: "active" as AccountStatus,
    createdAt: "2024-06-01T00:00:00Z",
    createdBy: "system",
    updatedAt: "2024-06-01T00:00:00Z",
  },
];
