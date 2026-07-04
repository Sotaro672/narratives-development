// frontend/amol/src/features/payment/utils/order.ts
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
  return cartItems
    .map((item): OrderItemSnapshot | null => {
      if (isResaleCartItem(item)) {
        const price = getItemPrice(item);

        return {
          type: "resale",
          resaleId: item.resaleId ?? "",
          productId: item.productId ?? "",
          productBlueprintId: item.productBlueprintId ?? "",
          tokenBlueprintId: item.tokenBlueprintId ?? "",
          brandId: item.brandId ?? "",
          price,
          qty: 1,
          isCanceled: false,
          isDispatched: false,
          transferred: false,
        };
      }

      const inventoryId = item.inventoryId ?? "";
      const listId = item.listId ?? "";
      const modelId = item.modelId ?? "";
      const price = getModelPrice(item.catalog, modelId) ?? getItemPrice(item);
      const qty = normalizeQty(item.qty);

      return {
        type: "list",
        inventoryId,
        isCanceled: false,
        isDispatched: false,
        listId,
        modelId,
        price,
        qty,
        transferred: false,
      };
    })
    .filter((item): item is OrderItemSnapshot => item !== null);
}

export function validateOrderItems(items: OrderItemSnapshot[]): string | null {
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
  item: Extract<OrderItemSnapshot, { type: "list" }>,
): string | null {
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

  return null;
}

function validateResaleOrderItem(
  item: Extract<OrderItemSnapshot, { type: "resale" }>,
): string | null {
  if (!item.resaleId) {
    return "リセール商品の resaleId を取得できませんでした。";
  }

  if (!item.productId) {
    return "リセール商品の productId を取得できませんでした。";
  }

  if (!item.productBlueprintId) {
    return "リセール商品の productBlueprintId を取得できませんでした。";
  }

  if (!item.tokenBlueprintId) {
    return "リセール商品の tokenBlueprintId を取得できませんでした。";
  }

  if (!item.brandId) {
    return "リセール商品の brandId を取得できませんでした。";
  }

  if (item.qty !== 1) {
    return "リセール商品の数量が不正です。";
  }

  if (item.price < 0) {
    return "リセール商品の価格が不正です。";
  }

  return null;
}

function isResaleCartItem(item: CanonicalCartDisplayItem): boolean {
  if (item.type === "resale") {
    return true;
  }

  return Boolean(item.resaleId || item.productId);
}

function getItemPrice(item: CanonicalCartDisplayItem): number {
  if (typeof item.price === "number" && Number.isFinite(item.price)) {
    return item.price;
  }

  return 0;
}

function normalizeQty(qty: number | undefined): number {
  if (typeof qty !== "number" || !Number.isFinite(qty) || qty <= 0) {
    return 1;
  }

  return Math.floor(qty);
}