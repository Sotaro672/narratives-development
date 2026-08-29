// frontend/amol/src/features/like/types.ts

import type { PageResult } from "../shared/pageResult";

export type LikePriceRow = {
  currency?: string;
  amount?: number;
  price?: number;
  [key: string]: unknown;
};

export type LikeListItem = {
  id: string;
  title?: string;
  description?: string;
  image?: string;
  imageUrl?: string;
  price?: number;
  prices?: LikePriceRow[];
  listId?: string;
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  productId?: string;
  brandId?: string;
  brandName?: string;
  productName?: string;
  likedAt?: string;
  createdAt?: string;
  updatedAt?: string;
};

export type LikeIndexResponse = PageResult<LikeListItem>;

export type LikeCatalogProductBlueprint = {
  id?: string;
  productName?: string;
  brandName?: string;
};

export type LikeCatalogResponse = {
  productBlueprint?: LikeCatalogProductBlueprint;
};

export type LikeCardItem = LikeListItem & {
  productName?: string;
  brandName?: string;
};