// frontend/shell/src/shared/types/announcementAttachment.ts
// (Generated from frontend/announcement/src/domain/entity/announcementAttachment.ts
//  & backend/internal/domain/announcementAttachment/entity.go)

/**
 * Default GCS bucket for announcement attachments.
 * backend の DefaultBucket と同期。
 */
export const ANNOUNCEMENT_ATTACHMENT_DEFAULT_BUCKET =
  "narratives_development_announcement_attachment";

/**
 * GCSDeleteOp
 * GCS 上のオブジェクト削除指示用。
 */
export interface GCSDeleteOp {
  bucket: string;
  objectPath: string;
}

/**
 * AttachmentFile
 * backend/internal/domain/announcementAttachment/entity.go に対応する共通型。
 *
 * Announcement 側は attachments にこの id を列挙して参照する想定。
 */
export interface AttachmentFile {
  announcementId: string;
  id: string; // announcementId + fileName から決定的に生成される安定 ID
  fileName: string;
  fileUrl: string;
  fileSize: number;
  mimeType: string;
  bucket: string;
  objectPath: string;
}

/**
 * Policy（backend と同期）
 */

// サイズ制約（0 は上限チェック無効）
export const ANNOUNCEMENT_ATTACHMENT_MIN_FILE_SIZE_BYTES = 1;
export const ANNOUNCEMENT_ATTACHMENT_MAX_FILE_SIZE_BYTES =
  50 * 1024 * 1024; // 50MB

// ファイル名長（0 は無効）
export const ANNOUNCEMENT_ATTACHMENT_MAX_FILE_NAME_LENGTH = 255;

// 許可 MIME タイプ（空配列 = MIME 形式のみチェック）
export const ANNOUNCEMENT_ATTACHMENT_ALLOWED_MIME_TYPES: string[] = [
  "application/pdf",
  "image/jpeg",
  "image/png",
  "image/webp",
  "image/gif",
  "text/plain",
];

// URL ホスト許可リスト（空配列 = 全許可）
export const ANNOUNCEMENT_ATTACHMENT_ALLOWED_URL_HOSTS: string[] = [];

// MIME 形式チェック用
const MIME_REGEX = /^[a-zA-Z0-9.+-]+\/[a-zA-Z0-9.+-]+$/;

/**
 * buildObjectPath
 * GCS 上の標準パスを構築:
 * announcements/{announcementId}/{fileName}
 */
export function buildObjectPath(
  announcementId: string,
  fileName: string
): string {
  const aid = announcementId.trim();
  const fn = fileName.trim();
  if (!aid) {
    throw new Error("announcementAttachment: invalid announcementId");
  }
  if (!fn) {
    throw new Error("announcementAttachment: invalid fileName");
  }
  return ["announcements", aid, fn].join("/");
}

/**
 * makeAttachmentId
 * backend 実装の SHA1 ベース ID を模した安定 ID 生成。
 * ここでは「同一入力に対して決定的に同じ値になる」ことを目的とした簡易実装。
 */
export function makeAttachmentId(
  announcementId: string,
  fileName: string
): string {
  const aid = announcementId.trim().toLowerCase();
  const fn = fileName.trim();
  const src = `${aid}:${fn}`;

  let h1 = 0;
  let h2 = 0;
  for (let i = 0; i < src.length; i++) {
    const c = src.charCodeAt(i);
    h1 = (h1 * 31 + c) | 0;
    h2 = (h2 * 17 + c) | 0;
  }
  const toHex = (n: number) =>
    (n >>> 0).toString(16).padStart(8, "0").slice(0, 8);

  return `${toHex(h1)}${toHex(h2)}`;
}

/**
 * AttachmentFile の簡易バリデーション
 * backend の validateAttachmentFile と概ね整合。
 */
export function validateAttachmentFile(file: AttachmentFile): boolean {
  // announcementId
  if (!file.announcementId?.trim()) return false;

  // fileName
  const name = file.fileName?.trim();
  if (!name) return false;
  if (
    ANNOUNCEMENT_ATTACHMENT_MAX_FILE_NAME_LENGTH > 0 &&
    [...name].length > ANNOUNCEMENT_ATTACHMENT_MAX_FILE_NAME_LENGTH
  ) {
    return false;
  }

  // id（決定的生成の結果と一致しているか）
  if (!file.id?.trim()) return false;
  if (file.id !== makeAttachmentId(file.announcementId, file.fileName)) {
    return false;
  }

  // fileUrl
  if (!isValidUrl(file.fileUrl)) return false;

  // fileSize
  if (!Number.isFinite(file.fileSize)) return false;
  if (file.fileSize < ANNOUNCEMENT_ATTACHMENT_MIN_FILE_SIZE_BYTES) {
    return false;
  }
  if (
    ANNOUNCEMENT_ATTACHMENT_MAX_FILE_SIZE_BYTES > 0 &&
    file.fileSize > ANNOUNCEMENT_ATTACHMENT_MAX_FILE_SIZE_BYTES
  ) {
    return false;
  }

  // mimeType
  const mt = file.mimeType?.trim();
  if (!mt || !MIME_REGEX.test(mt)) return false;
  if (
    ANNOUNCEMENT_ATTACHMENT_ALLOWED_MIME_TYPES.length > 0 &&
    !ANNOUNCEMENT_ATTACHMENT_ALLOWED_MIME_TYPES.includes(mt)
  ) {
    return false;
  }

  // bucket
  if (!file.bucket?.trim()) return false;

  // objectPath
  const op = file.objectPath?.trim();
  if (!op) return false;

  // objectPath が buildObjectPath の結果と整合しているか（先頭の / は許容）
  const expected = buildObjectPath(file.announcementId, file.fileName);
  if (op.replace(/^\/+/, "") !== expected) return false;

  return true;
}

/**
 * GCS 削除オペレーション生成
 */
export function toGCSDeleteOp(file: AttachmentFile): GCSDeleteOp {
  return {
    bucket:
      file.bucket?.trim() || ANNOUNCEMENT_ATTACHMENT_DEFAULT_BUCKET,
    objectPath: file.objectPath.trim().replace(/^\/+/, ""),
  };
}

export function buildGCSDeleteOps(
  files: AttachmentFile[]
): GCSDeleteOp[] {
  return files
    .map(toGCSDeleteOp)
    .filter((op) => op.bucket && op.objectPath);
}

export function buildGCSDeleteOpsFromFileNames(
  announcementId: string,
  fileNames: string[]
): GCSDeleteOp[] {
  const aid = announcementId.trim();
  if (!aid) return [];
  return fileNames
    .map((name) => name.trim())
    .filter(Boolean)
    .map((fileName) => ({
      bucket: ANNOUNCEMENT_ATTACHMENT_DEFAULT_BUCKET,
      objectPath: buildObjectPath(aid, fileName),
    }));
}

// ==============================
// Helpers (module private)
// ==============================

function isValidUrl(raw: string): boolean {
  const s = raw.trim();
  if (!s) return false;
  try {
    const u = new URL(s);
    if (!u.protocol || !u.hostname) return false;

    if (ANNOUNCEMENT_ATTACHMENT_ALLOWED_URL_HOSTS.length > 0) {
      const host = u.hostname.toLowerCase();
      return ANNOUNCEMENT_ATTACHMENT_ALLOWED_URL_HOSTS.some(
        (allowed) => allowed.toLowerCase() === host
      );
    }

    return true;
  } catch {
    return false;
  }
}
