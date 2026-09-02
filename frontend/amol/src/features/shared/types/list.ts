// frontend/amol/src/features/shared/types/list.ts

import type { PageResult } from "../pageResult";
import type { CatalogProductReviewSummary } from "./catalog";

export type ListPriceRow = {
  currency?: string;
  amount?: number;
  price?: number;
  [key: string]: unknown;
};

export type MallListItem = {
  id: string;
  title: string;
  description: string;
  image: string;
  prices: ListPriceRow[];
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
};

export type MallListIndexResponse = PageResult<MallListItem>;

export type CatalogProductBlueprint = {
  id?: string;
  productName?: string;
  brandName?: string;
};

export type MallCatalogResponse = {
  productBlueprint?: CatalogProductBlueprint;
  productReviewSummary?: CatalogProductReviewSummary;
};

export type MallListCardItem = MallListItem & {
  productName?: string;
  brandName?: string;
  reviewAverage?: number;
  reviewCount?: number;
};

export type LoadListPageResult = {
  items: MallListCardItem[];
  page: number;
  totalPages: number;
};