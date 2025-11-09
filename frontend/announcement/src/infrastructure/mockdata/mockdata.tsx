// frontend/announcement/src/infrastructure/mock/mockdata.tsx

import type { Announcement } from "../../../../shell/src/shared/types/announcement";

/**
 * ダミーアナウンスデータ
 * backend/internal/domain/announcement/entity.go と同じ構造を模倣。
 * - ISO8601形式の日時
 * - category / status / isPublished を正規化
 */
export const MOCK_ANNOUNCEMENTS: Announcement[] = [
  {
    id: "ann_001",
    title: "Solid State Console ベータ版リリース",
    content:
      "一部機能をベータ版として公開しました。皆さまからのフィードバックをお待ちしています。",
    category: "release", // backend準拠（例: "release"）
    targetAudience: "all",
    isPublished: true,
    publishedAt: "2025-11-01T00:00:00Z",
    attachments: [],
    status: "published",
    createdAt: "2025-10-30T12:00:00Z",
    createdBy: "admin",
    updatedAt: "2025-11-01T00:00:00Z",
    updatedBy: "admin",
  },
  {
    id: "ann_002",
    title: "システムメンテナンスのお知らせ",
    content:
      "2025年11月10日 01:00〜03:00 の間、システムメンテナンスを実施します。この時間帯は一部機能をご利用いただけません。",
    category: "maintenance",
    targetAudience: "all",
    isPublished: true,
    publishedAt: "2025-10-28T00:00:00Z",
    attachments: [],
    status: "published",
    createdAt: "2025-10-25T10:00:00Z",
    createdBy: "system",
    updatedAt: "2025-10-28T00:00:00Z",
    updatedBy: "system",
  },
  {
    id: "ann_003",
    title: "ブランド別お知らせテスト",
    content:
      "特定ブランド担当者のみに配信されるお知らせのテスト投稿です。",
    category: "system",
    targetAudience: "brand-managers",
    targetProducts: ["prd_001"],
    targetAvatars: [],
    isPublished: false,
    attachments: [],
    status: "draft",
    createdAt: "2025-11-05T06:00:00Z",
    createdBy: "support",
  },
];

/**
 * UI向け変換：Announcement → 一覧表示用オブジェクト
 */
export type AnnouncementRow = {
  id: string;
  title: string;
  category: string;
  status: string;
  publishedAt: string;
};

export const toAnnouncementRows = (
  list: Announcement[]
): AnnouncementRow[] => {
  return list.map((a) => ({
    id: a.id,
    title: a.title,
    category:
      a.category === "release"
        ? "アップデート"
        : a.category === "maintenance"
        ? "メンテナンス"
        : a.category === "system"
        ? "システム"
        : "お知らせ",
    status: a.isPublished ? "公開中" : "下書き",
    publishedAt: a.publishedAt
      ? a.publishedAt.slice(0, 10).replace(/-/g, "/")
      : "-",
  }));
};
