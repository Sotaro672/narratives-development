// frontend/console/shell/src/shared/types/order.ts

/**
 * Backend BFFのOrderItemTypeに対応。
 */
export type OrderItemType = "list" | "resale";

export interface ShippingSnapshot {
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
  country: string;
}

export interface PaymentMethodSnapshot {
  customerId: string;
  brand: string;
  last4: string;
  expMonth: number;
  expYear: number;
  cardholderName: string;
  isDefault: boolean;
}

export interface OrderItemSnapshot {
  type: OrderItemType;

  // list
  modelId?: string;
  inventoryId?: string;
  listId?: string;

  // resale
  resaleId?: string;

  // product
  productId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  brandId?: string;

  qty: number;
  price: number;

  isCanceled: boolean;
  isDispatched: boolean;

  transferred: boolean;
  transferredAt?: string | null;
}

/**
 * Backend BFFのOrder responseを正とする。
 * 日時はBackendから返される文字列をそのまま保持する。
 */
export interface Order {
  id: string;
  userId: string;
  avatarId: string;
  cartId: string;

  shippingSnapshot: ShippingSnapshot;
  paymentMethodSnapshot: PaymentMethodSnapshot;

  paid: boolean;

  items: OrderItemSnapshot[];
  createdAt: string;
}