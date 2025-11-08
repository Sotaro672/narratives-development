// frontend/brand/mockdata.tsx

export type BrandRow = {
  name: string;
  status: "active" | "inactive";
  owner: string;
  registeredAt: string; // YYYY/M/D 表示用
};

// ダミーデータ（必要に応じてAPI結果に置き換え）
export const ALL_BRANDS: BrandRow[] = [
  {
    name: "NEXUS Street",
    status: "active",
    owner: "渡辺 花子",
    registeredAt: "2024/2/1",
  },
  {
    name: "LUMINA Fashion",
    status: "active",
    owner: "佐藤 美咲",
    registeredAt: "2024/1/1",
  },
];
