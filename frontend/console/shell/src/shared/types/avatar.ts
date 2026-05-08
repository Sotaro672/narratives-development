// frontend/shell/src/shared/types/avatar.ts
// (Generated from frontend/inquiry/src/domain/entity/avatar.ts
//  and backend/internal/domain/avatar/entity.go)

/**
 * AvatarState
 *
 * backend/internal/domain/avatarState 由来の状態表現。
 * ここでは汎用的な形にしておき、実際の構造は avatarState ドメイン側に委譲します。
 * 必要に応じて拡張してください。
 */
export interface AvatarState {
  // 任意の状態フィールドを許容（ドメイン側の実装と同期させる想定）
  [key: string]: unknown;
}

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
  avatarState: AvatarState;
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
