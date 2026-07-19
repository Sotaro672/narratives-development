// frontend/shell/src/shared/types/order.ts

/**
 * backend/internal/domain/order/entity.go のOrderItemTypeに対応。
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

  // list item identifiers
  modelId?: string;
  inventoryId?: string;
  listId?: string;

  // resale item identifier
  resaleId?: string;

  // product identifiers
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
 * backend/internal/domain/order/entity.go のOrderに対応。
 *
 * 日時はRFC3339形式の文字列を使用する。
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

export const MIN_ITEMS_REQUIRED = 1;

export function isOrderItemType(value: string): value is OrderItemType {
  return value === "list" || value === "resale";
}

/**
 * backendの公開Validateと同じ不変条件をフロント側でも検証する。
 */
export function validateOrder(order: Order): boolean {
  if (!isNonEmptyString(order.id)) return false;
  if (!isNonEmptyString(order.userId)) return false;
  if (!isNonEmptyString(order.avatarId)) return false;
  if (!isNonEmptyString(order.cartId)) return false;

  if (!validateShippingSnapshot(order.shippingSnapshot)) {
    return false;
  }

  if (!validatePaymentMethodSnapshot(order.paymentMethodSnapshot)) {
    return false;
  }

  if (typeof order.paid !== "boolean") {
    return false;
  }

  if (
    !Array.isArray(order.items) ||
    order.items.length < MIN_ITEMS_REQUIRED
  ) {
    return false;
  }

  for (const item of order.items) {
    if (!validateOrderItemSnapshot(item)) {
      return false;
    }
  }

  return parseRFC3339(order.createdAt) !== null;
}

export function validateShippingSnapshot(
  snapshot: ShippingSnapshot,
): boolean {
  if (!snapshot || typeof snapshot !== "object") {
    return false;
  }

  return (
    isNonEmptyString(snapshot.state) &&
    isNonEmptyString(snapshot.city) &&
    isNonEmptyString(snapshot.street) &&
    isNonEmptyString(snapshot.country)
  );
}

export function validatePaymentMethodSnapshot(
  snapshot: PaymentMethodSnapshot,
): boolean {
  if (!snapshot || typeof snapshot !== "object") {
    return false;
  }

  if (!isNonEmptyString(snapshot.customerId)) return false;
  if (!isNonEmptyString(snapshot.brand)) return false;
  if (!isNonEmptyString(snapshot.last4)) return false;
  if (!isNonEmptyString(snapshot.cardholderName)) return false;

  if (
    !Number.isInteger(snapshot.expMonth) ||
    snapshot.expMonth < 1 ||
    snapshot.expMonth > 12
  ) {
    return false;
  }

  if (
    !Number.isInteger(snapshot.expYear) ||
    snapshot.expYear < 2000 ||
    snapshot.expYear > 9999
  ) {
    return false;
  }

  return typeof snapshot.isDefault === "boolean";
}

export function validateOrderItemSnapshot(
  item: OrderItemSnapshot,
): boolean {
  if (!item || typeof item !== "object") {
    return false;
  }

  if (!isOrderItemType(item.type)) {
    return false;
  }

  if (!Number.isInteger(item.qty)) {
    return false;
  }

  if (!Number.isInteger(item.price) || item.price < 0) {
    return false;
  }

  if (typeof item.isCanceled !== "boolean") {
    return false;
  }

  if (typeof item.isDispatched !== "boolean") {
    return false;
  }

  if (typeof item.transferred !== "boolean") {
    return false;
  }

  if (!validateTransferredState(item)) {
    return false;
  }

  switch (item.type) {
    case "list":
      return validateListItemSnapshot(item);

    case "resale":
      return validateResaleItemSnapshot(item);
  }
}

function validateListItemSnapshot(
  item: OrderItemSnapshot,
): boolean {
  return (
    isNonEmptyString(item.modelId) &&
    isNonEmptyString(item.inventoryId) &&
    isNonEmptyString(item.listId) &&
    item.qty > 0
  );
}

function validateResaleItemSnapshot(
  item: OrderItemSnapshot,
): boolean {
  return (
    isNonEmptyString(item.resaleId) &&
    isNonEmptyString(item.productId) &&
    isNonEmptyString(item.productBlueprintId) &&
    isNonEmptyString(item.tokenBlueprintId) &&
    isNonEmptyString(item.brandId) &&
    item.qty === 1
  );
}

function validateTransferredState(
  item: OrderItemSnapshot,
): boolean {
  if (item.transferred) {
    return parseRFC3339(item.transferredAt) !== null;
  }

  return item.transferredAt == null;
}

function isNonEmptyString(
  value: string | null | undefined,
): value is string {
  return typeof value === "string" && value.trim() !== "";
}

function parseRFC3339(
  value: string | null | undefined,
): Date | null {
  if (!isNonEmptyString(value)) {
    return null;
  }

  const timestamp = Date.parse(value);
  if (Number.isNaN(timestamp)) {
    return null;
  }

  return new Date(timestamp);
}