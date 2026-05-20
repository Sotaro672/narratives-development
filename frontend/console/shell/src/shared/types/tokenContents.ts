// frontend/shell/src/shared/types/tokenContents.ts

/**
 * Token Contents Type Definitions (Shared)
 *
 * Firebase Storage 移行後の正仕様:
 * - url: Firebase Storage downloadURL
 * - objectPath: Firebase Storage object path
 * - GCS bucket / public GCS URL / signed URL は扱わない
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

/** Firebase Storage token content metadata */
export interface FirebaseStorageTokenContent {
  id: string;
  name: string;
  type: ContentType;
  contentType: string;
  size: number;
  objectPath: string;
  url: string;
}

/** Delete operation target in Firebase Storage */
export interface FirebaseStorageDeleteOp {
  objectPath: string;
}

/** Upload policy */
export const MAX_TOKEN_CONTENT_FILE_SIZE = 50 * 1024 * 1024; // 50MB

/** Validation logic for FirebaseStorageTokenContent */
export function validateFirebaseStorageTokenContent(
  c: FirebaseStorageTokenContent,
): string[] {
  const errors: string[] = [];

  if (!c.id.trim()) errors.push("id is required");
  if (!c.name.trim()) errors.push("name is required");

  if (!isValidContentType(c.type)) {
    errors.push("type must be one of 'image' | 'video' | 'pdf' | 'document'");
  }

  if (!c.contentType.trim()) {
    errors.push("contentType is required");
  }

  if (c.size < 0) {
    errors.push("size must be >= 0");
  } else if (
    MAX_TOKEN_CONTENT_FILE_SIZE > 0 &&
    c.size > MAX_TOKEN_CONTENT_FILE_SIZE
  ) {
    errors.push(`size must be <= ${MAX_TOKEN_CONTENT_FILE_SIZE} bytes`);
  }

  if (!c.objectPath.trim()) {
    errors.push("objectPath is required");
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

  return errors;
}

/** Factory with validation */
export function createFirebaseStorageTokenContent(
  input: FirebaseStorageTokenContent,
): FirebaseStorageTokenContent {
  const errors = validateFirebaseStorageTokenContent(input);

  if (errors.length > 0) {
    throw new Error(
      `Invalid FirebaseStorageTokenContent: ${errors.join(", ")}`,
    );
  }

  return input;
}

/** Convert FirebaseStorageTokenContent to FirebaseStorageDeleteOp */
export function toFirebaseStorageDeleteOp(
  content: FirebaseStorageTokenContent,
): FirebaseStorageDeleteOp {
  return {
    objectPath: content.objectPath,
  };
}