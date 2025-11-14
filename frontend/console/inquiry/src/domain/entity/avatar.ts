// frontend/inquiry/src/domain/entity/avatar.ts

/**
 * Avatar domain entity (frontend)
 *
 * Mirrors backend/internal/domain/avatar/entity.go
 * and conforms to web shared types.
 *
 * Includes references to:
 * - avatarState.AvatarState (domain state type)
 * - user and wallet domain links (IDs)
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
  deletedAt?: Date | string;
}

/**
 * AvatarState mirrors backend/internal/domain/avatarState/entity.go
 * (simplified stub until the full type is imported)
 */
export interface AvatarState {
  state?: string;
  status?: string;
  lastSyncAt?: Date | string;
}

/**
 * Domain policy constants
 */
export const AVATAR_POLICY = {
  MAX_AVATAR_NAME_LENGTH: 50,
  MAX_BIO_LENGTH: 1000,
} as const;

/**
 * Domain errors
 */
export const AVATAR_ERRORS = {
  invalidID: "avatar: invalid id",
  invalidUserID: "avatar: invalid userId",
  invalidAvatarName: "avatar: invalid avatarName",
  invalidBio: "avatar: invalid bio",
  invalidWebsite: "avatar: invalid website",
  invalidCreatedAt: "avatar: invalid createdAt",
  invalidUpdatedAt: "avatar: invalid updatedAt",
  invalidDeletedAt: "avatar: invalid deletedAt",
  invalidWalletAddressLink: "avatar: invalid walletAddress link",
} as const;

/**
 * Utility: normalize optional string
 */
function normalizeOptionalString(
  v?: string | null
): string | undefined {
  if (v == null) return undefined;
  const t = `${v}`.trim();
  return t === "" ? undefined : t;
}

/**
 * Utility: validate URL
 */
function validateWebsite(urlStr: string): boolean {
  try {
    const u = new URL(urlStr.trim());
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}

/**
 * Validate Avatar domain entity
 */
export function validateAvatar(a: Avatar): void {
  if (!a.id || !a.id.trim()) throw new Error(AVATAR_ERRORS.invalidID);
  if (!a.userId || !a.userId.trim())
    throw new Error(AVATAR_ERRORS.invalidUserID);
  if (!a.avatarName || a.avatarName.trim().length === 0)
    throw new Error(AVATAR_ERRORS.invalidAvatarName);
  if (a.avatarName.length > AVATAR_POLICY.MAX_AVATAR_NAME_LENGTH)
    throw new Error(AVATAR_ERRORS.invalidAvatarName);

  if (a.bio && a.bio.length > AVATAR_POLICY.MAX_BIO_LENGTH)
    throw new Error(AVATAR_ERRORS.invalidBio);

  if (a.website && !validateWebsite(a.website))
    throw new Error(AVATAR_ERRORS.invalidWebsite);

  const created = new Date(a.createdAt);
  const updated = new Date(a.updatedAt);
  if (isNaN(created.getTime())) throw new Error(AVATAR_ERRORS.invalidCreatedAt);
  if (isNaN(updated.getTime()) || updated < created)
    throw new Error(AVATAR_ERRORS.invalidUpdatedAt);

  if (a.deletedAt) {
    const deleted = new Date(a.deletedAt);
    if (isNaN(deleted.getTime()) || deleted < created)
      throw new Error(AVATAR_ERRORS.invalidDeletedAt);
  }
}

/**
 * Factory to normalize and validate Avatar
 */
export function createAvatar(input: Avatar): Avatar {
  const normalized: Avatar = {
    id: input.id.trim(),
    userId: input.userId.trim(),
    avatarName: input.avatarName.trim(),
    avatarIconId: normalizeOptionalString(input.avatarIconId),
    avatarState: input.avatarState,
    walletAddress: normalizeOptionalString(input.walletAddress),
    bio: normalizeOptionalString(input.bio),
    website: normalizeOptionalString(input.website),
    createdAt: input.createdAt,
    updatedAt: input.updatedAt,
    deletedAt: input.deletedAt,
  };

  validateAvatar(normalized);
  return normalized;
}

/**
 * Sorting options for avatar listing
 */
export type SortBy = "created_at" | "updated_at" | "avatar_name";

/**
 * List filter type (frontend mirror)
 */
export interface AvatarListFilter {
  userId?: string;
  nameContains?: string;
  walletAddress?: string;
  includeDeleted?: boolean;
  limit?: number;
  offset?: number;
  sortBy?: SortBy;
  desc?: boolean;
}

/**
 * Sanitize filter
 */
export function sanitizeAvatarFilter(f: AvatarListFilter): AvatarListFilter {
  return {
    userId: f.userId?.trim() || undefined,
    nameContains: f.nameContains?.trim() || "",
    walletAddress: f.walletAddress?.trim() || undefined,
    includeDeleted: f.includeDeleted ?? false,
    limit: f.limit && f.limit > 0 ? f.limit : 0,
    offset: f.offset && f.offset > 0 ? f.offset : 0,
    sortBy: f.sortBy ?? "created_at",
    desc: f.desc ?? false,
  };
}
