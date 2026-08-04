// frontend/amol/src/features/cart/types/cart.ts

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
 * cart featureで必要な項目だけを定義します。
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

  kind?:
    | CartModelKind
    | null;

  modelNumber?: string;
  modelLabel?: string;

  size?:
    | string
    | null;

  colorName?:
    | string
    | null;

  colorRGB?:
    | number
    | null;

  volumeValue?:
    | number
    | null;

  volumeUnit?:
    | string
    | null;

  /**
   * APIレスポンスにモデル単位の価格が含まれる場合に使用します。
   * 通常はlist.pricesを参照します。
   */
  price?: number;
};

/**
 * カート画面で表示に使用するカタログ情報。
 *
 * catalog featureの型には依存しません。
 */
export type CartCatalogSnapshot = {
  list: CartListSnapshot;

  listImages:
    CartListImageSnapshot[];

  productBlueprint:
    CartProductSnapshot;

  modelVariations:
    CartModelSnapshot[];
};

/**
 * GET /mall/me/cart の各カート項目。
 */
export type CartItemDTO = {
  /**
   * list:
   * 通常販売商品
   *
   * resale:
   * 二次流通商品
   */
  type: CartItemType;

  /**
   * 通常販売商品の識別子。
   */
  inventoryId?: string;
  listId?: string;
  modelId?: string;

  /**
   * 二次流通商品の識別子。
   */
  resaleId?: string;

  /**
   * 商品関連の識別子。
   */
  productId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  brandId?: string;

  /**
   * 商品数量。
   *
   * resaleでは常に1として扱います。
   */
  qty: number;

  /**
   * カートレスポンスから直接表示に使用する情報。
   */
  title?: string;
  listImage?: string;
  imageUrl?: string;
  price?: number;
  brandName?: string;
  productName?: string;

  /**
   * apparel / alcohol共通のモデル情報。
   */
  modelKind?: CartModelKind;
  modelNumber?: string;
  modelLabel?: string;

  /**
   * apparel用の情報。
   */
  size?: string;
  color?: string;
  colorName?: string;
  colorRGB?: number;

  /**
   * alcohol用の情報。
   */
  volumeValue?: number;
  volumeUnit?: string;
};

/**
 * GET /mall/me/cart のレスポンス。
 *
 * itemsはitemKeyをキーとするmap形式です。
 */
export type CartDTO = {
  avatarId: string;

  items: Record<
    string,
    CartItemDTO
  >;

  createdAt?:
    | string
    | null;

  updatedAt?:
    | string
    | null;

  expiresAt?:
    | string
    | null;
};

/**
 * CartPageおよびPaymentPageで使用する表示用カート項目。
 */
export type CartDisplayItem = {
  /**
   * CartDTO.itemsのmapキー。
   */
  itemKey: string;

  avatarId: string;

  /**
   * list:
   * 通常販売商品
   *
   * resale:
   * 二次流通商品
   */
  type: CartItemType;

  /**
   * 通常販売商品の識別子。
   */
  inventoryId?: string;
  listId?: string;
  modelId?: string;

  /**
   * 二次流通商品の識別子。
   */
  resaleId?: string;

  /**
   * 商品関連の識別子。
   */
  productId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  brandId?: string;

  qty: number;

  /**
   * カートレスポンス由来の表示情報。
   */
  title?: string;
  listImage?: string;
  imageUrl?: string;
  price?: number;
  brandName?: string;
  productName?: string;

  /**
   * apparel / alcoholの表示切り替えに使用します。
   */
  modelKind?: CartModelKind;
  modelNumber?: string;
  modelLabel?: string;

  /**
   * apparel用の情報。
   *
   * colorはカートレスポンス、
   * colorNameとcolorRGBはカタログレスポンス由来の値です。
   */
  size?: string;
  color?: string;
  colorName?: string;
  colorRGB?: number;

  /**
   * alcohol用の情報。
   */
  volumeValue?: number;
  volumeUnit?: string;

  /**
   * 通常販売商品のカタログ情報。
   *
   * resale商品またはカタログ取得失敗時はnullです。
   */
  catalog:
    | CartCatalogSnapshot
    | null;
};