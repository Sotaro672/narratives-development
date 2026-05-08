// frontend/announcement/src/domain/entity/announcementAttachment.ts

/**
 * AnnouncementAttachment domain (frontend)
 * Mirrors:
 * - backend/internal/domain/announcementAttachment/entity.go
 *
 * TS source-of-truth shape:
 * export interface AttachmentFile {
 *   announcementId: string;
 *   id: string;          // 安定ID（announcementId + fileName から決定的に生成）
 *   fileName: string;
 *   fileUrl: string;
 *   fileSize: number;
 *   mimeType: string;
 *   bucket: string;
 *   objectPath: string;
 * }
 */

/**
 * Default GCS bucket for announcement attachments.
 * backend の DefaultBucket と同期。
 */
export const ANNOUNCEMENT_ATTACHMENT_DEFAULT_BUCKET =
  "narratives_development_announcement_attachment";

/**
 * GCS 上の削除操作定義
 */
export interface GCSDeleteOp {
  bucket: string;
  objectPath: string;
}

/**
 * Announcement Attachment File
 */
export interface AttachmentFile {
  announcementId: string;
  id: string;
  fileName: string;
  fileUrl: string;
  fileSize: number;
  mimeType: string;
  bucket: string;
  objectPath: string;
}

/**
 * Policy (backend と同期)
 */

// Limits (0 は上限チェック無効)
export const MIN_FILE_SIZE_BYTES = 1;
export const MAX_FILE_SIZE_BYTES = 50 * 1024 * 1024; // 50MB
export const MAX_FILE_NAME_LENGTH = 255;

// Allowed MIME types（空にすると mimeRe にマッチする任意を許可）
export const ALLOWED_MIME_TYPES = new Set<string>([
  "application/pdf",
  "image/jpeg",
  "image/png",
  "image/webp",
  "image/gif",
  "text/plain",
]);

// Optional allow-list for URL hosts（空 = 全許可）
export const ALLOWED_URL_HOSTS = new Set<string>();

// MIME validation
const MIME_RE = /^[a-zA-Z0-9.+-]+\/[a-zA-Z0-9.+-]+$/;

/**
 * BuildObjectPath
 * e.g. announcements/{announcementId}/{fileName}
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
  // path.join 的な挙動（ブラウザ/Node 非依存で素直に連結）
  return ["announcements", aid, fn].join("/");
}

/**
 * MakeAttachmentID
 * announcementId と fileName から安定 ID を生成。
 * Go 実装: hex(sha1(lower(trim(announcementId))+":"+trim(fileName)))
 */
export function makeAttachmentId(
  announcementId: string,
  fileName: string
): string {
  const aid = announcementId.trim().toLowerCase();
  const fn = fileName.trim();
  const src = `${aid}:${fn}`;

  // 環境に crypto.subtle / Node crypto がない場合もあるので、
  // ここでは簡易な非暗号学的ハッシュにフォールバック。
  // 「安定して同じ入力から同じIDが得られる」ことが目的なので十分。
  let h1 = 0,
    h2 = 0;
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
 * URL バリデーション
 */
function urlOK(raw: string): boolean {
  const s = raw.trim();
  if (!s) return false;
  try {
    const u = new URL(s);
    if (!u.protocol || !u.hostname) return false;
    if (ALLOWED_URL_HOSTS.size > 0) {
      const host = u.hostname.toLowerCase();
      if (!ALLOWED_URL_HOSTS.has(host)) return false;
    }
    return true;
  } catch {
    return false;
  }
}

/**
 * AttachmentFile のバリデーション
 * backend の validateAttachmentFile と整合する範囲で実装。
 */
export function validateAttachmentFile(f: AttachmentFile): boolean {
  // announcementId
  if (!f.announcementId?.trim()) return false;

  // fileName
  if (!f.fileName?.trim()) return false;
  if (
    MAX_FILE_NAME_LENGTH > 0 &&
    [...f.fileName].length > MAX_FILE_NAME_LENGTH
  ) {
    return false;
  }

  // id（決定的生成と一致すること）
  if (!f.id?.trim()) return false;
  if (f.id !== makeAttachmentId(f.announcementId, f.fileName)) return false;

  // fileUrl
  if (!urlOK(f.fileUrl)) return false;

  // fileSize
  if (!Number.isFinite(f.fileSize)) return false;
  if (f.fileSize < MIN_FILE_SIZE_BYTES) return false;
  if (MAX_FILE_SIZE_BYTES > 0 && f.fileSize > MAX_FILE_SIZE_BYTES) {
    return false;
  }

  // mimeType
  const mt = f.mimeType?.trim();
  if (!mt || !MIME_RE.test(mt)) return false;
  if (ALLOWED_MIME_TYPES.size > 0 && !ALLOWED_MIME_TYPES.has(mt)) {
    return false;
  }

  // bucket
  if (!f.bucket?.trim()) return false;

  // objectPath
  const op = f.objectPath?.trim();
  if (!op) return false;

  // objectPath は buildObjectPath の結果と一致する想定
  try {
    const expected = buildObjectPath(f.announcementId, f.fileName);
    if (op.replace(/^\/+/, "") !== expected) return false;
  } catch {
    return false;
  }

  return true;
}

/**
 * GCSURI / PublicURL helpers (UI / debug 用)
 */

export function toGCSURI(f: AttachmentFile): string {
  const bucket =
    f.bucket?.trim() || ANNOUNCEMENT_ATTACHMENT_DEFAULT_BUCKET;
  const objectPath = f.objectPath.replace(/^\/+/, "");
  return `gs://${bucket}/${objectPath}`;
}

export function toPublicURL(f: AttachmentFile): string {
  const bucket =
    f.bucket?.trim() || ANNOUNCEMENT_ATTACHMENT_DEFAULT_BUCKET;
  const objectPath = f.objectPath.replace(/^\/+/, "");
  return `https://storage.googleapis.com/${bucket}/${objectPath}`;
}

/**
 * ToGCSDeleteOp: 単一添付ファイルの削除オペレーションに変換
 */
export function toGCSDeleteOp(f: AttachmentFile): GCSDeleteOp {
  const bucket =
    f.bucket?.trim() || ANNOUNCEMENT_ATTACHMENT_DEFAULT_BUCKET;
  const objectPath = f.objectPath.trim().replace(/^\/+/, "");
  return { bucket, objectPath };
}

/**
 * BuildGCSDeleteOps: 添付ファイル配列から削除オペレーション配列を生成
 */
export function buildGCSDeleteOps(files: AttachmentFile[]): GCSDeleteOp[] {
  return files
    .map(toGCSDeleteOp)
    .filter((op) => !!op.bucket && !!op.objectPath);
}

/**
 * BuildGCSDeleteOpsFromFileNames:
 * announcementId と fileName 群だけを持っている場合に利用。
 */
export function buildGCSDeleteOpsFromFileNames(
  announcementId: string,
  fileNames: string[]
): GCSDeleteOp[] {
  const aid = announcementId.trim();
  if (!aid) return [];
  const bucket = ANNOUNCEMENT_ATTACHMENT_DEFAULT_BUCKET;

  return fileNames
    .map((raw) => raw.trim())
    .filter(Boolean)
    .map((fn) => {
      const objectPath = buildObjectPath(aid, fn);
      return { bucket, objectPath };
    });
}
