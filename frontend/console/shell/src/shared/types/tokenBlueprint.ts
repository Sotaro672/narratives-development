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

  /**
   * backend の GET レスポンスで「閲覧用URL（署名URL/プロキシURL/公開URL等）」が返る場合がある。
   * shell/shared 型として保持しておくことで、UI 側で表示に利用できる。
   */
  url?: string;
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

  /** ★追加: 作成者表示名（backend で解決された任意のラベル） */
  createdByName?: string;

  /** ★追加: 更新者表示名（backend で解決された任意のラベル） */
  updatedByName?: string;

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
