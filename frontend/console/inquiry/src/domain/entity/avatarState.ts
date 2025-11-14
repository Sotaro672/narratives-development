// frontend/inquiry/src/domain/entity/avatarState.ts
// ------------------------------------------------------
// Domain entity for AvatarState (mirrors backend/internal/domain/avatarState/entity.go)
// and shared type definition intention from avatarState.ts.
//
// TS reference:
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
// ------------------------------------------------------

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
 * Policy (mirrors Go-side semantics)
 */
export const AVATAR_STATE_MIN_COUNT = 0;

/**
 * Create a valid AvatarState object with basic normalization.
 * - Trims avatarId
 * - Accepts Date or ISO-8601 strings for lastActiveAt/updatedAt
 */
export function createAvatarState(input: AvatarState): AvatarState {
  const normalized: AvatarState = {
    ...input,
    id: input.id?.trim() || undefined,
    avatarId: input.avatarId.trim(),
    followerCount:
      input.followerCount !== undefined ? Number(input.followerCount) : undefined,
    followingCount:
      input.followingCount !== undefined
        ? Number(input.followingCount)
        : undefined,
    postCount:
      input.postCount !== undefined ? Number(input.postCount) : undefined,
    lastActiveAt: input.lastActiveAt,
    updatedAt: input.updatedAt,
  };

  if (!validateAvatarState(normalized)) {
    throw new Error("Invalid AvatarState");
  }

  return normalized;
}

/**
 * Validation (aligned with backend AvatarState.validate)
 *
 * - avatarId must be non-empty
 * - followerCount / followingCount / postCount must be >= 0 when provided
 * - lastActiveAt must be a valid time (non-zero)
 * - updatedAt, if present, must be >= lastActiveAt
 */
export function validateAvatarState(state: AvatarState): boolean {
  // avatarId
  if (!state.avatarId || !state.avatarId.trim()) {
    return false;
  }

  // counts (non-negative if provided)
  if (!countOK(state.followerCount)) return false;
  if (!countOK(state.followingCount)) return false;
  if (!countOK(state.postCount)) return false;

  // lastActiveAt required & valid
  const lastActive = toDate(state.lastActiveAt);
  if (!lastActive) return false;

  // updatedAt optional but if set must be >= lastActiveAt
  if (state.updatedAt != null) {
    const updated = toDate(state.updatedAt);
    if (!updated) return false;
    if (updated.getTime() < lastActive.getTime()) return false;
  }

  return true;
}

// ============ Helpers ============

function countOK(v: number | undefined): boolean {
  if (v == null) return true;
  return Number.isFinite(v) && v >= AVATAR_STATE_MIN_COUNT;
}

function toDate(v: Date | string): Date | null {
  if (v instanceof Date) {
    return isNaN(v.getTime()) ? null : v;
  }
  const s = String(v).trim();
  if (!s) return null;
  const d = new Date(s);
  if (isNaN(d.getTime())) return null;
  return d;
}
