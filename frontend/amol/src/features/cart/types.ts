// frontend/amol/src/features/cart/types/types.ts

export type CartModelKind = "apparel" | "alcohol" | "unknown" | string;

export type CartItemType = "list" | "resale";

export type CartItemDTO = {
  avatarId?: string;

  /**
   * item type
   * - list: 通常販売 item
   * - resale: 二次流通 item
   *
   * 既存レスポンス互換のため optional。
   * 未指定で inventoryId/listId/modelId があれば list として扱う。
   */
  type?: CartItemType | string;

  /**
   * list item identifiers
   */
  inventoryId?: string;
  listId?: string;
  modelId?: string;

  /**
   * resale item identifiers
   */
  resaleId?: string;

  /**
   * product identifiers
   */
  productId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  brandId?: string;

  qty?: number;
  quantity?: number;
  itemKey?: string;

  /**
   * cart response から直接表示に使う商品情報。
   * catalog が取得できない場合も cart response だけで表示できるように保持する。
   */
  title?: string;
  listImage?: string;
  imageUrl?: string;
  price?: number;
  productName?: string;

  /**
   * apparel / alcohol 共通の model 情報。
   */
  modelKind?: CartModelKind;
  kind?: CartModelKind;
  modelNumber?: string;
  modelLabel?: string;

  /**
   * apparel 用。
   */
  size?: string;
  colorName?: string;
  color?: string;
  colorRGB?: number;

  /**
   * alcohol 用。
   */
  volumeValue?: number;
  volumeUnit?: string;

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

  /**
   * item type
   * - list: 通常販売 item
   * - resale: 二次流通 item
   *
   * 既存データ互換のため optional。
   */
  type?: CartItemType;

  /**
   * list item identifiers
   *
   * resale item では存在しないため optional。
   */
  inventoryId?: string;
  listId?: string;
  modelId?: string;

  /**
   * resale item identifiers
   */
  resaleId?: string;

  /**
   * product identifiers
   */
  productId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  brandId?: string;

  qty: number;

  /**
   * cart response 由来の表示用情報。
   * catalog が null の場合でも CartPage / PaymentPage で表示できるようにする。
   */
  title?: string;
  listImage?: string;
  imageUrl?: string;
  price?: number;
  productName?: string;

  /**
   * apparel / alcohol の表示切り替え用。
   *
   * resale item では存在しないことがあるため optional。
   */
  modelKind?: CartModelKind;
  modelNumber?: string;
  modelLabel?: string;

  /**
   * apparel 用。
   */
  size?: string;
  colorName?: string;
  color?: string;
  colorRGB?: number;

  /**
   * alcohol 用。
   */
  volumeValue?: number;
  volumeUnit?: string;

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

export type CatalogQualityAssurance = {
  title?: string;
  body?: string;
  label?: string;
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
  itemType: string;
  fit: string;
  material: string;
  weight?: number | null;
  printed: boolean;
  qualityAssurance: CatalogQualityAssurance[] | string[] | string | null;
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

export type CatalogModelKind = "apparel" | "alcohol" | "unknown";

export type CatalogModelVariation = {
  id: string;
  productBlueprintId: string;

  /**
   * apparel / alcohol を判定するための種別。
   * backend から未返却の古いデータもあり得るため optional にする。
   */
  kind?: CatalogModelKind | string;

  /**
   * apparel / alcohol 共通で使える型番。
   */
  modelNumber: string;

  /**
   * apparel 用。
   * alcohol では返らないため optional。
   */
  size?: string;
  colorName?: string;
  colorRGB?: number;
  measurements?: Record<string, number>;

  /**
   * alcohol 用。
   */
  volumeValue?: number;
  volumeUnit?: string;

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
  productReviewSummary?: CatalogProductReviewSummary;
};