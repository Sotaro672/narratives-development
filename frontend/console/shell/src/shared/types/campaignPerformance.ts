// frontend/shell/src/shared/types/campaignPerformance.ts
// (Generated from frontend/campaignPerformance/src/domain/entity/campaignPerformance.ts
//  and backend/internal/domain/campaignPerformance/entity.go)

/**
 * CampaignPerformance
 * 広告キャンペーンの実績データを表す共通型。
 * backend/internal/domain/campaignPerformance/entity.go に準拠。
 *
 * - impressions, clicks, conversions, purchases は非負整数
 * - 各値は論理的順序を満たす必要あり：
 *   purchases ≤ conversions ≤ clicks ≤ impressions
 * - lastUpdatedAt は ISO8601 UTC 文字列（例: "2025-01-10T12:30:00Z"）
 */
export interface CampaignPerformance {
  id: string;
  campaignId: string;
  impressions: number;
  clicks: number;
  conversions: number;
  purchases: number;
  lastUpdatedAt: string;
}

/**
 * Policy（backend の Go 実装と同期）
 */
export const CAMPAIGN_PERFORMANCE_POLICY = {
  minImpressions: 0,
  minClicks: 0,
  minConversions: 0,
  minPurchases: 0,
  enforceClicksLEImpressions: true,
  enforceConversionsLEClicks: true,
  enforcePurchasesLEConversions: true,
  maxImpressions: 0, // 0 disables upper bound check
  maxClicks: 0,
  maxConversions: 0,
  maxPurchases: 0,
};

/**
 * CampaignPerformance のバリデーション関数
 * backend の validate() および validateCounts() に準拠。
 */
export function validateCampaignPerformance(cp: CampaignPerformance): boolean {
  if (!cp.id?.trim()) return false;
  if (!cp.campaignId?.trim()) return false;
  if (!cp.lastUpdatedAt || isNaN(Date.parse(cp.lastUpdatedAt))) return false;

  const {
    minImpressions,
    minClicks,
    minConversions,
    minPurchases,
    maxImpressions,
    maxClicks,
    maxConversions,
    maxPurchases,
    enforceClicksLEImpressions,
    enforceConversionsLEClicks,
    enforcePurchasesLEConversions,
  } = CAMPAIGN_PERFORMANCE_POLICY;

  // Non-negative checks
  if (
    cp.impressions < minImpressions ||
    cp.clicks < minClicks ||
    cp.conversions < minConversions ||
    cp.purchases < minPurchases
  ) {
    return false;
  }

  // Upper bounds
  if (
    (maxImpressions > 0 && cp.impressions > maxImpressions) ||
    (maxClicks > 0 && cp.clicks > maxClicks) ||
    (maxConversions > 0 && cp.conversions > maxConversions) ||
    (maxPurchases > 0 && cp.purchases > maxPurchases)
  ) {
    return false;
  }

  // Logical order
  if (enforceClicksLEImpressions && cp.clicks > cp.impressions) return false;
  if (enforceConversionsLEClicks && cp.conversions > cp.clicks) return false;
  if (enforcePurchasesLEConversions && cp.purchases > cp.conversions)
    return false;

  return true;
}

/**
 * Utility: 新しい CampaignPerformance オブジェクトを生成
 * - now 引数を省略した場合、現在時刻を ISO8601 形式で設定。
 */
export function createCampaignPerformance(
  id: string,
  campaignId: string,
  impressions: number,
  clicks: number,
  conversions: number,
  purchases: number,
  now: string = new Date().toISOString()
): CampaignPerformance {
  return {
    id: id.trim(),
    campaignId: campaignId.trim(),
    impressions,
    clicks,
    conversions,
    purchases,
    lastUpdatedAt: now,
  };
}

/**
 * Example:
 * const perf = createCampaignPerformance("perf_001", "cmp_001", 10000, 800, 200, 50);
 * if (validateCampaignPerformance(perf)) console.log("✅ valid performance");
 */
