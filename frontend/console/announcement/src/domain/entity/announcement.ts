// frontend/announcement/src/domain/entity/announcement.ts

/**
 * AnnouncementCategory / TargetAudience / AnnouncementStatus
 * backend/internal/domain/announcement/entity.go の string alias に対応。
 *
 * 実際の値セット（"system", "release", ... 等）は
 * APIスキーマ / デザイン仕様に合わせて管理側で定義してください。
 */
export type AnnouncementCategory = string;
export type TargetAudience = string;
export type AnnouncementStatus = string;

/**
 * Announcement
 * backend/internal/domain/announcement/entity.go の Announcement に対応するフロントエンド用ドメインモデル。
 *
 * - 日付は ISO8601 文字列として扱う
 * - フロント側も backend と同じフィールド名 & camelCase を採用
 */
export interface Announcement {
  id: string;
  title: string;
  content: string;

  category: AnnouncementCategory;

  /** 配信対象（例: "all", "brand-managers", "token-managers", ...） */
  targetAudience: TargetAudience;

  /** 特定トークンに紐づく場合のトークンID（任意） */
  targetToken?: string | null;

  /** 対象となる商品ID一覧（任意） */
  targetProducts?: string[];

  /** 対象となるアバターID一覧（任意） */
  targetAvatars?: string[];

  /** 公開フラグ */
  isPublished: boolean;

  /** 公開日時（任意, isPublished=true の場合に設定される想定） */
  publishedAt?: string | null;

  /** 添付ファイルID一覧（announcementAttachment の ID） */
  attachments?: string[];

  /** ステータス（例: "draft" | "scheduled" | "published" | "archived" 等） */
  status: AnnouncementStatus;

  /** 作成情報 */
  createdAt: string; // ISO8601
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
 * 部分更新用（backend の更新仕様と整合させる前提）
 *
 * - undefined: 変更なし
 * - null: サーバ側仕様に応じて「クリア」を意味させる場合に利用
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
 * フロント側簡易バリデーション
 * （バックエンドの厳密な validate() とは独立して、フォーム送信前チェック等に使用）
 */
export function validateAnnouncement(a: Announcement): string[] {
  const errors: string[] = [];

  if (!a.id?.trim()) errors.push("id is required");
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
 * 公開状態をローカルで切り替えるヘルパ
 * 実際の状態変更は必ず API（usecase）側で確定させること。
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
    // publishedAt はクリアするかどうかは運用ポリシー次第（ここでは保持）
    updatedAt,
    updatedBy: updatedBy ?? a.updatedBy ?? null,
  };
}
