// frontend/account/mockdata.tsx

export type Account = {
  id: string;
  name: string;
  email: string;
  role: string;
  brand: string;
  createdAt: string;
};

export const ACCOUNTS: Account[] = [
  {
    id: "acc_001",
    name: "山田 太郎",
    email: "admin@narratives.com",
    role: "管理者",
    brand: "LUMINA Fashion",
    createdAt: "2024/05/20",
  },
  {
    id: "acc_002",
    name: "佐藤 美咲",
    email: "manager.lumina@narratives.com",
    role: "ブランド管理者",
    brand: "LUMINA Fashion",
    createdAt: "2024/06/01",
  },
];
