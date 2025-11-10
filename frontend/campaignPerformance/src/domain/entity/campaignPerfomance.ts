// frontend/campaignPerformance/src/domain/entity/campaignPerformance.ts

/**
 * CampaignPerformance
 * backend/internal/domain/campaignPerformance/entity.go に対応。
 *
 * 広告キャンペーンの実績データを表すエンティティ。
 * - impressions, clicks, conversions, purchases は非負整数
 * - 各値の関係性: purchases ≤ conversions ≤ clicks ≤ impressions
 * - lastUpdatedAt は ISO8601 UTC 文字列
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
 * Domain constants（GoのPolicyと同期）
 */
export const PERFORMANCE_POLICY = {
  minImpressions: 0,
  minClicks: 0,
  minConversions: 0,
  minPurchases: 0,
  enforceClicksLEImpressions: true,
  enforceConversionsLEClicks: true,
  enforcePurchasesLEConversions: true,
  maxImpressions: 0, // 0 disables upper bound
  maxClicks: 0,
  maxConversions: 0,
  maxPurchases: 0,
};

/**
 * Validation
 * backend の validate() / validateCounts() と同様のロジックを実装。
 */
export function validateCampaignPerformance(cp: CampaignPerformance): boolean {
  if (!cp.id?.trim()) return false;
  if (!cp.campaignId?.trim()) return false;
  if (cp.lastUpdatedAt == null || Number.isNaN(Date.parse(cp.lastUpdatedAt)))
    return false;

  const { 
    minImpressions, minClicks, minConversions, minPurchases,
    maxImpressions, maxClicks, maxConversions, maxPurchases,
    enforceClicksLEImpressions, enforceConversionsLEClicks, enforcePurchasesLEConversions
  } = PERFORMANCE_POLICY;

  // Non-negative lower bounds
  if (
    cp.impressions < minImpressions ||
    cp.clicks < minClicks ||
    cp.conversions < minConversions ||
    cp.purchases < minPurchases
  ) {
    return false;
  }

  // Optional upper bounds
  if (
    (maxImpressions > 0 && cp.impressions > maxImpressions) ||
    (maxClicks > 0 && cp.clicks > maxClicks) ||
    (maxConversions > 0 && cp.conversions > maxConversions) ||
    (maxPurchases > 0 && cp.purchases > maxPurchases)
  ) {
    return false;
  }

  // Logical order checks
  if (enforceClicksLEImpressions && cp.clicks > cp.impressions) return false;
  if (enforceConversionsLEClicks && cp.conversions > cp.clicks) return false;
  if (enforcePurchasesLEConversions && cp.purchases > cp.conversions)
    return false;

  return true;
}

/**
 * Utility: create a new CampaignPerformance object
 */
export function createCampaignPerformance(
  id: string,
  campaignId: string,
  impressions: number,
  clicks: number,
  conversions: number,
  purchases: number,
  lastUpdatedAt: string = new Date().toISOString()
): CampaignPerformance {
  return {
    id: id.trim(),
    campaignId: campaignId.trim(),
    impressions,
    clicks,
    conversions,
    purchases,
    lastUpdatedAt,
  };
}

/**
 * Example:
 * const cp = createCampaignPerformance("perf_001", "cmp_001", 12000, 800, 150, 30);
 * if (validateCampaignPerformance(cp)) console.log("✅ valid performance");
 */
