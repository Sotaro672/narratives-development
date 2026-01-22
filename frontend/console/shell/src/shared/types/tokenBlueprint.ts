// frontend/shell/src/shared/types/tokenBlueprint.ts

/**
 * ContentFileType
 * backend/internal/domain/tokenBlueprint/entity.go の ContentFileType に対応。
 *
 * - "image" | "video" | "pdf" | "document"
 */
export type ContentFileType = "image" | "video" | "pdf" | "document";

/**
 * ContentVisibility
 * backend/internal/domain/tokenBlueprint/entity.go の ContentVisibility に対応。
 *
 * - "public" | "private"
 */
export type ContentVisibility = "public" | "private";

/**
 * ContentFile
 * backend/internal/domain/tokenBlueprint/entity.go の ContentFile に対応。
 *
 * TokenBlueprint の contentFiles は「ID配列」ではなく、
 * ContentFile（embedded）の配列が正。
 */
export interface ContentFile {
  id: string;
  name: string;
  type: ContentFileType;

  /** MIME type */
  contentType: string;

  /** ファイルサイズ（bytes） */
  size: number;

  /** GCS object path 等（保存場所の参照） */
  objectPath: string;

  /** 公開範囲 */
  visibility: ContentVisibility;

  /** 作成情報 */
  createdAt?: string; // ISO8601（backend は time.Time）
  createdBy?: string;

  /** 更新情報 */
  updatedAt?: string; // ISO8601（backend は time.Time）
  updatedBy?: string;
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
 *
 * ★変更（entity.go 正）:
 * - iconId は存在しない（削除）
 * - metadataUri は string（追加）
 * - minted は boolean（必須扱いに寄せる）
 * - contentFiles は string[] ではなく ContentFile[]（embedded）
 */
export interface TokenBlueprint {
  /** Firestore docId / 作成前のドラフトでは空文字の場合がある */
  id: string;

  name: string;
  symbol: string; // /^[A-Z0-9]{1,10}$/ を想定
  brandId: string;

  /** companyId（domain 正） */
  companyId: string;

  /** ブランド表示名（backend で解決された任意のラベル） */
  brandName?: string;

  /** 説明（空でも許容する運用があり得る） */
  description: string;

  /** backend が解決して返す icon URL（任意） */
  iconUrl?: string;

  /** create レスポンスで返る署名付きURL情報（任意） */
  iconUpload?: SignedIconUpload;

  /** entity.go 正: embedded content files */
  contentFiles: ContentFile[];

  /** 担当者 Member ID（必須） */
  assigneeId: string;

  /** 担当者表示名（backend で解決されたフルネームなど、任意） */
  assigneeName?: string;

  /** ミント済みか（entity.go 正: bool） */
  minted: boolean;

  /** メタデータ URI（entity.go 正: string） */
  metadataUri: string;

  /** 作成情報 */
  createdAt: string; // ISO8601
  createdBy: string;

