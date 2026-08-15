// frontend/amol/src/features/resale/types/resaleTypes.ts

import type { PageResult } from "../../shared/pageResult";

import type {
  ResaleEditableStatus,
  ResaleListingBase,
} from "../../shared/types/resale";

type ResaleCreateRequiredFields = Pick<
  ResaleListingBase,
  | "assetId"
  | "tokenBlueprintId"
  | "productId"
  | "price"
  | "condition"
  | "description"
>;

type ResaleCreateOptionalFields = Pick<
  ResaleListingBase,
  | "brandId"
  | "productBlueprintId"
>;

type ResaleUpdateFields = Partial<
  Pick<
    ResaleListingBase,
    | "price"
    | "condition"
    | "description"
  >
>;

export type ResaleListing = ResaleListingBase & {
  createdBy: string;
  createdAt: string;
  updatedBy?: string | null;
  updatedAt?: string | null;
};

export type ResaleConditionImage = {
  id: string;
  resaleId: string;
  url: string;
  displayOrder: number;
  createdAt: string;
  createdBy?: string;
  updatedAt?: string | null;
  updatedBy?: string | null;
};

export type CreateResaleListingParams =
  ResaleCreateRequiredFields &
  ResaleCreateOptionalFields & {
    conditionImages: File[];
  };

export type CreateResaleListingRecordParams = Omit<
  CreateResaleListingParams,
  "conditionImages"
>;

export type UpdateResaleListingParams = {
  resaleId: string;
  status?: ResaleEditableStatus;
} & ResaleUpdateFields;

export type ListMyResaleListingsParams = {
  page?: number;
  perPage?: number;
};

export type ListMyResaleListingsResponse =
  PageResult<ResaleListing>;

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