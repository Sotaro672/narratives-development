//frontend\amol\src\features\payment\utils\order.ts
import { getModelPrice } from "../../cart/utils/cartUtils";
import type {
  CanonicalCartDisplayItem,
  CanonicalShippingAddress,
  OrderItemSnapshot,
  OrderPaymentMethodSnapshot,
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

export function buildPaymentMethodSnapshot(
  method: PaymentMethod,
): OrderPaymentMethodSnapshot {
  return {
    customerId: method.stripeCustomerId,
    brand: method.brand,
    last4: method.last4,
    expMonth: method.expMonth,
    expYear: method.expYear,
    cardholderName: method.cardholderName,
    isDefault: method.isDefault,
  };
}

export function buildOrderItems(
  cartItems: CanonicalCartDisplayItem[],
): OrderItemSnapshot[] {
  return cartItems.map((item) => {
    const price = getModelPrice(item.catalog, item.modelId);

    return {
      inventoryId: item.inventoryId,
      isCanceled: false,
      isDispatched: false,
      listId: item.listId,
      modelId: item.modelId,
      price: price ?? 0,
      qty: item.qty,
      transferred: false,
    };
  });
}

export function validateOrderItems(items: OrderItemSnapshot[]): string | null {
  if (items.length === 0) {
    return "注文対象の商品がありません。";
  }

  for (const item of items) {
    if (!item.inventoryId) {
      return "注文商品の inventoryId を取得できませんでした。";
    }

    if (!item.listId) {
      return "注文商品の listId を取得できませんでした。";
    }

    if (!item.modelId) {
      return "注文商品の modelId を取得できませんでした。";
    }

    if (!item.qty || item.qty <= 0) {
      return "注文商品の数量が不正です。";
    }

    if (item.price < 0) {
      return "注文商品の価格が不正です。";
    }
  }

  return null;
}