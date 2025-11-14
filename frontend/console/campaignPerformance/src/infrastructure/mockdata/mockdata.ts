// frontend/campaignPerformance/src/infrastructure/mockdata/mockdata.ts
// (Generated from frontend/shell/src/shared/types/campaignPerformance.ts)

import type { CampaignPerformance } from "../../../../shell/src/shared/types/campaignPerformance";

/**
 * モックデータ: CampaignPerformance
 * backend/internal/domain/campaignPerformance/entity.go に準拠。
 *
 * - 各値は論理的順序を満たす (purchases ≤ conversions ≤ clicks ≤ impressions)
 * - lastUpdatedAt は ISO8601 UTC 文字列
 */
export const CAMPAIGN_PERFORMANCES: CampaignPerformance[] = [
  {
    id: "perf_001",
    campaignId: "cmp_001",
    impressions: 12000,
    clicks: 950,
    conversions: 180,
    purchases: 45,
    lastUpdatedAt: "2024-03-20T10:00:00Z",
  },
  {
    id: "perf_002",
    campaignId: "cmp_002",
    impressions: 18500,
    clicks: 1320,
    conversions: 300,
    purchases: 72,
    lastUpdatedAt: "2024-03-21T09:30:00Z",
  },
  {
    id: "perf_003",
    campaignId: "cmp_003",
    impressions: 9400,
    clicks: 710,
    conversions: 155,
    purchases: 39,
    lastUpdatedAt: "2024-03-19T15:45:00Z",
  },
  {
    id: "perf_004",
    campaignId: "cmp_004",
    impressions: 15200,
    clicks: 890,
    conversions: 200,
    purchases: 50,
    lastUpdatedAt: "2024-03-22T11:10:00Z",
  },
  {
    id: "perf_005",
    campaignId: "cmp_005",
    impressions: 20300,
    clicks: 1500,
    conversions: 330,
    purchases: 84,
    lastUpdatedAt: "2024-03-23T08:00:00Z",
  },
];

/**
 * 集計や表示用のユーティリティ関数群
 */

/** CTR（クリック率）を算出: clicks / impressions * 100 */
export function calcCTR(perf: CampaignPerformance): string {
  if (perf.impressions === 0) return "0.0%";
  return ((perf.clicks / perf.impressions) * 100).toFixed(1) + "%";
}

/** CVR（コンバージョン率）を算出: conversions / clicks * 100 */
export function calcCVR(perf: CampaignPerformance): string {
  if (perf.clicks === 0) return "0.0%";
  return ((perf.conversions / perf.clicks) * 100).toFixed(1) + "%";
}

/** 購入率（Purchase Rate）を算出: purchases / conversions * 100 */
export function calcPurchaseRate(perf: CampaignPerformance): string {
  if (perf.conversions === 0) return "0.0%";
  return ((perf.purchases / perf.conversions) * 100).toFixed(1) + "%";
}

/**
 * 集計データ例:
 * - CTR: クリック率
 * - CVR: コンバージョン率
 * - PR: 購入率
 */
export const CAMPAIGN_PERFORMANCE_SUMMARY = CAMPAIGN_PERFORMANCES.map((p) => ({
  campaignId: p.campaignId,
  impressions: p.impressions,
  clicks: p.clicks,
  conversions: p.conversions,
  purchases: p.purchases,
  ctr: calcCTR(p),
  cvr: calcCVR(p),
  purchaseRate: calcPurchaseRate(p),
  lastUpdatedAt: p.lastUpdatedAt,
}));
