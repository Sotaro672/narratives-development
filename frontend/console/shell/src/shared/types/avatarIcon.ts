// frontend/shell/src/shared/types/avatarIcon.ts
// ─────────────────────────────────────────────
// Shared type corresponding to backend/internal/domain/avatarIcon/entity.go
// and frontend/inquiry/src/domain/entity/avatarIcon.ts
// ─────────────────────────────────────────────

/**
 * AvatarIcon
 * Represents an uploaded icon file associated with an avatar.
 * Mirrors backend/internal/domain/avatarIcon/entity.go.
 */
export interface AvatarIcon {
  /** Unique identifier of the icon */
  id: string;

  /** Optional reference to the owning Avatar entity */
  avatarId?: string;

  /** Publicly accessible URL (e.g., GCS HTTPS endpoint) */
  url: string;

  /** Optional original filename */
  fileName?: string;

  /** Optional file size in bytes */
  size?: number;
}

/**
 * GCSDeleteOp
 * Represents a deletion operation target in Google Cloud Storage.
 */
export interface GCSDeleteOp {
  bucket: string;
  objectPath: string;
}

/**
 * Constants for validation and constraints.
 * Matches backend/internal/domain/avatarIcon/entity.go.
 */
export const AVATAR_ICON_POLICY = {
  /** Default GCS bucket name for avatar icons */
  DEFAULT_BUCKET: "narratives_development_avatar_icon",

  /** Maximum allowed file size (10 MB) */
  MAX_FILE_SIZE: 10 * 1024 * 1024,

  /** Allowed file extensions */
  ALLOWED_EXTENSIONS: [".png", ".jpg", ".jpeg", ".webp", ".gif"],
} as const;

/**
 * Domain-level error messages for AvatarIcon validation.
 */
export const AVATAR_ICON_ERRORS = {
  invalidID: "avatarIcon: invalid id",
  invalidURL: "avatarIcon: invalid url",
  invalidFileName: "avatarIcon: invalid fileName",
  invalidSize: "avatarIcon: invalid size",
} as const;
