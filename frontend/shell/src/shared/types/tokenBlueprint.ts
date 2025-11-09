// frontend/shell/src/shared/types/tokenBlueprint.ts

/**
 * ContentFileType
 * backend/internal/domain/tokenBlueprint/entity.go の ContentFileType に対応。
 *
 * - "image" | "video" | "pdf" | "document"
 */
export type ContentFileType = "image" | "video" | "pdf" | "document";

/**
 * ContentFile
 * backend/internal/domain/tokenBlueprint/entity.go の ContentFile に対応。
 *
 * TokenBlueprint とは別エンティティ（例: tokenIcon や添付ファイル管理）で
 * 利用される想定のメタ情報。
 */
export interface ContentFile {
  id: string;
  name: string;
  type: ContentFileType;
  url: string;
  /** ファイルサイズ（bytes） */
  size: number;
}

/**
 * TokenBlueprint
 * backend/internal/domain/tokenBlueprint/entity.go の TokenBlueprint に対応。
 *
 * - 日付は ISO8601 文字列として表現
 * - camelCase 命名に揃える
 * - contentFiles は添付ファイルなどの ID 配列
 */
export interface TokenBlueprint {
  id: string;
  name: string;
  symbol: string; // /^[A-Z0-9]{1,10}$/ を想定
  brandId: string;
  description: string;

  /** token_icons 等の ID（任意） */
  iconId?: string | null;

  /** 関連コンテンツファイルの ID 一覧（空文字禁止） */
  contentFiles: string[];

  /** 担当者 Member ID（必須） */
  assigneeId: string;

  /** 作成情報 */
  createdAt: string; // ISO8601
  createdBy: string;

  /** 更新情報 */
  updatedAt: string; // ISO8601
  updatedBy: string;

  /** 論理削除情報 */
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/* =========================================================
 * ユーティリティ / バリデーション
 * =======================================================*/

/** ContentFileType の妥当性チェック */
export function isValidContentFileType(t: string): t is ContentFileType {
  return t === "image" || t === "video" || t === "pdf" || t === "document";
}

/** ContentFile の簡易バリデーション（backend の Validate と整合） */
export function validateContentFile(file: ContentFile): string[] {
  const errors: string[] = [];

  if (!file.id?.trim()) errors.push("contentFile.id is required");
  if (!file.name?.trim()) errors.push("contentFile.name is required");
  if (!isValidContentFileType(file.type)) {
    errors.push(
      "contentFile.type must be one of 'image' | 'video' | 'pdf' | 'document'",
    );
  }
  if (file.size < 0) {
    errors.push("contentFile.size must be >= 0");
  }

  return errors;
}

/** TokenBlueprint の簡易バリデーション（Go側 validate() と概ね対応） */
export function validateTokenBlueprint(tb: TokenBlueprint): string[] {
  const errors: string[] = [];

  if (!tb.id?.trim()) errors.push("id is required");
  if (!tb.name?.trim()) errors.push("name is required");
  if (!tb.symbol?.trim()) {
    errors.push("symbol is required");
  } else if (!/^[A-Z0-9]{1,10}$/.test(tb.symbol)) {
    errors.push("symbol must match ^[A-Z0-9]{1,10}$");
  }
  if (!tb.brandId?.trim()) errors.push("brandId is required");
  if (!tb.description?.trim()) errors.push("description is required");
  if (!tb.assigneeId?.trim()) errors.push("assigneeId is required");
  if (!tb.createdBy?.trim()) errors.push("createdBy is required");
  if (!tb.createdAt?.trim()) errors.push("createdAt is required");
  if (!tb.updatedBy?.trim()) errors.push("updatedBy is required");
  if (!tb.updatedAt?.trim()) errors.push("updatedAt is required");

  if (tb.iconId !== undefined && tb.iconId !== null) {
    if (!tb.iconId.trim()) {
      errors.push("iconId, if set, must not be empty");
    }
  }

  // contentFiles: 空文字禁止
  for (const id of tb.contentFiles || []) {
    if (!id || !id.trim()) {
      errors.push("contentFiles must not contain empty id");
      break;
    }
  }

  return errors;
}

/**
 * contentFiles の重複/空文字を排除する正規化ヘルパ
 */
export function normalizeContentFiles(ids: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of ids || []) {
    const id = raw.trim();
    if (!id || seen.has(id)) continue;
    seen.add(id);
    out.push(id);
  }
  return out;
}

/**
 * TokenBlueprint 作成用ヘルパ
 * - 文字列トリム
 * - contentFiles 正規化
 * - iconId の空文字 → null
 */
export function createTokenBlueprint(
  input: Omit<TokenBlueprint, "contentFiles"> & {
    contentFiles?: string[];
  },
): TokenBlueprint {
  const iconId =
    input.iconId && input.iconId.trim()
      ? input.iconId.trim()
      : null;

  return {
    ...input,
    iconId,
    contentFiles: normalizeContentFiles(input.contentFiles ?? []),
  };
}
