//frontend\amol\src\features\market\types\marketResale.ts
import type {
  PageResult,
} from "../pageResult";
import type {
  ResaleCondition,
  ResaleListingBase,
  ResaleStatus,
} from "./resale";
import type {
  MarketResaleConditionImage,
} from "./marketResaleImage";

export type MarketResaleListing =
  ResaleListingBase & {
    imageUrl?: string;
    tokenIcon?: string;
    avatarName?: string;
    avatarIcon?: string;
    images?: MarketResaleConditionImage[];
    conditionImages?: MarketResaleConditionImage[];
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

export type MarketResaleSortOrder =
  | "asc"
  | "desc";

export type FetchMarketResalesParams = {
  page?: number;
  perPage?: number;
  q?: string;
  search?: string;
  searchQuery?: string;
  ids?: string[];
  mintAddresses?: string[];
  tokenBlueprintIds?: string[];
  productIds?: string[];
  brandIds?: string[];
  productBlueprintIds?: string[];
  avatarIds?: string[];
  avatarId?: string;
  viewerAvatarId?: string;
  viewerAvatarIds?: string[];
  status?: ResaleStatus;
  statuses?: ResaleStatus[];
  condition?: ResaleCondition;
  conditions?: ResaleCondition[];
  minPrice?: number;
  maxPrice?: number;
  sort?: string;
  sortBy?: string;
  orderBy?: string;
  order?: MarketResaleSortOrder;
  sortOrder?: MarketResaleSortOrder;
  direction?: MarketResaleSortOrder;
};