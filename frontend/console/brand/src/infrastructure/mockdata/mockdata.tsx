// frontend\brand\src\infrastructure\mockdata\mockdata.tsx

import type { Brand } from "../../../../shell/src/shared/types/brand";

/**
 * BrandRow
 * 一覧表示などUI用に Brand から派生した軽量型。
 */
export type BrandRow = {
  id: string;
  name: string;
  isActive: boolean;
  owner: string;
  walletAddress: string;
  websiteUrl?: string;
  registeredAt: string; // YYYY/MM/DD 表示用
};

/**
 * ダミーデータ（バックエンドAPIの Brand モデルを模倣）
 * - isActive: status代替
 * - createdAt: ISO8601
 */
export const ALL_BRANDS: Brand[] = [
  {
    id: "brand-001",
    companyId: "comp-001",
    name: "NEXUS Street",
    description: "都市系ストリートブランド。NFTトークンによる真贋証明を実装。",
    websiteUrl: "https://nexus-street.example.com",
    isActive: true,
    managerId: "mem-002",
    walletAddress: "SoL11111111111111111111111111111111111111111",
    createdAt: "2024-02-01T00:00:00Z",
    createdBy: "mem-009",
  },
  {
    id: "brand-002",
    companyId: "comp-001",
    name: "LUMINA Fashion",
    description: "ラグジュアリーラインを展開するファッションブランド。",
    websiteUrl: "https://lumina-fashion.example.com",
    isActive: true,
    managerId: "mem-008",
    walletAddress: "SoL22222222222222222222222222222222222222222",
    createdAt: "2024-01-01T00:00:00Z",
    createdBy: "mem-009",
  },
];

/**
 * Brand → BrandRow 変換ユーティリティ
 * UI（一覧表示など）用に整形。
 */
export const toBrandRows = (brands: Brand[]): BrandRow[] =>
  brands.map((b) => ({
    id: b.id,
    name: b.name,
    isActive: b.isActive,
    owner: b.managerId ? resolveOwnerName(b.managerId) : "未設定",
    walletAddress: b.walletAddress,
    websiteUrl: b.websiteUrl,
    registeredAt: b.createdAt.slice(0, 10).replace(/-/g, "/"),
  }));

/**
 * ダミー: managerId → 氏名変換（実際は Member API から取得）
 */
function resolveOwnerName(managerId: string): string {
  switch (managerId) {
    case "mem-008":
      return "佐藤 美咲";
    case "mem-002":
      return "渡辺 花子";
    default:
      return "未設定";
  }
}
