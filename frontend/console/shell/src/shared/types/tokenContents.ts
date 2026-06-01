// frontend/shell/src/shared/types/tokenContents.ts

/**
 * Token Contents Type Definitions (Shared)
 *
 * Firebase Storage 移行後の正仕様:
 * - url: Firebase Storage downloadURL
 * - backend/domain では objectPath / name / size を保持しない
 * - backend は url だけで content file を制御する
 */

/** ContentType definition */
export type ContentType = "image" | "video" | "pdf" | "document";

/** ContentType validation helper */
export function isValidContentType(t: string): t is ContentType {
  return t === "image" || t === "video" || t === "pdf" || t === "document";
}

/** Available content types for UI */
export const ALL_CONTENT_TYPES: ContentType[] = [
  "image",
  "video",
  "pdf",
  "document",
];

/** ContentVisibility definition */
export type ContentVisibility = "private" | "public";

/** ContentVisibility validation helper */
export function isValidContentVisibility(v: string): v is ContentVisibility {
  return v === "private" || v === "public";
}

/**
 * Firebase Storage token content metadata
 *
 * backend ContentFile struct と一致:
 * - id
 * - type
 * - contentType
 * - url
 * - visibility
 * - createdAt
 * - createdBy
 * - updatedAt
 * - updatedBy
 */
export interface FirebaseStorageTokenContent {
  id: string;
  type: ContentType;
  contentType: string;
  url: string;
  visibility: ContentVisibility;

  createdAt?: string;
  createdBy?: string;
  updatedAt?: string;
  updatedBy?: string;
}

/**
 * Delete operation target
 *
 * objectPath は廃止済み。
 * 削除・制御は backend 側で url を基準に行う。
 */
export interface FirebaseStorageDeleteOp {
  url: string;
}

/** Validation logic for FirebaseStorageTokenContent */
export function validateFirebaseStorageTokenContent(
  c: FirebaseStorageTokenContent,
): string[] {
  const errors: string[] = [];

  if (!c.id.trim()) {
    errors.push("id is required");
  }

  if (!isValidContentType(c.type)) {
    errors.push("type must be one of 'image' | 'video' | 'pdf' | 'document'");
  }

  if (c.contentType && !c.contentType.trim()) {
    errors.push("contentType must not be blank when provided");
  }

  if (!c.url.trim()) {
    errors.push("url is required");
  } else {
    try {
      const u = new URL(c.url);

      if (!/^https?:$/i.test(u.protocol)) {
        errors.push("url must be http(s)");
      }
    } catch {
      errors.push("url must be a valid URL");
    }
  }

  if (!isValidContentVisibility(c.visibility)) {
    errors.push("visibility must be one of 'private' | 'public'");
  }

  return errors;
}

/** Factory with validation */
export function createFirebaseStorageTokenContent(
  input: FirebaseStorageTokenContent,
): FirebaseStorageTokenContent {
  const normalized: FirebaseStorageTokenContent = {
    id: input.id.trim(),
    type: input.type,
    contentType: input.contentType.trim(),
    url: input.url.trim(),
    visibility: input.visibility,

    createdAt: input.createdAt?.trim(),
    createdBy: input.createdBy?.trim(),
    updatedAt: input.updatedAt?.trim(),
    updatedBy: input.updatedBy?.trim(),
  };

  const errors = validateFirebaseStorageTokenContent(normalized);

  if (errors.length > 0) {
    throw new Error(
      `Invalid FirebaseStorageTokenContent: ${errors.join(", ")}`,
    );
  }

  return normalized;
}

/** Convert FirebaseStorageTokenContent to FirebaseStorageDeleteOp */
export function toFirebaseStorageDeleteOp(
  content: FirebaseStorageTokenContent,
): FirebaseStorageDeleteOp {
  return {
    url: content.url,
  };
}