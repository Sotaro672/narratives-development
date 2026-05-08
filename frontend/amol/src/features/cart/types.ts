// frontend/amol/src/features/cart/types/types.ts

export type CartItemDTO = {
  avatarId?: string;
  inventoryId?: string;
  listId?: string;
  modelId?: string;
  qty?: number;
  quantity?: number;
  itemKey?: string;
  [key: string]: unknown;
};

export type CartDTO = {
  avatarId: string;
  items: Record<string, CartItemDTO> | CartItemDTO[];
  createdAt?: string | null;
  updatedAt?: string | null;
  expiresAt?: string | null;
};

export type CartDisplayItem = {
  itemKey: string;
  avatarId: string;
  inventoryId: string;
  listId: string;
  modelId: string;
  qty: number;
  catalog: CatalogResponse | null;
};

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

export type CatalogResponse = {
  list: CatalogList;
  listImages: CatalogListImage[];
  inventory: CatalogInventory;
  productBlueprint: CatalogProductBlueprint;
  tokenBlueprint: CatalogTokenBlueprint;
  modelVariations: CatalogModelVariation[];
};

export type MeAvatarStateResponse = {
  avatarId: string;
  followerCount?: number;
  followingCount?: number;
  postCount?: number;
  lastActiveAt?: string;
  updatedAt?: string;
};