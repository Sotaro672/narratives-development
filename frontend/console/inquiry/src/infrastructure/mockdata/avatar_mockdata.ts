// frontend/inquiry/src/infrastructure/mockdata/avatar_mockdata.ts

import type { Avatar } from "../../../../shell/src/shared/types/avatar";

/**
 * モック用 Avatar データ
 * frontend/shell/src/shared/types/avatar.ts に準拠。
 *
 * - createdAt / updatedAt は ISO8601 UTC 文字列
 * - avatarState は簡易オブジェクトで代替
 */

export const AVATARS: Avatar[] = [
  {
    id: "avatar_001",
    userId: "user_001",
    avatarName: "LUMINA Manager",
    avatarIconId: "icon_001",
    avatarState: { status: "active", visibility: "public" },
    walletAddress: "9Zb1qJxP2V6kHrAq8VJrwB6aQ9Xt3Fj1XcS3KoYuE81J",
    bio: "ファッションブランドLUMINAのブランドマネージャー。",
    website: "https://lumina.example.com",
    createdAt: "2024-03-01T10:00:00Z",
    updatedAt: "2024-04-01T12:00:00Z",
    deletedAt: null,
  },
  {
    id: "avatar_002",
    userId: "user_002",
    avatarName: "Creative Studio",
    avatarIconId: "icon_002",
    avatarState: { status: "inactive", visibility: "private" },
    walletAddress: "7jFh2zA9xHYvC2kWu1qKQ4bJf6Nt7PzqXo1qTYuE7RrH",
    bio: "クリエイティブデザインとNFT制作を担当。",
    website: "https://creative.example.com",
    createdAt: "2024-02-15T09:30:00Z",
    updatedAt: "2024-03-05T15:00:00Z",
    deletedAt: null,
  },
  {
    id: "avatar_003",
    userId: "user_003",
    avatarName: "Artisan Factory",
    avatarIconId: "icon_003",
    avatarState: { status: "active", visibility: "public" },
    walletAddress: "4jPk2rDzxHYvV8qWu1qPQ7dFg6Ny8BzqYo8qVYtE9RsP",
    bio: "職人による高品質縫製と染色を提供する工房。",
    website: "https://artisan.example.com",
    createdAt: "2024-01-10T08:00:00Z",
    updatedAt: "2024-04-10T09:00:00Z",
    deletedAt: null,
  },
  {
    id: "avatar_004",
    userId: "user_004",
    avatarName: "LUMINA Customer Support",
    avatarIconId: "icon_004",
    avatarState: { status: "active", visibility: "internal" },
    walletAddress: "8fTg1cBxP3FkHrAq9LJrwB3zQ8Xt2Fj2LcS3KoYiE61T",
    bio: "顧客対応およびブランド品質管理を担当。",
    website: "https://support.lumina.example.com",
    createdAt: "2024-03-10T12:00:00Z",
    updatedAt: "2024-04-15T13:30:00Z",
    deletedAt: null,
  },
  {
    id: "avatar_005",
    userId: "user_005",
    avatarName: "Narratives Admin",
    avatarIconId: "icon_005",
    avatarState: { status: "superuser", visibility: "hidden" },
    walletAddress: "6Rz1kYxQ5VcHrAq7GJrwB2aM9Xt1Fj8LcS3KoYqP2R9H",
    bio: "Narratives プラットフォーム全体を管理する管理者。",
    website: "https://narratives.example.com",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-05-01T00:00:00Z",
    deletedAt: null,
  },
];

/**
 * Avatar 検索ユーティリティ
 */
export function getAvatarById(id: string): Avatar | undefined {
  return AVATARS.find((a) => a.id === id);
}

/**
 * UserID から Avatar を検索
 */
export function getAvatarByUserId(userId: string): Avatar[] {
  return AVATARS.filter((a) => a.userId === userId);
}
