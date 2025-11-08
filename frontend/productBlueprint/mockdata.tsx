// frontend/productBlueprint/mockdata.tsx

export type ProductBlueprintRow = {
  product: string;
  brand: "LUMINA Fashion" | "NEXUS Street";
  owner: string;
  productId: string;
  createdAtA: string;
  createdAtB: string;
};

export const RAW_ROWS: ProductBlueprintRow[] = [
  {
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    owner: "佐藤 美咲",
    productId: "QR",
    createdAtA: "2024/1/15",
    createdAtB: "2024/1/15",
  },
  {
    product: "デニムジャケット ヴィンテージ加工",
    brand: "NEXUS Street",
    owner: "高橋 健太",
    productId: "QR",
    createdAtA: "2024/1/10",
    createdAtB: "2024/1/10",
  },
];
