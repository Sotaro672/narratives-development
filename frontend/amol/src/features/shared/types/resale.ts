// frontend/amol/src/features/shared/types/resale.ts

import type { PageResult } from "../pageResult";

export const RESALE_CONDITIONS = [
  "新品・未使用",
  "未使用に近い",
  "目立った傷や汚れなし",
  "やや傷や汚れあり",
  "傷や汚れあり",
] as const;

export type ResaleCondition = (typeof RESALE_CONDITIONS)[number];

export const DEFAULT_RESALE_CONDITION: ResaleCondition = "未使用に近い";

export const RESALE_CONDITION_OPTIONS = RESALE_CONDITIONS.map((condition) => ({
  value: condition,
  label: condition,
}));

export const RESALE_STATUSES = ["listing", "suspended", "sold"] as const;

export type ResaleStatus = (typeof RESALE_STATUSES)[number];

export type ResaleEditableStatus = Exclude<ResaleStatus, "sold">;

export type ResaleStatusOption = {
  value: ResaleEditableStatus;
  label: string;
};

export const RESALE_STATUS_OPTIONS: ResaleStatusOption[] = [
  { value: "listing", label: "出品中" },
  { value: "suspended", label: "公開停止" },
];

export const DEFAULT_RESALE_EDITABLE_STATUS: ResaleEditableStatus = "listing";

export type ResaleColor = {
  name?: string;
  rgb?: number;
};

export type ResaleVolume = {
  amount?: number;
  unit?: string;
};

export type ResaleListingBase = {
  id: string;
  status: ResaleStatus;
  assetId: string;
  tokenBlueprintId: string;
  productId: string;
  brandId?: string;
  productBlueprintId?: string;
  avatarId: string;
  price: number;
  condition: ResaleCondition;
  description: string;
  imageId?: string;
  productName?: string;
  tokenName?: string;
  tokenIcon?: string;
  brandName?: string;
  avatarName?: string;
  avatarIcon?: string;
  imageUrl?: string;
  modelId?: string;
  kind?: string;
  modelNumber?: string;
  size?: string;
  color?: ResaleColor;
  measurements?: Record<string, number>;
  volume?: ResaleVolume;
};

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
  "brandId" | "productBlueprintId"
>;

type ResaleUpdateFields = Partial<
  Pick<ResaleListingBase, "price" | "condition" | "description">
>;

export type CreateResaleListingParams = ResaleCreateRequiredFields &
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

export type ListMyResaleListingsResponse = PageResult<ResaleListing>;

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