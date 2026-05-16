// frontend/amol/src/features/catalog/types.ts

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

export type CatalogQualityAssuranceItem = {
  label?: string;
  title?: string;
  value?: string;
  [key: string]: unknown;
};

export type CatalogProductBlueprint = {
  id: string;
  productName: string;
  brandId: string;
  companyId: string;
  brandName: string;
  companyName: string;

  /**
   * apparel では category / itemType 的に使う。
   * alcohol では空文字の場合がある。
   */
  itemType: string;

  /**
   * apparel 用。
   * alcohol では空文字または未使用。
   */
  fit: string;

  /**
   * apparel では素材、alcohol では原材料として扱う。
   */
  material: string;

  /**
   * apparel 用。
   * alcohol では返らない可能性があるため optional / nullable。
   */
  weight?: number | null;

  printed: boolean;

  /**
   * backend から null が返るケースがある。
   */
  qualityAssurance:
    | string[]
    | string
    | CatalogQualityAssuranceItem[]
    | null;

  productIdTagType: string;
  modelRefs: CatalogProductBlueprintModelRef[];

  /**
   * category / classification 系。
   * backend の返却揺れを許容する。
   */
  category?: string | null;
  categoryCode?: string | null;
  classification?: string | null;

  /**
   * alcohol 用。
   */
  region?: string | null;
  vintage?: string | number | null;
  alcoholContent?: string | number | null;
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

export type CatalogModelVariationKind =
  | "apparel"
  | "alcohol"
  | (string & {});

export type CatalogModelVariation = {
  id: string;
  productBlueprintId: string;

  /**
   * model variation kind.
   *
   * - apparel: size / colorName / colorRGB / measurements を使う
   * - alcohol: volumeValue / volumeUnit を使う
   */
  kind?: CatalogModelVariationKind | null;

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