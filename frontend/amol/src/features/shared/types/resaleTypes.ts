// frontend/amol/src/features/resale/types/resaleTypes.ts

import type {
  PageResultResponse,
} from "../pageResult";

import type {
  ResaleEditableStatus,
  ResaleListingBase,
} from "../../shared/types/resale";

type ResaleCreateRequiredFields =
  Required<
    Pick<
      ResaleListingBase,
      | "mintAddress"
      | "tokenBlueprintId"
      | "productId"
      | "price"
      | "condition"
      | "description"
    >
  >;

type ResaleCreateOptionalFields =
  Pick<
    ResaleListingBase,
    | "brandId"
    | "productBlueprintId"
  >;

type ResaleUpdateFields =
  Partial<
    Pick<
      ResaleListingBase,
      | "price"
      | "condition"
      | "description"
    >
  >;

export type ResaleListing =
  Partial<ResaleListingBase> & {
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

export type CreateResaleListingParams =
  ResaleCreateRequiredFields &
  ResaleCreateOptionalFields & {
    conditionImages: File[];
  };

export type CreateResaleListingRecordParams =
  Omit<
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