// frontend/amol/src/features/shared/types/resale.ts

export const RESALE_CONDITIONS = [
  "新品・未使用",
  "未使用に近い",
  "目立った傷や汚れなし",
  "やや傷や汚れあり",
  "傷や汚れあり",
] as const;

export type ResaleCondition =
  (typeof RESALE_CONDITIONS)[number];

export type ResaleStatus =
  | "listing"
  | "suspended"
  | "sold";

export type ResaleEditableStatus =
  Exclude<
    ResaleStatus,
    "sold"
  >;

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