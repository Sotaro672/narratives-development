// frontend/amol/src/features/shared/types/cart.ts

export type CartModelKind = "apparel" | "alcohol" | "unknown";
export type CartItemType = "list" | "resale";

/**
 * GET /mall/me/cart の各カート項目。
 *
 * バックエンドの現行DTOを正とし、
 * 旧レスポンスとの互換項目は保持しない。
 */
export type CartItemDTO = {
  type: CartItemType;

  // list item identifiers
  inventoryId?: string;
  listId?: string;
  modelId?: string;

  // resale item identifiers
  resaleId?: string;

  // product identifiers
  productId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  brandId?: string;

  // 商品数量。resaleでは常に1。
  qty: number;

  // cart responseから直接表示に使用する商品情報
  title?: string;
  listImage?: string;
  imageUrl?: string;
  price?: number;
  brandName?: string;
  productName?: string;

  // apparel / alcohol共通のmodel情報
  modelKind?: CartModelKind;
  modelNumber?: string;
  modelLabel?: string;

  // apparel
  size?: string;
  color?: string;

  // alcohol
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
 *
 * 表示情報はGET /mall/me/cartを唯一の正とし、
 * frontendからcatalog情報を後付けしない。
 */
export type CartDisplayItem = CartItemDTO & {
  itemKey: string;
  avatarId: string;
};