// frontend/shell/src/shared/types/message.ts

/**
 * MessageStatus
 * backend/internal/domain/message/entity.go の MessageStatus に対応。
 *
 * - "draft"     : 下書き
 * - "sent"      : 送信済み
 * - "canceled"  : 送信キャンセル
 * - "delivered" : 配信完了
 * - "read"      : 既読
 */
export type MessageStatus =
  | "draft"
  | "sent"
  | "canceled"
  | "delivered"
  | "read";

/** MessageStatus の妥当性チェック */
export function isValidMessageStatus(s: string): s is MessageStatus {
  return (
    s === "draft" ||
    s === "sent" ||
    s === "canceled" ||
    s === "delivered" ||
    s === "read"
  );
}

/**
 * ImageRef
 * backend/internal/domain/message/entity.go の ImageRef に対応。
 *
 * - objectPath: 必須 (例: "messages/{messageId}/{fileName}" や "gs://bucket/...")
 * - url: 任意（署名付きURL等）
 * - fileSize: バイト数（0以上）
 * - mimeType: `type/subtype` 形式
 * - uploadedAt: ISO8601 文字列
 */
export interface ImageRef {
  objectPath: string;
  url?: string;
  fileName: string;
  fileSize: number;
  mimeType: string;
  width?: number | null;
  height?: number | null;
  uploadedAt: string;
}

/**
 * Message
 * backend/internal/domain/message/entity.go の Message に対応。
 *
 * - 日付系は全て ISO8601 文字列
 * - images は GCS 参照情報の配列
 */
export interface Message {
  id: string;
  senderId: string;
  receiverId: string;
  content: string;
  status: MessageStatus;
  images: ImageRef[];
  createdAt: string;
  updatedAt?: string | null;
  deletedAt?: string | null;
  readAt?: string | null;
  canceledAt?: string | null;
}

/**
 * MessageDTO
 * backend/internal/domain/message/entity.go の MessageDTO に対応。
 * API レスポンス / 送受信用フォーマットとして利用。
 */
export interface MessageDTO {
  id: string;
  senderId: string;
  receiverId: string;
  content: string;
  status: MessageStatus;
  images?: ImageRef[];
  createdAt: string;
  updatedAt?: string | null;
  deletedAt?: string | null;
  readAt?: string | null;
  canceledAt?: string | null;
}

/**
 * MessageThread
 * backend/internal/domain/message/entity.go の MessageThread に対応。
 * 会話一覧ビュー用のスレッド情報。
 */
export interface MessageThread {
  id: string;
  participantIds: string[];
  lastMessageId: string;
  lastMessageAt: string; // ISO8601
  lastMessageText: string;
  unreadCounts?: Record<string, number>;
  createdAt: string;
  updatedAt?: string | null;
}

/**
 * ImageRef の簡易バリデーション
 * （厳密な URL / MIME 判定は backend に委譲し、フロントでは形式的チェックのみ）
 */
export function validateImageRef(ref: ImageRef): boolean {
  if (!ref.fileName?.trim()) return false;
  if (!ref.objectPath?.trim()) return false;
  if (ref.fileSize < 0) return false;
  if (!/^[a-zA-Z0-9.+-]+\/[a-zA-Z0-9.+-]+$/.test(ref.mimeType)) return false;

  const uploaded = parseIso(ref.uploadedAt);
  if (!uploaded) return false;

  if (ref.url && ref.url.trim()) {
    try {
      // eslint-disable-next-line no-new
      new URL(ref.url);
    } catch {
      return false;
    }
  }

  return true;
}

/**
 * Message の簡易バリデーション
 * backend の validate() ロジックと整合する範囲でチェック。
 */
export function validateMessage(m: Message): boolean {
  if (!m.id?.trim()) return false;
  if (!m.senderId?.trim() || !m.receiverId?.trim()) return false;
  if (!m.content?.trim()) return false;
  if (!isValidMessageStatus(m.status)) return false;

  const created = parseIso(m.createdAt);
  if (!created) return false;

  // images
  for (const img of m.images ?? []) {
    if (!validateImageRef(img)) return false;
  }

  // 時系列
  const updated = m.updatedAt ? parseIso(m.updatedAt) : null;
  const deleted = m.deletedAt ? parseIso(m.deletedAt) : null;
  const read = m.readAt ? parseIso(m.readAt) : null;
  const canceled = m.canceledAt ? parseIso(m.canceledAt) : null;

  if (updated && updated < created) return false;
  if (deleted && deleted < created) return false;
  if (read && read < created) return false;
  if (canceled && canceled < created) return false;

  // ステータスに応じた必須フィールド
  if (m.status === "read" && !read) return false;
  if (m.status === "canceled" && !canceled) return false;

  return true;
}

/**
 * Message -> MessageDTO 変換
 */
export function toMessageDTO(m: Message): MessageDTO {
  return {
    id: m.id,
    senderId: m.senderId,
    receiverId: m.receiverId,
    content: m.content,
    status: m.status,
    images: m.images,
    createdAt: m.createdAt,
    updatedAt: m.updatedAt ?? null,
    deletedAt: m.deletedAt ?? null,
    readAt: m.readAt ?? null,
    canceledAt: m.canceledAt ?? null,
  };
}

/** ISO8601 文字列を Date に変換（失敗時は null） */
function parseIso(s: string | null | undefined): Date | null {
  if (!s) return null;
  const t = Date.parse(s);
  if (Number.isNaN(t)) return null;
  return new Date(t);
}
