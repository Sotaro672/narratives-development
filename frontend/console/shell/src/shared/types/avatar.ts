// frontend/shell/src/shared/types/avatar.ts
// (Generated from frontend/inquiry/src/domain/entity/avatar.ts
//  and backend/internal/domain/avatar/entity.go)

/**
 * Avatar
 *
 * - backend/internal/domain/avatar/entity.go
 * - frontend/inquiry/src/domain/entity/avatar.ts
 * と整合する共通型。
 */
export interface Avatar {
  id: string;
  userId: string;
  avatarName: string;
  avatarIconId?: string;
  walletAddress?: string;
  bio?: string;
  website?: string;
  createdAt: Date | string;
  updatedAt: Date | string;
  deletedAt?: Date | string | null;
}

/**
 * 並び替えキー
 * backend の SortBy と対応。
 */
export type AvatarSortBy = "created_at" | "updated_at" | "avatar_name";

/**
 * Avatar 一覧取得用フィルタ
 * backend の ListFilter と対応（型名のみ Avatar 向けに調整）。
 */
export interface AvatarListFilter {
  userId?: string;
  nameContains?: string;
  walletAddress?: string;
  includeDeleted?: boolean;
  limit?: number;
  offset?: number;
  sortBy?: AvatarSortBy;
  desc?: boolean;
}