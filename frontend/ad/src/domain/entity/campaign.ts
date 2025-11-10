// frontend/ad/src/domain/entity/campaign.ts

/**
 * CampaignStatus
 * backend/internal/domain/campaign/entity.go の CampaignStatus に対応。
 *
 * - "draft"     : 下書き
 * - "active"    : 配信中
 * - "paused"    : 一時停止
 * - "scheduled" : 配信予約
 * - "completed" : 配信終了
 * - "deleted"   : 削除済み（論理削除）
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
 * backend/internal/domain/campaign/entity.go の AdType に対応。
 *
 * - "image_carousel"
 * - "video"
 * - "story"
 * - "reel"
 * - "banner"
 * - "native"
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
 * backend/internal/domain/campaign/entity.go の Campaign に対応。
 *
 * - 日付は ISO8601 文字列（例: "2025-01-10T00:00:00Z"）を想定
 * - *_By 系は省略可能（null or undefined）
 * - *_At 系も省略可能なものは null or undefined
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
 * Policy (backend と同期させる定数群)
 * backend/internal/domain/campaign/entity.go の Policy 相当。
 */
export const MIN_CAMPAIGN_BUDGET = 0; // 0 以上
export const MAX_CAMPAIGN_BUDGET = 0; // 0 の場合は上限なし

export const MIN_CAMPAIGN_SPENT = 0; // 0 以上
export const MAX_CAMPAIGN_SPENT = 0; // 0 の場合は上限なし

// overspend を禁止するかどうか（spent <= budget）
export const DISALLOW_CAMPAIGN_OVERSPEND = true;

// 文字数制限
export const MAX_CAMPAIGN_NAME_LENGTH = 200;
export const MAX_CAMPAIGN_AUDIENCE_LENGTH = 1000;
export const MAX_CAMPAIGN_HEADLINE_LENGTH = 120;
export const MAX_CAMPAIGN_DESCRIPTION_LENGTH = 2000;

/**
 * Campaign の簡易バリデーション
 * backend の validate() ロジックと整合する範囲でチェック。
 * UI/フォーム入力チェック用。
 */
export function validateCampaign(c: Campaign): boolean {
  // id
  if (!c.id?.trim()) return false;

  // name
  if (!c.name?.trim()) return false;
  if (
    MAX_CAMPAIGN_NAME_LENGTH > 0 &&
    [...c.name].length > MAX_CAMPAIGN_NAME_LENGTH
  ) {
    return false;
  }

  // brandId / assigneeId / listId
  if (!c.brandId?.trim()) return false;
  if (!c.assigneeId?.trim()) return false;
  if (!c.listId?.trim()) return false;

  // status
  if (!isValidCampaignStatus(c.status)) return false;

  // budget / spent
  if (
    typeof c.budget !== "number" ||
    Number.isNaN(c.budget) ||
    c.budget < MIN_CAMPAIGN_BUDGET ||
    (MAX_CAMPAIGN_BUDGET > 0 && c.budget > MAX_CAMPAIGN_BUDGET)
  ) {
    return false;
  }
  if (
    typeof c.spent !== "number" ||
    Number.isNaN(c.spent) ||
    c.spent < MIN_CAMPAIGN_SPENT ||
    (MAX_CAMPAIGN_SPENT > 0 && c.spent > MAX_CAMPAIGN_SPENT)
  ) {
    return false;
  }
  if (DISALLOW_CAMPAIGN_OVERSPEND && c.spent > c.budget) {
    return false;
  }

  // dates
  const start = parseIso(c.startDate);
  const end = parseIso(c.endDate);
  if (!start || !end || end.getTime() < start.getTime()) {
    return false;
  }

  // targetAudience
  if (!c.targetAudience?.trim()) return false;
  if (
    MAX_CAMPAIGN_AUDIENCE_LENGTH > 0 &&
    [...c.targetAudience].length > MAX_CAMPAIGN_AUDIENCE_LENGTH
  ) {
    return false;
  }

  // adType
  if (!isValidAdType(c.adType)) return false;

  // headline
  if (!c.headline?.trim()) return false;
  if (
    MAX_CAMPAIGN_HEADLINE_LENGTH > 0 &&
    [...c.headline].length > MAX_CAMPAIGN_HEADLINE_LENGTH
  ) {
    return false;
  }

  // description
  if (!c.description?.trim()) return false;
  if (
    MAX_CAMPAIGN_DESCRIPTION_LENGTH > 0 &&
    [...c.description].length > MAX_CAMPAIGN_DESCRIPTION_LENGTH
  ) {
    return false;
  }

  // Optional IDs / By: 空文字は不可（指定されるなら非空）
  if (c.performanceId != null && !c.performanceId.trim()) return false;
  if (c.imageId != null && !c.imageId.trim()) return false;
  if (c.createdBy != null && !c.createdBy.trim()) return false;
  if (c.updatedBy != null && !c.updatedBy.trim()) return false;
  if (c.deletedBy != null && !c.deletedBy.trim()) return false;

  // createdAt
  const createdAt = parseIso(c.createdAt);
  if (!createdAt) return false;

  // updatedAt / deletedAt: createdAt との前後関係
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

/**
 * GraphQL / API 通信用の入力 DTO
 * - 新規登録・更新時に利用する軽量型
 * - optional なフィールドは backend 側の default / patch に委譲
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

/** ISO8601 文字列を Date に変換（失敗時は null） */
function parseIso(s: string | null | undefined): Date | null {
  if (!s) return null;
  const t = Date.parse(s);
  if (Number.isNaN(t)) return null;
  return new Date(t);
}
