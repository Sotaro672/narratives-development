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
 * SignedIconUpload
 * TokenBlueprint 作成レスポンスに embed される「署名付き PUT URL」情報（方針A）
 */
export type SignedIconUpload = {
  uploadUrl: string;
  objectPath: string; // 例: "{tokenBlueprintId}/icon"
  publicUrl: string; // 例: https://storage.googleapis.com/<bucket>/{tokenBlueprintId}/icon
  expiresAt?: string;
  contentType?: string; // PUT 時に一致必須
};

/**
 * TokenBlueprint
 * backend/internal/domain/tokenBlueprint/entity.go の TokenBlueprint に対応。
 *
 * - 日付は ISO8601 文字列として表現
 * - camelCase 命名に揃える
 * - contentFiles は添付ファイルなどの ID 配列
 *
 * ★変更:
 * - minted は boolean（"notYet"/"minted" は廃止）
 */
export interface TokenBlueprint {
  /** Firestore docId / 作成前のドラフトでは空文字の場合がある */
  id: string;

  name: string;
  symbol: string; // /^[A-Z0-9]{1,10}$/ を想定
  brandId: string;

  /** ブランド表示名（backend で解決された任意のラベル） */
  brandName?: string;

  /** 説明（空でも許容する運用があり得る） */
  description: string;

  /** token_icons 等の ID（任意） */
  iconId?: string | null;

  /** backend が解決して返す icon URL（任意） */
  iconUrl?: string;

  /** create レスポンスで返る署名付きURL情報（任意） */
  iconUpload?: SignedIconUpload;

  /** 関連コンテンツファイルの ID 一覧（空文字禁止） */
  contentFiles: string[];

  /** 担当者 Member ID（必須） */
  assigneeId: string;

  /** 担当者表示名（backend で解決されたフルネームなど、任意） */
  assigneeName?: string;

  /** ミント済みか（未設定/旧データは false 扱いに寄せる） */
  minted?: boolean;

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

/**
 * TokenBlueprint の簡易バリデーション（Go側 validate() と概ね対応）
 *
 * ★注意:
 * - 新規作成前は id が空文字でも動くため、ここでは id の必須チェックを外しています。
 * - description は空文字でも運用上許容し得るため、必須チェックを外しています。
 */
export function validateTokenBlueprint(tb: TokenBlueprint): string[] {
  const errors: string[] = [];

  if (!tb.name?.trim()) errors.push("name is required");

  if (!tb.symbol?.trim()) {
    errors.push("symbol is required");
  } else if (!/^[A-Z0-9]{1,10}$/.test(tb.symbol)) {
    errors.push("symbol must match ^[A-Z0-9]{1,10}$");
  }

  if (!tb.brandId?.trim()) errors.push("brandId is required");
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

  // minted: あれば boolean
  if (tb.minted !== undefined && typeof tb.minted !== "boolean") {
    errors.push("minted must be boolean if set");
  }

  // iconUpload: あれば最低限 uploadUrl/objectPath/publicUrl
  if (tb.iconUpload !== undefined) {
    const u = tb.iconUpload as any;
    if (!String(u?.uploadUrl ?? "").trim()) errors.push("iconUpload.uploadUrl is required");
    if (!String(u?.objectPath ?? "").trim()) errors.push("iconUpload.objectPath is required");
    if (!String(u?.publicUrl ?? "").trim()) errors.push("iconUpload.publicUrl is required");
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
 * - minted が未指定の場合は false をデフォルトとする
 */
export function createTokenBlueprint(
  input: Omit<TokenBlueprint, "contentFiles"> & {
    contentFiles?: string[];
  },
): TokenBlueprint {
  const iconId = input.iconId && input.iconId.trim() ? input.iconId.trim() : null;

  const minted: boolean = typeof input.minted === "boolean" ? input.minted : false;

  return {
    ...input,
    iconId,
    minted,
    contentFiles: normalizeContentFiles(input.contentFiles ?? []),
  };
}
