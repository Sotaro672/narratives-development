// frontend/amol/src/features/resale/types/resaleTypes.ts

import type {
  PageResultResponse,
} from "../pageResult";

export type ResaleListing = {
  id?: string;
  status?: string;
  mintAddress?: string;
  tokenBlueprintId?: string;
  productId?: string;
  brandId?: string;
  productBlueprintId?: string;
  avatarId?: string;
  price?: number;
  condition?: string;
  description?: string;
  imageId?: string;
  productName?: string;
  tokenName?: string;
  brandName?: string;
  createdBy?: string;
  createdAt?: string;
  updatedBy?: string | null;
  updatedAt?: string | null;
};

export type ResaleConditionImage = {
  id: string;
  resaleId?: string;
  url: string;
  objectPath: string;
  fileName: string;
  fileSize: number;
  mimeType: string;
  displayOrder: number;
};

export type CreateResaleListingParams = {
  mintAddress: string;
  tokenBlueprintId: string;
  productId: string;
  brandId?: string;
  productBlueprintId?: string;
  price: number;
  condition: string;
  description: string;
  conditionImages: File[];
};

export type CreateResaleListingRecordParams =
  Omit<
    CreateResaleListingParams,
    "conditionImages"
  >;

export type UpdateResaleListingParams = {
  resaleId: string;
  price?: number;
  condition?: string;
  description?: string;
  status?: string;
};

export type ListMyResaleListingsParams = {
  page?: number;
  perPage?: number;
};

export type ListMyResaleListingsResponse =
  PageResultResponse<ResaleListing>;

export type ListResaleListingsByAvatarIdParams = {
  avatarId: string;
  page?: number;
  perPage?: number;
};

export type AddResaleConditionImagesParams = {
  resaleId: string;
  files: File[];
  startDisplayOrder?: number;
};

export type ResaleImageIdentifier = {
  resaleId: string;
  imageId: string;
};