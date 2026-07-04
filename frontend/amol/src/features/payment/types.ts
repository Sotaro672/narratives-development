// frontend/amol/src/features/payment/types.ts
import type { CartDisplayItem } from "../cart/types";
import type { ShippingAddress } from "../shipping-address/types";

export type PaymentMethod = {
  id: string;
  userId: string;
  stripeCustomerId: string;
  stripePaymentMethodId: string;
  brand: string;
  last4: string;
  expMonth: number;
  expYear: number;
  cardholderName: string;
  isDefault: boolean;
  createdAt?: string;
  updatedAt?: string;
};

export type PaymentContext = {
  avatarId?: string;
  avatarUid?: string;
};

export type PaymentMethodListResponse = {
  data?: PaymentMethod[];
  error?: string;
};

export type PaymentMethodDefaultResponse = {
  data?: PaymentMethod | null;
  error?: string;
};

export type CreatedPayment = {
  id?: string;
  paymentId?: string;
  paymentMethodId?: string;
  stripeCustomerId?: string;
  stripePaymentMethodId?: string;
  stripePaymentIntentId?: string;
  amount?: number;
  status?: string;
  clientSecret?: string;
  requiresAction?: boolean;
  createdAt?: string;
};

export type CreatedOrder = {
  id?: string;
  userId?: string;
  avatarId?: string;
  cartId?: string;
  paid?: boolean;
  createdAt?: string;
};

export type CartItemType = "list" | "resale";

export type CanonicalCartDisplayItem = CartDisplayItem & {
  avatarId: string;

  /**
   * type:
   * - list: 通常販売 item
   * - resale: 二次流通 item
   *
   * 既存レスポンス互換のため optional。
   * 未指定で inventoryId/listId/modelId がある場合は list item として扱う。
   */
  type?: CartItemType;

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

  // display fields
  title?: string;
  productName?: string;
  listImage?: string;
  imageUrl?: string;

  price?: number;
  qty: number;
};

export type CanonicalShippingAddress = ShippingAddress & {
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
};

export type OrderShippingSnapshot = {
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
  country: "JP";
};

export type OrderPaymentMethodSnapshot = {
  customerId: string;
  brand: string;
  last4: string;
  expMonth: number;
  expYear: number;
  cardholderName: string;
  isDefault: boolean;
};

export type ListOrderItemSnapshot = {
  type: "list";

  inventoryId: string;
  listId: string;
  modelId: string;

  price: number;
  qty: number;

  isCanceled: false;
  isDispatched: false;
  transferred: false;
};

export type ResaleOrderItemSnapshot = {
  type: "resale";

  resaleId: string;
  productId: string;
  productBlueprintId: string;
  tokenBlueprintId: string;
  brandId: string;

  price: number;
  qty: 1;

  isCanceled: false;
  isDispatched: false;
  transferred: false;
};

export type OrderItemSnapshot = ListOrderItemSnapshot | ResaleOrderItemSnapshot;

export type CreateOrderRequest = {
  id: string;
  avatarId: string;
  cartId: string;
  shippingSnapshot: OrderShippingSnapshot;
  paymentMethodSnapshot: OrderPaymentMethodSnapshot;
  items: OrderItemSnapshot[];
};

export type CreatePaymentRequest = {
  paymentId: string;
  paymentMethodId: string;
  stripeCustomerId: string;
  stripePaymentMethodId: string;
  amount: number;
};