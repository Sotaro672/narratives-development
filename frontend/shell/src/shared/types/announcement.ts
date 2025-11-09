// frontend/shell/src/shared/types/announcement.ts

/**
 * AnnouncementCategory / TargetAudience / AnnouncementStatus
 * backend/internal/domain/announcement/entity.go の string alias に対応。
 *
 * 実際の値セット（"system", "release", "maintenance" など）は
 * サーバーまたは CMS 側で定義されたものに準拠する。
 */
export type AnnouncementCategory = string;
export type TargetAudience = string;
export type AnnouncementStatus = string;

/**
 * Announcement
 * backend/internal/domain/announcement/entity.go に対応する共通型。
 *
 * - 日付は ISO8601 文字列
 * - camelCase 命名を採用
 */
export interface Announcement {
  id: string;
  title: string;
  content: string;

  category: AnnouncementCategory;
  targetAudience: TargetAudience;

  /** 特定トークンに紐づく場合のトークンID（任意） */
  targetToken?: string | null;

  /** 対象となる商品ID一覧（任意） */
  targetProducts?: string[];

  /** 対象となるアバターID一覧（任意） */
  targetAvatars?: string[];

  /** 公開フラグ */
  isPublished: boolean;

  /** 公開日時（任意） */
  publishedAt?: string | null;

  /** 添付ファイルID一覧（announcementAttachment の ID） */
  attachments?: string[];

  /** ステータス（例: "draft" | "scheduled" | "published" | "archived"） */
  status: AnnouncementStatus;

  /** 作成情報 */
  createdAt: string;
  createdBy: string;

  /** 更新情報 */
  updatedAt?: string | null;
  updatedBy?: string | null;

  /** 論理削除情報 */
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * AnnouncementPatch
 * 部分更新用（backend の Patch モデルに対応）
 *
 * - undefined: 変更なし
 * - null: クリアを意味する場合に利用（サーバー仕様に依存）
 */
export interface AnnouncementPatch {
  title?: string | null;
  content?: string | null;
  category?: AnnouncementCategory | null;
  targetAudience?: TargetAudience | null;
  targetToken?: string | null;
  targetProducts?: string[] | null;
  targetAvatars?: string[] | null;
  isPublished?: boolean | null;
  publishedAt?: string | null;
  attachments?: string[] | null;
  status?: AnnouncementStatus | null;

  createdAt?: string | null;
  createdBy?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * バリデーション関数（フォーム送信前などで使用）
 */
export function validateAnnouncement(a: Announcement): string[] {
  const errors: string[] = [];

  if (!a.id?.trim()) errors.push("id は必須です");
  if (!a.title?.trim()) errors.push("タイトルは必須です");
  if (!a.content?.trim()) errors.push("本文は必須です");
  if (!String(a.category).trim()) errors.push("カテゴリは必須です");
  if (!String(a.targetAudience).trim()) errors.push("配信対象は必須です");
  if (!String(a.status).trim()) errors.push("ステータスは必須です");
  if (!a.createdBy?.trim()) errors.push("作成者は必須です");
  if (!a.createdAt?.trim()) errors.push("作成日時は必須です");

  return errors;
}

/**
 * 公開状態切り替えヘルパ
 * 実際の状態変更は API 層で確定させること
 */
export function markPublished(
  a: Announcement,
  publishedAt: string,
  updatedBy?: string
): Announcement {
  return {
    ...a,
    isPublished: true,
    publishedAt,
    updatedAt: publishedAt,
    updatedBy: updatedBy ?? a.updatedBy ?? null,
  };
}

export function markUnpublished(
  a: Announcement,
  updatedAt: string,
  updatedBy?: string
): Announcement {
  return {
    ...a,
    isPublished: false,
    updatedAt,
    updatedBy: updatedBy ?? a.updatedBy ?? null,
  };
}
