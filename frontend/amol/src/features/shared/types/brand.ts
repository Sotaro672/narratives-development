// frontend/amol/src/features/brand/types/brand.ts

export type ListPriceRow = {
  currency?: string;
  amount?: number;
  price?: number;
  [key: string]: unknown;
};

export type BrandListItem = {
  id: string;
  title: string;
  description: string;
  image: string;
  prices: ListPriceRow[];

  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
};

export type BrandDetail = {
  brandId: string;
  brandName: string;
  websiteUrl: string;
  brandIcon: string;
  brandBackgroundImage: string;
  description: string;
  companyId: string;
  companyName: string;
  inventoryIds: string[];
  listIds: string[];
};