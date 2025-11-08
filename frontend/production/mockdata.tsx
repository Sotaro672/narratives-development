// frontend/production/mockdata.tsx

export type Production = {
  id: string;
  product: string;
  brand: string;
  manager: string;
  quantity: number;
  productId: string;
  printedAt: string; // YYYY/M/D or "-"
  createdAt: string; // YYYY/M/D
};

export const PRODUCTIONS: Production[] = [
  {
    id: "production_001",
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    manager: "佐藤 美咲",
    quantity: 10,
    productId: "QR",
    printedAt: "2025/11/3",
    createdAt: "2025/11/5",
  },
  {
    id: "production_002",
    product: "デニムジャケット ヴィンテージ加工",
    brand: "NEXUS Street",
    manager: "高橋 健太",
    quantity: 9,
    productId: "QR",
    printedAt: "2025/11/4",
    createdAt: "2025/11/5",
  },
  {
    id: "production_003",
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    manager: "佐藤 美咲",
    quantity: 7,
    productId: "QR",
    printedAt: "2025/11/1",
    createdAt: "2025/10/31",
  },
  {
    id: "production_004",
    product: "デニムジャケット ヴィンテージ加工",
    brand: "NEXUS Street",
    manager: "高橋 健太",
    quantity: 4,
    productId: "QR",
    printedAt: "2025/10/30",
    createdAt: "2025/10/29",
  },
  {
    id: "production_005",
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    manager: "佐藤 美咲",
    quantity: 2,
    productId: "QR",
    printedAt: "-",
    createdAt: "2025/11/4",
  },
];
