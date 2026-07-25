// frontend/console/tokenBlueprint/src/infrastructure/dto/tokenBlueprint.dto.ts
export type ContentVisibilityDTO = "private" | "public";

export type ContentFileTypeDTO = "image" | "video" | "pdf" | "document";

export type ContentFileDTO = {
  id: string;
  name: string;
  type: ContentFileTypeDTO;
  contentType: string;
  url: string;
  objectPath: string;
  visibility: ContentVisibilityDTO;
  size: number;

  createdAt: string;
  createdBy: string;
  updatedAt: string;
  updatedBy: string;
};

export type TokenBlueprintDTO = {
  id: string;
  name: string;
  symbol: string;

  brandId: string;
  brandName?: string;
  companyId: string;

  description?: string;

  iconUrl?: string | null;
  iconObjectPath?: string | null;
  iconFileName?: string | null;
  iconContentType?: string | null;
  iconSize?: number | null;

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