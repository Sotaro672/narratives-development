// frontend/amol/src/features/payment/utils/order.ts
import type {
  CanonicalCartDisplayItem,
  CanonicalShippingAddress,
  CreateOrderItemRequest,
  OrderShippingSnapshot,
  PaymentMethod,
} from "../types";

export function selectPrimaryPaymentMethod(
  methods: PaymentMethod[],
  defaultMethod: PaymentMethod | null,
): PaymentMethod | null {
  if (defaultMethod) {
    return defaultMethod;
  }

  return methods.find((method) => method.isDefault) ?? methods[0] ?? null;
}

export function buildShippingSnapshot(
  address: CanonicalShippingAddress,
): OrderShippingSnapshot {
  return {
    zipCode: address.zipCode,
    state: address.state,
    city: address.city,
    street: address.street,
    street2: address.street2,
    country: "JP",
  };
}

export function buildOrderItems(
  cartItems: CanonicalCartDisplayItem[],
): CreateOrderItemRequest[] {
  return cartItems.map((item): CreateOrderItemRequest => {
    if (item.type === "resale") {
      return {
        type: "resale",
        resaleId: item.resaleId ?? "",
        qty: 1,
        isCanceled: false,
        isDispatched: false,
      };
    }

    return {
      type: "list",
      listId: item.listId ?? "",
      modelId: item.modelId ?? "",
      qty: item.qty,
      isCanceled: false,
      isDispatched: false,
    };
  });
}

export function validateOrderItems(
  items: CreateOrderItemRequest[],
): string | null {
  if (items.length === 0) {
    return "注文対象の商品がありません。";
  }

  for (const item of items) {
    if (item.type === "resale") {
      const error = validateResaleOrderItem(item);
      if (error) {
        return error;
      }

      continue;
    }

    const error = validateListOrderItem(item);
    if (error) {
      return error;
    }
  }

  return null;
}

function validateListOrderItem(
  item: Extract<CreateOrderItemRequest, { type: "list" }>,
): string | null {
  if (!item.listId) {
    return "注文商品の listId を取得できませんでした。";
  }

  if (!item.modelId) {
    return "注文商品の modelId を取得できませんでした。";
  }

  if (item.qty <= 0) {
    return "注文商品の数量が不正です。";
  }

  return null;
}

function validateResaleOrderItem(
  item: Extract<CreateOrderItemRequest, { type: "resale" }>,
): string | null {
  if (!item.resaleId) {
    return "リセール商品の resaleId を取得できませんでした。";
  }

  if (item.qty !== 1) {
    return "リセール商品の数量が不正です。";
  }

  return null;
}