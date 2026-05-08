//frontend\amol\src\features\catalog\types.ts
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
  itemType: string;
  fit: string;
  material: string;
  weight: number;
  printed: boolean;
  qualityAssurance: string[];
  productIdTagType: string;
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
  modelNumber: string;
  size: string;
  colorName: string;
  colorRGB: number;
  measurements: Record<string, number>;
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

export type CatalogProductBlueprintReview = {
  id: string;
  productBlueprintId: string;
  avatarId: string;
  rating: number;
  title: string;
  body: string;
  helpfulVotes: number;
  totalVotes: number;
  reviewedAt: string;
  status: string;
  avatarName: string;
  avatarIcon: string;
};

export type CatalogProductBlueprintReviewPage = {
  items: CatalogProductBlueprintReview[];
  page: number;
  perPage: number;
  total: number;
  hasNext: boolean;
};

export type MeAvatarStateResponse = {
  avatarId: string;
  followerCount?: number;
  followingCount?: number;
  postCount?: number;
  lastActiveAt?: string;
  updatedAt?: string;
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