  /** 更新情報（未設定の可能性があるなら optional にしても良いが、ここでは現行維持） */
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

/** ContentVisibility の妥当性チェック */
export function isValidContentVisibility(v: string): v is ContentVisibility {
  return v === "public" || v === "private";
}

/** ContentFile の簡易バリデーション（backend の Validate と整合） */
export function validateContentFile(file: ContentFile): string[] {
  const errors: string[] = [];

  if (!file.id?.trim()) errors.push("contentFile.id is required");
  if (!file.name?.trim()) errors.push("contentFile.name is required");

  if (!isValidContentFileType(String(file.type ?? "").trim())) {
    errors.push(
      "contentFile.type must be one of 'image' | 'video' | 'pdf' | 'document'",
    );
  }

  if (!String(file.contentType ?? "").trim()) {
    errors.push("contentFile.contentType is required");
  }

  if (!String(file.objectPath ?? "").trim()) {
    errors.push("contentFile.objectPath is required");
  }

  if (!isValidContentVisibility(String(file.visibility ?? "").trim())) {
    errors.push("contentFile.visibility must be 'public' or 'private'");
  }

  if (Number.isNaN(Number(file.size))) {
    errors.push("contentFile.size must be a number");
  } else if (file.size < 0) {
    errors.push("contentFile.size must be >= 0");
  }

  // createdAt/updatedAt は backend 側で time.Time として扱われるが、UIでは任意
  if (file.createdAt !== undefined && String(file.createdAt).trim() === "") {
    errors.push("contentFile.createdAt, if set, must not be empty");
  }
  if (file.updatedAt !== undefined && String(file.updatedAt).trim() === "") {
    errors.push("contentFile.updatedAt, if set, must not be empty");
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
  if (!tb.companyId?.trim()) errors.push("companyId is required");
  if (!tb.assigneeId?.trim()) errors.push("assigneeId is required");
  if (!tb.createdBy?.trim()) errors.push("createdBy is required");
  if (!tb.createdAt?.trim()) errors.push("createdAt is required");
  if (!tb.updatedBy?.trim()) errors.push("updatedBy is required");
  if (!tb.updatedAt?.trim()) errors.push("updatedAt is required");

  // minted: entity.go 正は bool なので必須（ここでは boolean を要求）
  if (typeof tb.minted !== "boolean") {
    errors.push("minted is required and must be boolean");
  }

  // metadataUri: entity.go 正は string（空許容かどうかは運用次第だが、型としては必須）
  if (tb.metadataUri === undefined || tb.metadataUri === null) {
    errors.push("metadataUri is required (string)");
  } else if (typeof tb.metadataUri !== "string") {
    errors.push("metadataUri must be string");
  }

  // contentFiles: embedded（空配列は許容。要素は Validate）
  if (!Array.isArray(tb.contentFiles)) {
    errors.push("contentFiles must be an array");
  } else {
    for (const f of tb.contentFiles) {
      const ferrs = validateContentFile(f);
      if (ferrs.length > 0) {
        errors.push(...ferrs);
        break;
      }
    }
  }

  // iconUpload: あれば最低限 uploadUrl/objectPath/publicUrl
  if (tb.iconUpload !== undefined) {
    const u = tb.iconUpload as any;
    if (!String(u?.uploadUrl ?? "").trim())
      errors.push("iconUpload.uploadUrl is required");
    if (!String(u?.objectPath ?? "").trim())
      errors.push("iconUpload.objectPath is required");
    if (!String(u?.publicUrl ?? "").trim())
      errors.push("iconUpload.publicUrl is required");
  }

  return errors;
}

/**
 * contentFiles の重複/空IDを排除する正規化ヘルパ（id 基準）
 */
export function normalizeContentFiles(files: ContentFile[]): ContentFile[] {
  const seen = new Set<string>();
  const out: ContentFile[] = [];

  for (const f of files || []) {
    const id = String((f as any)?.id ?? "").trim();
    if (!id || seen.has(id)) continue;
    seen.add(id);

    out.push({
      ...f,
      id,
      name: String((f as any)?.name ?? "").trim(),
      type: String((f as any)?.type ?? "").trim() as ContentFileType,
      contentType: String((f as any)?.contentType ?? "").trim(),
      objectPath: String((f as any)?.objectPath ?? "").trim(),
      visibility:
        (String((f as any)?.visibility ?? "").trim() as ContentVisibility) ||
        "private",
      size: Number((f as any)?.size ?? 0) || 0,
      createdAt:
        (f as any).createdAt != null ? String((f as any).createdAt) : (f as any).createdAt,
      updatedAt:
        (f as any).updatedAt != null ? String((f as any).updatedAt) : (f as any).updatedAt,
      createdBy:
        (f as any).createdBy != null ? String((f as any).createdBy).trim() : (f as any).createdBy,
      updatedBy:
        (f as any).updatedBy != null ? String((f as any).updatedBy).trim() : (f as any).updatedBy,
    });
  }

  return out;
}

/**
 * TokenBlueprint 作成用ヘルパ
 * - 文字列トリム
 * - contentFiles 正規化
 * - minted が未指定の場合は false をデフォルトとする（entity.go 正: bool）
 * - metadataUri が未指定の場合は "" に寄せる（空許容運用の想定）
 */
export function createTokenBlueprint(
  input: Omit<TokenBlueprint, "contentFiles" | "minted" | "metadataUri"> & {
    contentFiles?: ContentFile[];
    minted?: boolean;
    metadataUri?: string;
  },
): TokenBlueprint {
  const minted: boolean = typeof input.minted === "boolean" ? input.minted : false;
  const metadataUri: string = input.metadataUri != null ? String(input.metadataUri).trim() : "";

  return {
    ...input,
    id: String((input as any).id ?? "").trim(),
    name: String((input as any).name ?? "").trim(),
    symbol: String((input as any).symbol ?? "").trim(),
    brandId: String((input as any).brandId ?? "").trim(),
    companyId: String((input as any).companyId ?? "").trim(),
    description: String((input as any).description ?? "").trim(),
    assigneeId: String((input as any).assigneeId ?? "").trim(),
    createdBy: String((input as any).createdBy ?? "").trim(),
    createdAt: String((input as any).createdAt ?? "").trim(),
    updatedBy: String((input as any).updatedBy ?? "").trim(),
    updatedAt: String((input as any).updatedAt ?? "").trim(),
    minted,
    metadataUri,
    contentFiles: normalizeContentFiles(input.contentFiles ?? []),
  };
}
