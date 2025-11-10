// frontend\shell\src\shared\types\campaginImage.ts

import type { Campaign, CampaignStatus, AdType } from "../../../../shell/src/shared/types/campaign";

/**
 * 広告キャンペーンのモックデータ
 * - backend/internal/domain/campaign/entity.go および shared/types/campaign.ts に準拠
 * - 予算・支出は数値（JPY）で保持
 * - 日付は ISO8601 UTC 文字列
 */
export const CAMPAIGNS: Campaign[] = [
  {
    id: "cmp_001",
    name: "NEXUS Street デニムジャケット新作",
    brandId: "brand_nexus",
    assigneeId: "member_watanabe",
    listId: "list_ads_2024spring",
    status: "active" satisfies CampaignStatus,
    budget: 300000,
    spent: 198000,
    startDate: "2024-03-10T00:00:00Z",
    endDate: "2024-04-10T00:00:00Z",
    targetAudience: "20〜40代のファッション志向ユーザー（都心エリア）",
    adType: "image_carousel" satisfies AdType,
    headline: "春の新作デニム、登場。",
    description:
      "NEXUS Street から新作デニムジャケットが登場。限定カラーと特別キャンペーン実施中！",
    performanceId: null,
    imageId: null,
    createdBy: "member_watanabe",
    createdAt: "2024-03-01T10:00:00Z",
    updatedBy: "member_watanabe",
    updatedAt: "2024-03-15T08:00:00Z",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "cmp_002",
    name: "LUMINA Fashion 春コレクション",
    brandId: "brand_lumina",
    assigneeId: "member_sato",
    listId: "list_ads_2024spring",
    status: "active" satisfies CampaignStatus,
    budget: 500000,
    spent: 342000,
    startDate: "2024-03-01T00:00:00Z",
    endDate: "2024-03-31T00:00:00Z",
    targetAudience: "女性向けアパレル愛好層（25〜45歳、関東・関西圏）",
    adType: "video" satisfies AdType,
    headline: "春をまとう、LUMINA。",
    description:
      "LUMINA Fashion の2024春コレクション。軽やかな素材と彩りで、春をもっと華やかに。",
    performanceId: null,
    imageId: null,
    createdBy: "member_sato",
    createdAt: "2024-02-25T09:30:00Z",
    updatedBy: "member_sato",
    updatedAt: "2024-03-10T11:15:00Z",
    deletedAt: null,
    deletedBy: null,
  },
];

/**
 * 表示用の派生データ型（UI レイヤーで利用）
 * - フロントエンドの一覧表示用に加工された形式
 */
export type AdRow = {
  campaign: string;
  brand: string;
  owner: string;
  period: string; // "YYYY/M/D - YYYY/M/D"
  status: string; // e.g. "実行中", "停止中", etc.
  spendRate: string; // "66.0%"
  spend: string; // "¥198,000"
  budget: string; // "¥300,000"
};

/**
 * CAMPAIGNS から AdRow を生成するユーティリティ
 */
export function mapCampaignsToAdRows(campaigns: Campaign[]): AdRow[] {
  return campaigns.map((c) => {
    const start = new Date(c.startDate);
    const end = new Date(c.endDate);
    const period = `${start.getFullYear()}/${start.getMonth() + 1}/${start.getDate()} - ${end.getFullYear()}/${end.getMonth() + 1}/${end.getDate()}`;
    const spendRate =
      c.budget > 0 ? ((c.spent / c.budget) * 100).toFixed(1) + "%" : "0%";
    return {
      campaign: c.name,
      brand:
        c.brandId === "brand_nexus"
          ? "NEXUS Street"
          : c.brandId === "brand_lumina"
          ? "LUMINA Fashion"
          : "不明ブランド",
      owner:
        c.assigneeId === "member_sato"
          ? "佐藤 美咲"
          : c.assigneeId === "member_watanabe"
          ? "渡辺 花子"
          : "不明担当者",
      period,
      status: c.status === "active" ? "実行中" : "停止中",
      spendRate,
      spend: `¥${c.spent.toLocaleString()}`,
      budget: `¥${c.budget.toLocaleString()}`,
    };
  });
}

// 派生データとして利用可能な一覧
export const ADS: AdRow[] = mapCampaignsToAdRows(CAMPAIGNS);
