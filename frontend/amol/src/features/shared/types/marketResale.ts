// frontend/amol/src/features/shared/types/marketResale.ts

import type {
  PageResult,
} from "../pageResult";
import type {
  ResaleCondition,
  ResaleListingBase,
} from "./resale";

export type MarketResaleListing =
  ResaleListingBase & {
    imageUrl?: string;
    tokenIcon?: string;
    avatarName?: string;
    avatarIcon?: string;
    createdBy?: string;
    createdAt?: string;
    updatedBy?: string | null;
    updatedAt?: string | null;
  };

export type MarketResaleListResponse =
  PageResult<MarketResaleListing>;

export type MarketResaleDetailResponse = {
  data: MarketResaleListing;
};

export type MarketResaleSortField =
  | "createdAt"
  | "price"
  | "productName"
  | "brandName"
  | "tokenName";

export type MarketResaleSortOrder =
  | "asc"
  | "desc";

export type FetchMarketResalesParams = {
  page?: number;
  perPage?: number;
  q?: string;
  ids?: string[];
  mintAddresses?: string[];
  tokenBlueprintIds?: string[];
  productIds?: string[];
  brandIds?: string[];
  productBlueprintIds?: string[];
  viewerAvatarIds?: string[];
  conditions?: ResaleCondition[];
  minPrice?: number;
  maxPrice?: number;
  sort?: MarketResaleSortField;
  order?: MarketResaleSortOrder;
};