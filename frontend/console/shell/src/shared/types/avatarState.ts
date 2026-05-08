// frontend/shell/src/shared/types/avatarState.ts
// (Generated from frontend/inquiry/src/domain/entity/avatarState.ts
//  and backend/internal/domain/avatarState/entity.go)
//
// Shared cross-frontend type for AvatarState.
//
// Backend/TS spec:
//
// export interface AvatarState {
//   id?: string;
//   avatarId: string;
//   followerCount?: number;
//   followingCount?: number;
//   postCount?: number;
//   lastActiveAt: Date | string;
//   updatedAt?: Date | string;
// }

export interface AvatarState {
  id?: string;
  avatarId: string;
  followerCount?: number;
  followingCount?: number;
  postCount?: number;
  lastActiveAt: Date | string;
  updatedAt?: Date | string;
}

/**
 * Domain-aligned policy
 */
export const AVATAR_STATE_MIN_COUNT = 0;

/**
 * Validate AvatarState (aligned with backend AvatarState.validate).
 */
export function validateAvatarState(state: AvatarState): boolean {
  // avatarId
  if (!state.avatarId || !state.avatarId.trim()) return false;

  // non-negative counts if provided
  if (!countOK(state.followerCount)) return false;
  if (!countOK(state.followingCount)) return false;
  if (!countOK(state.postCount)) return false;

  // lastActiveAt required & valid
  const lastActive = toDate(state.lastActiveAt);
  if (!lastActive) return false;

  // updatedAt optional, but if present must be >= lastActiveAt
  if (state.updatedAt != null) {
    const updated = toDate(state.updatedAt);
    if (!updated) return false;
    if (updated.getTime() < lastActive.getTime()) return false;
  }

  return true;
}

// ==============================
// Helpers
// ==============================

function countOK(v: number | undefined): boolean {
  if (v == null) return true;
  return Number.isFinite(v) && v >= AVATAR_STATE_MIN_COUNT;
}

function toDate(v: Date | string): Date | null {
  if (v instanceof Date) {
    return isNaN(v.getTime()) ? null : v;
  }
  const s = `${v}`.trim();
  if (!s) return null;
  const d = new Date(s);
  return isNaN(d.getTime()) ? null : d;
}
