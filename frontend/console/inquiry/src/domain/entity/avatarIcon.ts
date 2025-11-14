// frontend/inquiry/src/domain/entity/avatarIcon.ts
// Mirrors backend/internal/domain/avatarIcon/entity.go

/**
 * AvatarIcon
 * backend/internal/domain/avatarIcon/entity.go に対応。
 */
export interface AvatarIcon {
  id: string;
  avatarId?: string;
  url: string;
  fileName?: string;
  size?: number;
}

/**
 * 定数（バックエンドポリシーに対応）
 */
export const AVATAR_ICON_POLICY = {
  DEFAULT_BUCKET: "narratives_development_avatar_icon",
  MAX_FILE_SIZE: 10 * 1024 * 1024, // 10MB
  ALLOWED_EXTENSIONS: [".png", ".jpg", ".jpeg", ".webp", ".gif"],
} as const;

// ※ 型を広げて includes で string を受けられるようにする
const ALLOWED_EXTENSIONS: readonly string[] =
  AVATAR_ICON_POLICY.ALLOWED_EXTENSIONS;

/**
 * ドメインエラー
 */
export const AVATAR_ICON_ERRORS = {
  invalidID: "avatarIcon: invalid id",
  invalidURL: "avatarIcon: invalid url",
  invalidFileName: "avatarIcon: invalid fileName",
  invalidSize: "avatarIcon: invalid size",
} as const;

/**
 * 文字列トリムと空文字除去
 */
function normalizeStr(v?: string | null): string | undefined {
  if (!v) return undefined;
  const t = v.trim();
  return t === "" ? undefined : t;
}

/**
 * URLの妥当性チェック
 */
function isValidUrl(u: string): boolean {
  try {
    const parsed = new URL(u);
    return !!parsed.protocol && !!parsed.host;
  } catch {
    return false;
  }
}

/**
 * ファイル拡張子チェック
 */
function isAllowedExtension(name: string): boolean {
  const dotIndex = name.lastIndexOf(".");
  if (dotIndex === -1) return false;
  const ext = name.slice(dotIndex).toLowerCase();
  // ここで readonly string[] に対して判定するのでエラーにならない
  return ALLOWED_EXTENSIONS.includes(ext);
}

/**
 * PublicURL を生成
 * https://storage.googleapis.com/{bucket}/{objectPath}
 */
export function publicURL(bucket: string, objectPath: string): string {
  const b = bucket.trim() || AVATAR_ICON_POLICY.DEFAULT_BUCKET;
  const obj = objectPath.trim().replace(/^\/+/, "");
  return `https://storage.googleapis.com/${b}/${obj}`;
}

/**
 * AvatarIcon エンティティ生成
 */
export function createAvatarIcon(input: AvatarIcon): AvatarIcon {
  const normalized: AvatarIcon = {
    id: input.id.trim(),
    avatarId: normalizeStr(input.avatarId),
    url: input.url.trim(),
    fileName: normalizeStr(input.fileName),
    size: input.size,
  };

  validateAvatarIcon(normalized);
  return normalized;
}

/**
 * AvatarIcon バリデーション
 */
export function validateAvatarIcon(icon: AvatarIcon): void {
  if (!icon.id) throw new Error(AVATAR_ICON_ERRORS.invalidID);
  if (!isValidUrl(icon.url)) throw new Error(AVATAR_ICON_ERRORS.invalidURL);

  if (icon.fileName && !isAllowedExtension(icon.fileName)) {
    throw new Error(AVATAR_ICON_ERRORS.invalidFileName);
  }

  if (
    icon.size !== undefined &&
    (icon.size < 0 || icon.size > AVATAR_ICON_POLICY.MAX_FILE_SIZE)
  ) {
    throw new Error(AVATAR_ICON_ERRORS.invalidSize);
  }
}

/**
 * GCSDeleteOp: GCS上の削除操作を表す型。
 */
export interface GCSDeleteOp {
  bucket: string;
  objectPath: string;
}

/**
 * AvatarIcon から GCS削除操作を生成。
 * objectPath は呼び出し元で Build 済みを想定。
 */
export function toGCSDeleteOp(
  icon: AvatarIcon,
  objectPath: string,
  bucket?: string
): GCSDeleteOp {
  return {
    bucket: (bucket ?? AVATAR_ICON_POLICY.DEFAULT_BUCKET).trim(),
    objectPath: objectPath.trim().replace(/^\/+/, ""),
  };
}
