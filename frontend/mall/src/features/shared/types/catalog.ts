// frontend/amol/src/features/shared/types/catalog.ts

import type {
  ProductBlueprintCategoryFields,
  ProductCategoryKind,
} from "./category";

export type CatalogListPrice = {
  modelId: string;
  price: number;
};

export type CatalogList = {
  id: string;
  title: string;
  description: string;
  image: string;
  prices: CatalogListPrice[];
  inventoryId: string;
};

export type CatalogListImage = {
  id: string;
  listId: string;
  url: string;
  objectPath: string;
  fileName: string;
  displayOrder: number;
  size: number;
};

export type CatalogInventoryStockItem = {
  accumulation: number;
  reservedCount: number;
};

export type CatalogInventory = {
  id: string;
  productBlueprintId: string;
  tokenBlueprintId: string;
  modelIds: string[];
  stock: Record<string, CatalogInventoryStockItem>;
};

export type CatalogProductBlueprintModelRef = {
  modelId: string;
  displayOrder: number;
};

export type CatalogProductBlueprint = {
  id: string;
  productName: string;
  brandId: string;
  companyId: string;
  brandName: string;
  companyName: string;
  printed: boolean;
  productIdTagType: string;
  productBlueprintCategoryPath?: string[] | null;

  /**
   * category ごとの productBlueprint 入力値。
   *
   * 例:
   * - alcohol.sake: material / region / vintage / alcoholContent
   * - apparel.tops: material / fit / weight
   *
   * category ごとに項目が変わるため、
   * 固定 field ではなく map として扱う。
   */
  categoryFields?: ProductBlueprintCategoryFields | null;
  modelRefs: CatalogProductBlueprintModelRef[];
};

export type CatalogTokenBlueprint = {
  id: string;
  tokenName: string;
  symbol: string;
  brandId: string;
  brandName: string;
  companyName: string;
  description: string;
  tokenIcon: string;
};

export type CatalogModelVariation = {
  id: string;
  productBlueprintId: string;

  /**
   * model variation kind.
   *
   * - apparel:
   *   size / colorName / colorRGB / measurements を使う
   * - alcohol:
   *   volumeValue / volumeUnit を使う
   */
  kind?: ProductCategoryKind | null;
  modelNumber: string;

  // apparel
  size?: string | null;
  colorName?: string | null;
  colorRGB?: number | null;
  measurements?: Record<string, number>;

  // alcohol
  volumeValue?: number | null;
  volumeUnit?: string | null;

  stockKeys: number;
};

export type CatalogProductReviewSummary = {
  productBlueprintId: string;
  status: string;
  totalCount: number;
  averageRating: number;
  rating5Count: number;
  rating4Count: number;
  rating3Count: number;
  rating2Count: number;
  rating1Count: number;
};

export type CatalogResponse = {
  list: CatalogList;
  listImages: CatalogListImage[];
  inventory: CatalogInventory;
  productBlueprint: CatalogProductBlueprint;
  tokenBlueprint: CatalogTokenBlueprint;
  modelVariations: CatalogModelVariation[];
  productReviewSummary: CatalogProductReviewSummary;
};

export type MeasurementTableRow = {
  id: string;
  size: string;
  measurements: Record<string, number>;
};

export type ModelColorOption = {
  key: string;
  colorName: string;
  colorRGB: number;
};