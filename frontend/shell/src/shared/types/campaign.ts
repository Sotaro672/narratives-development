// frontend/shell/src/shared/types/campaign.ts

/**
 * CampaignStatus
 * backend/internal/domain/campaign/entity.go に対応。
 */
export type CampaignStatus =
  | "draft"
  | "active"
  | "paused"
  | "scheduled"
  | "completed"
  | "deleted";

/** CampaignStatus の妥当性チェック */
export function isValidCampaignStatus(s: string): s is CampaignStatus {
  return (
    s === "draft" ||
    s === "active" ||
    s === "paused" ||
    s === "scheduled" ||
    s === "completed" ||
    s === "deleted"
  );
}

/**
 * AdType
 * backend/internal/domain/campaign/entity.go に対応。
 */
export type AdType =
  | "image_carousel"
  | "video"
  | "story"
  | "reel"
  | "banner"
  | "native";

/** AdType の妥当性チェック */
export function isValidAdType(t: string): t is AdType {
  return (
    t === "image_carousel" ||
    t === "video" ||
    t === "story" ||
    t === "reel" ||
    t === "banner" ||
    t === "native"
  );
}

/**
 * Campaign
 * backend/internal/domain/campaign/entity.go の構造に対応。
 * - ISO8601 文字列を日付型として保持。
 * - *_At, *_By は省略可能。
 */
export interface Campaign {
  id: string;
  name: string;
  brandId: string;
  assigneeId: string;
  listId: string;
  status: CampaignStatus;
  budget: number;
  spent: number;
  startDate: string;
  endDate: string;
  targetAudience: string;
  adType: AdType;
  headline: string;
  description: string;

  performanceId?: string | null;
  imageId?: string | null;
  createdBy?: string | null;
  createdAt: string;
  updatedBy?: string | null;
  updatedAt?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * CampaignInput
 * GraphQL / API 用の入力データ構造。
 */
export interface CampaignInput {
  id?: string;
  name: string;
  brandId: string;
  assigneeId: string;
  listId: string;
  status: CampaignStatus;
  budget: number;
  spent: number;
  startDate: string;
  endDate: string;
  targetAudience: string;
  adType: AdType;
  headline: string;
  description: string;
  performanceId?: string | null;
  imageId?: string | null;
}

/**
 * Policy constants (backend と同期)
 */
export const CAMPAIGN_POLICY = {
  MIN_BUDGET: 0,
  MAX_BUDGET: 0, // 0 = no upper bound
  MIN_SPENT: 0,
  MAX_SPENT: 0, // 0 = no upper bound
  DISALLOW_OVERSPEND: true,
  MAX_NAME_LENGTH: 200,
  MAX_AUDIENCE_LENGTH: 1000,
  MAX_HEADLINE_LENGTH: 120,
  MAX_DESCRIPTION_LENGTH: 2000,
};

/**
 * Campaign の簡易バリデーション
 * backend の validate() と整合する範囲で検証。
 */
export function validateCampaign(c: Campaign): boolean {
  if (!c.id?.trim()) return false;
  if (!c.name?.trim() || c.name.length > CAMPAIGN_POLICY.MAX_NAME_LENGTH)
    return false;
  if (!c.brandId?.trim()) return false;
  if (!c.assigneeId?.trim()) return false;
  if (!c.listId?.trim()) return false;
  if (!isValidCampaignStatus(c.status)) return false;

  if (
    c.budget < CAMPAIGN_POLICY.MIN_BUDGET ||
    (CAMPAIGN_POLICY.MAX_BUDGET > 0 && c.budget > CAMPAIGN_POLICY.MAX_BUDGET)
  )
    return false;

  if (
    c.spent < CAMPAIGN_POLICY.MIN_SPENT ||
    (CAMPAIGN_POLICY.MAX_SPENT > 0 && c.spent > CAMPAIGN_POLICY.MAX_SPENT)
  )
    return false;

  if (CAMPAIGN_POLICY.DISALLOW_OVERSPEND && c.spent > c.budget) return false;

  const start = parseIso(c.startDate);
  const end = parseIso(c.endDate);
  if (!start || !end || end.getTime() < start.getTime()) return false;

  if (
    !c.targetAudience?.trim() ||
    c.targetAudience.length > CAMPAIGN_POLICY.MAX_AUDIENCE_LENGTH
  )
    return false;

  if (!isValidAdType(c.adType)) return false;

  if (
    !c.headline?.trim() ||
    c.headline.length > CAMPAIGN_POLICY.MAX_HEADLINE_LENGTH
  )
    return false;

  if (
    !c.description?.trim() ||
    c.description.length > CAMPAIGN_POLICY.MAX_DESCRIPTION_LENGTH
  )
    return false;

  const createdAt = parseIso(c.createdAt);
  if (!createdAt) return false;

  if (c.updatedAt) {
    const ua = parseIso(c.updatedAt);
    if (!ua || ua.getTime() < createdAt.getTime()) return false;
  }

  if (c.deletedAt) {
    const da = parseIso(c.deletedAt);
    if (!da || da.getTime() < createdAt.getTime()) return false;
    if (c.updatedAt) {
      const ua = parseIso(c.updatedAt);
      if (ua && da.getTime() < ua.getTime()) return false;
    }
  }

  return true;
}

/** ISO8601 日付の解析（失敗時は null） */
function parseIso(s?: string | null): Date | null {
  if (!s) return null;
  const t = Date.parse(s);
  return Number.isNaN(t) ? null : new Date(t);
}
