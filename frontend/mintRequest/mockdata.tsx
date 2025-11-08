// frontend/mintRequest/mockdata.tsx

export type MintStatus = "リクエスト済み" | "Mint完了" | "計画中";

export type MintRequestRow = {
  planId: string;
  tokenDesign: string;
  productName: string;
  quantity: number;
  status: MintStatus;
  requester: string;
  requestAt: string; // "YYYY/MM/DD HH:mm:ss"
  executedAt: string; // same or "-"
};

export const ROWS: MintRequestRow[] = [
  {
    planId: "production_002",
    tokenDesign: "NEXUS Street Token",
    productName: "production_002",
    quantity: 5,
    status: "リクエスト済み",
    requester: "佐藤 美咲",
    requestAt: "2025/11/3 11:05:08",
    executedAt: "-",
  },
  {
    planId: "production_003",
    tokenDesign: "LUMINA VIP Token",
    productName: "production_003",
    quantity: 10,
    status: "Mint完了",
    requester: "高橋 健太",
    requestAt: "2025/10/26 11:05:08",
    executedAt: "2025/10/28 11:05:08",
  },
  {
    planId: "production_004",
    tokenDesign: "NEXUS Community Token",
    productName: "production_004",
    quantity: 12,
    status: "Mint完了",
    requester: "山田 太郎",
    requestAt: "2025/10/21 11:05:08",
    executedAt: "2025/10/24 11:05:08",
  },
  {
    planId: "production_001",
    tokenDesign: "SILK Premium Token",
    productName: "production_001",
    quantity: 10,
    status: "計画中",
    requester: "山田 太郎",
    requestAt: "1970/1/1 9:00:00",
    executedAt: "-",
  },
];
