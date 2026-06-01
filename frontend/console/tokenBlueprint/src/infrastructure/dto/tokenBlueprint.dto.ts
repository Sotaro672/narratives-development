// frontend/console/tokenBlueprint/src/infrastructure/dto/tokenBlueprint.dto.ts

// backend(entity.go)を正にしたときの contentFiles の形（embedded object）
//
// backend/internal/domain/tokenBlueprint/entity.go
// type ContentFile struct {
//   ID          string            `json:"id"`
//   Type        ContentFileType   `json:"type"`
//   ContentType string            `json:"contentType,omitempty"`
//   URL         string            `json:"url"`
//   Visibility  ContentVisibility `json:"visibility"`
//   CreatedAt   time.Time         `json:"createdAt"`
//   CreatedBy   string            `json:"createdBy"`
//   UpdatedAt   time.Time         `json:"updatedAt"`
//   UpdatedBy   string            `json:"updatedBy"`
// }
//
// objectPath / name / size は廃止済み。
export type ContentVisibilityDTO = "private" | "public" | string;
export type ContentFileTypeDTO = "image" | "video" | "pdf" | "document" | string;

export type ContentFileDTO = {
  id: string;
  type: ContentFileTypeDTO;
  contentType?: string;
  url: string;
  visibility: ContentVisibilityDTO;

  createdAt?: string;
  createdBy?: string;
  updatedAt?: string;
  updatedBy?: string;
};

// TokenBlueprint のHTTPレスポンス形（domain(shared)ではない）
//
// backend/internal/domain/tokenBlueprint/entity.go
// type TokenBlueprint struct {
//   ID           string        `json:"id"`
//   Name         string        `json:"name"`
//   Symbol       string        `json:"symbol"`
//   BrandID      string        `json:"brandId"`
//   CompanyID    string        `json:"companyId"`
//   Description  string        `json:"description,omitempty"`
//   IconURL      string        `json:"iconUrl,omitempty"`
//   ContentFiles []ContentFile `json:"contentFiles"`
//   AssigneeID   string        `json:"assigneeId"`
//   Minted       bool          `json:"minted"`
//   CreatedAt    time.Time     `json:"createdAt"`
//   CreatedBy    string        `json:"createdBy"`
//   UpdatedAt    time.Time     `json:"updatedAt"`
//   UpdatedBy    string        `json:"updatedBy"`
//   DeletedAt    *time.Time    `json:"deletedAt,omitempty"`
//   DeletedBy    *string       `json:"deletedBy,omitempty"`
//   MetadataURI  string        `json:"metadataUri,omitempty"`
// }
//
// brandName / assigneeName / createdByName / updatedByName は、
// backend handler / resolver が画面表示用に付与する補助フィールドとして optional で扱う。
export type TokenBlueprintDTO = {
  id: string;
  name: string;
  symbol: string;

  brandId: string;
  brandName?: string;
  companyId: string;

  description?: string;

  iconUrl?: string | null;

  contentFiles: ContentFileDTO[];

  assigneeId: string;
  assigneeName?: string;

  minted: boolean;

  createdAt?: string;
  createdBy?: string;
  createdByName?: string;

  updatedAt?: string;
  updatedBy?: string;
  updatedByName?: string;

  deletedAt?: string | null;
  deletedBy?: string | null;

  metadataUri?: string;
};

export type TokenBlueprintPageResultDTO = {
  items: TokenBlueprintDTO[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};