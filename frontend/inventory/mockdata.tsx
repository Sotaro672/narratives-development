// frontend/inventory/mockdata.tsx

export type InventoryRow = {
  product: string;
  brand: string;
  token: string;
  total: number;
};

export const INVENTORIES: InventoryRow[] = [
  {
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    token: "LUMINA VIP Token",
    total: 221,
  },
  {
    product: "デニムジャケット ヴィンテージ加工",
    brand: "NEXUS Street",
    token: "NEXUS Community Token",
    total: 222,
  },
];
