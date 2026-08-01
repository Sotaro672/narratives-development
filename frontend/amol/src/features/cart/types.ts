// frontend/amol/src/features/cart/types.ts

export type CartModelKind =
  | "apparel"
  | "alcohol"
  | "unknown";

export type CartItemType =
  | "list"
  | "resale";

/**
 * カート画面で使用する価格情報。
 *
 * catalog featureの型には依存せず、
 * cart featureで必要な項目だけを定義する。
 */
export type CartPrice = {
  modelId: string;
  price: number;
};

/**
 * カート画面で使用するリスト情報。
 */
export type CartListSnapshot = {
  id: string;
  title: string;
  image: string;
  prices: CartPrice[];
};

/**
 * カート画面で使用するリスト画像情報。
 */
export type CartListImageSnapshot = {
  url: string;
};

/**
 * カート画面で使用する商品設計情報。
 */
export type CartProductSnapshot = {
  productName: string;
  brandName: string;
};

/**
 * カート画面で使用するモデル情報。
 */
export type CartModelSnapshot = {
  id: string;

  kind?: CartModelKind | null;

  modelNumber?: string;
  modelLabel?: string;

  size?: string | null;
  colorName?: string | null;
  colorRGB?: number | null;

  volumeValue?: number | null;
  volumeUnit?: string | null;

  /**
   * APIレスポンスにモデル単位の価格が含まれる場合に使用する。
   * 通常はlist.pricesを参照する。
   */
  price?: number;
};

/**
 * カート画面で表示に使用するカタログ情報。
 *
 * features/catalog/types.tsには依存しない。
 */
export type CartCatalogSnapshot = {
  list: CartListSnapshot;
  listImages: CartListImageSnapshot[];
  productBlueprint: CartProductSnapshot;
  modelVariations: CartModelSnapshot[];
};

/**
 * GET /mall/me/cart の各カート項目。
 *
 * バックエンドの現行DTOを正とし、
 * 旧レスポンスとの互換項目は保持しない。
 */
export type CartItemDTO = {
  /**
   * item type
   * - list: 通常販売item
   * - resale: 二次流通item
   */
  type: CartItemType;

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

  /**
   * 商品数量。
   *
   * resaleでは常に1。
   */
  qty: number;

  /**
   * cart responseから直接表示に使用する商品情報。
   */
  title?: string;
  listImage?: string;
  imageUrl?: string;
  price?: number;
  brandName?: string;
  productName?: string;

  /**
   * apparel / alcohol共通のmodel情報。
   */
  modelKind?: CartModelKind;
  modelNumber?: string;
  modelLabel?: string;

  /**
   * apparel用。
   */
  size?: string;
  color?: string;

  /**
   * alcohol用。
   */
  volumeValue?: number;
  volumeUnit?: string;
};

/**
 * GET /mall/me/cart のレスポンス。
 *
 * itemsはitemKeyをキーとするmap形式。
 */
export type CartDTO = {
  avatarId: string;
  items: Record<string, CartItemDTO>;
  createdAt?: string | null;
  updatedAt?: string | null;
  expiresAt?: string | null;
};

/**
 * CartPage / PaymentPageで使用する表示用カート項目。
 */
export type CartDisplayItem = {
  /**
   * CartDTO.itemsのmapキー。
   */
  itemKey: string;

  avatarId: string;

  /**
   * item type
   * - list: 通常販売item
   * - resale: 二次流通item
   */
  type: CartItemType;

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

  qty: number;

  /**
   * cart response由来の表示情報。
   */
  title?: string;
  listImage?: string;
  imageUrl?: string;
  price?: number;
  brandName?: string;
  productName?: string;

  /**
   * apparel / alcoholの表示切り替えに使用する。
   */
  modelKind?: CartModelKind;
  modelNumber?: string;
  modelLabel?: string;

  /**
   * apparel用。
   *
   * colorはcart response、
   * colorNameとcolorRGBはcatalog response由来の値に使用する。
   */
  size?: string;
  color?: string;
  colorName?: string;
  colorRGB?: number;

  /**
   * alcohol用。
   */
  volumeValue?: number;
  volumeUnit?: string;

  /**
   * 通常販売itemのカタログ情報。
   *
   * resale itemまたはカタログ取得失敗時はnull。
   */
  catalog: CartCatalogSnapshot | null;
};