// frontend/list/mockdata.tsx

export type ListingRow = {
  id: string;
  product: string;
  brand: string;
  token: string;
  stock: number;
  manager: string;
  status: "出品中" | "停止中";
};

export const LISTINGS: ListingRow[] = [
  {
    id: "list_001",
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    token: "LUMINA VIP Token",
    stock: 221,
    manager: "山田 太郎",
    status: "出品中",
  },
  {
    id: "list_002",
    product: "デニムジャケット ヴィンテージ加工",
    brand: "NEXUS Street",
    token: "NEXUS Community Token",
    stock: 222,
    manager: "佐藤 美咲",
    status: "出品中",
  },
  {
    id: "list_003",
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    token: "LUMINA VIP Token",
    stock: 221,
    manager: "山田 太郎",
    status: "停止中",
  },
];
