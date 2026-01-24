// frontend/console/tokenBlueprint/src/infrastructure/dto/tokenBlueprint.dto.ts

export type SignedIconUploadDTO = {
  uploadUrl: string;
  objectPath: string;
  publicUrl: string;
  expiresAt?: string;
  contentType?: string;
};

// backend(entity.go)を正にしたときの contentFiles の形（embedded object）
export type ContentVisibilityDTO = "private" | "public" | string;
export type ContentFileTypeDTO = "image" | "video" | "pdf" | "document" | string;

export type ContentFileDTO = {
  id: string;
  name: string;
  type: ContentFileTypeDTO;
  contentType: string;
  size: number; // JSONは number で来ることが多い
  objectPath: string;
  visibility: ContentVisibilityDTO;

  createdAt?: string; // FirestoreのTimeはISO文字列で来る前提（実装に合わせて調整）
  createdBy?: string;
  updatedAt?: string;
  updatedBy?: string;

  // backend が返す閲覧用 URL（署名URL/プロキシURL/公開URL等）
  url?: string;
};

// TokenBlueprint のHTTPレスポンス形（domain(shared)ではない）
export type TokenBlueprintDTO = {
  id: string;
  name: string;
  symbol: string;

  brandId: string;
  companyId?: string;

  description: string;
  assigneeId?: string;

  minted: boolean;
  metadataUri?: string;

  // resolver が返す想定
  iconUrl?: string | null;

  // embedded
  contentFiles: ContentFileDTO[];

  // Create/Updateで返ることがある
  iconUpload?: SignedIconUploadDTO;

  // 必要なら来る補助
  brandName?: string;

  // ★ backend が解決して返す表示名
  assigneeName?: string;
  createdByName?: string;
  updatedByName?: string;

  // ★ 作成/更新情報（一覧で表示するために必要）
  createdAt?: string; // ISO8601 想定
  createdBy?: string; // memberId
  updatedAt?: string; // ISO8601 想定（未更新なら来ない場合あり）
  updatedBy?: string; // memberId（未更新なら来ない場合あり）
};

export type TokenBlueprintPageResultDTO = {
  items: TokenBlueprintDTO[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

// create/update payload は「送信用 DTO」として定義しても良いが、
// 推奨は repository 側で shared 型(ContentFile)→DTO を組み立てること（後述）
